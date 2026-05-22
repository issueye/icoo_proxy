use crate::{
    model::{Provider, ProviderModel, ProviderModelSnapshot, ProviderSnapshot, Route, RoutingRule},
    repository::Repository,
};

#[derive(Clone)]
pub struct RouteResolver {
    repo: Repository,
}

impl RouteResolver {
    pub fn new(repo: Repository) -> Self {
        Self { repo }
    }

    pub fn resolve(&self, downstream: &str, requested_model: &str) -> anyhow::Result<Route> {
        let requested_model = requested_model.trim();
        let providers = self.load_providers()?;
        if let Some(result) = self.resolve_direct(&providers, requested_model)? {
            return Ok(result);
        }
        let mut rules = self.repo.list_enabled_rules()?;
        rules.sort_by_key(|r| r.priority);
        for rule in rules {
            if !rule_matches(&rule, downstream, requested_model) {
                continue;
            }
            return self.route_from_rule(&providers, &rule, requested_model);
        }
        if requested_model.is_empty() {
            anyhow::bail!("no route matched downstream protocol {:?}", downstream);
        }
        anyhow::bail!(
            "no route matched downstream protocol {:?} and model {:?}",
            downstream,
            requested_model
        );
    }

    fn load_providers(&self) -> anyhow::Result<Vec<ProviderSnapshot>> {
        let mut snapshots = Vec::new();
        for provider in self.repo.list_providers()? {
            if !provider.enabled {
                continue;
            }
            let models = self.repo.list_models_by_provider(&provider.id)?;
            snapshots.push(provider_snapshot(provider, models));
        }
        Ok(snapshots)
    }

    fn resolve_direct(
        &self,
        providers: &[ProviderSnapshot],
        requested_model: &str,
    ) -> anyhow::Result<Option<Route>> {
        let Some((provider_name, model_name)) = requested_model.split_once('/') else {
            return Ok(None);
        };
        if provider_name.trim().is_empty() || model_name.trim().is_empty() {
            return Ok(None);
        }
        let provider = find_provider(providers, provider_name).ok_or_else(|| {
            anyhow::anyhow!(
                "direct route provider {:?} was not found or is disabled",
                provider_name
            )
        })?;
        let model = find_model(&provider.models, model_name).ok_or_else(|| {
            anyhow::anyhow!(
                "direct route model {:?} was not found or is disabled for provider {:?}",
                model_name,
                provider_name
            )
        })?;
        Ok(Some(build_route(
            &format!("{}/{}", provider.name, model.name),
            provider,
            &provider.protocol,
            &model.name,
            model.max_tokens,
            "direct",
            0,
        )))
    }

    fn route_from_rule(
        &self,
        providers: &[ProviderSnapshot],
        rule: &RoutingRule,
        requested_model: &str,
    ) -> anyhow::Result<Route> {
        let provider = find_provider(providers, &rule.target_provider_id).ok_or_else(|| {
            anyhow::anyhow!(
                "routing rule {:?} targets missing or disabled provider {:?}",
                rule.name,
                rule.target_provider_id
            )
        })?;
        let target_model = if rule.target_model.trim().is_empty() {
            requested_model.trim()
        } else {
            rule.target_model.trim()
        };
        if target_model.is_empty() {
            anyhow::bail!(
                "routing rule {:?} did not specify a target model",
                rule.name
            );
        }
        let model = find_model(&provider.models, target_model).ok_or_else(|| {
            anyhow::anyhow!(
                "routing rule {:?} targets missing or disabled model {:?} for provider {:?}",
                rule.name,
                target_model,
                provider.name
            )
        })?;
        let upstream = if rule.upstream_protocol.trim().is_empty() {
            provider.protocol.clone()
        } else {
            rule.upstream_protocol.clone()
        };
        Ok(build_route(
            &rule.name,
            provider,
            &upstream,
            &model.name,
            model.max_tokens,
            &format!("routing_rule:{}", rule.id),
            rule.priority,
        ))
    }
}

fn provider_snapshot(provider: Provider, models: Vec<ProviderModel>) -> ProviderSnapshot {
    ProviderSnapshot {
        id: provider.id,
        name: provider.name,
        protocol: provider.protocol,
        vendor: provider.vendor,
        base_url: provider.base_url,
        api_key: provider.api_key_cipher,
        only_stream: provider.only_stream,
        user_agent: provider.user_agent,
        enabled: provider.enabled,
        description: provider.description,
        models: models
            .into_iter()
            .filter(|m| m.enabled)
            .map(|m| ProviderModelSnapshot {
                name: m.name,
                max_tokens: m.max_tokens,
                enabled: m.enabled,
            })
            .collect(),
    }
}

fn rule_matches(rule: &RoutingRule, downstream: &str, requested_model: &str) -> bool {
    if !rule.match_protocol.trim().is_empty() && rule.match_protocol != downstream {
        return false;
    }
    model_pattern_matches(&rule.match_model_pattern, requested_model)
}

fn model_pattern_matches(pattern: &str, model: &str) -> bool {
    let pattern = pattern.trim();
    let model = model.trim();
    if pattern.is_empty() {
        return model.is_empty();
    }
    if pattern == "*" {
        return true;
    }
    if !pattern.contains('*') && !pattern.contains('?') && !pattern.contains('[') {
        return pattern == model;
    }
    glob_match(pattern, model)
}

fn glob_match(pattern: &str, text: &str) -> bool {
    fn inner(p: &[u8], t: &[u8]) -> bool {
        if p.is_empty() {
            return t.is_empty();
        }
        match p[0] {
            b'*' => inner(&p[1..], t) || (!t.is_empty() && inner(p, &t[1..])),
            b'?' => !t.is_empty() && inner(&p[1..], &t[1..]),
            c => !t.is_empty() && c == t[0] && inner(&p[1..], &t[1..]),
        }
    }
    inner(pattern.as_bytes(), text.as_bytes())
}

fn find_provider<'a>(providers: &'a [ProviderSnapshot], key: &str) -> Option<&'a ProviderSnapshot> {
    let key = key.trim();
    providers.iter().find(|p| p.id == key || p.name == key)
}

fn find_model<'a>(
    models: &'a [ProviderModelSnapshot],
    name: &str,
) -> Option<&'a ProviderModelSnapshot> {
    let name = name.trim();
    models.iter().find(|m| m.name == name)
}

fn build_route(
    name: &str,
    provider: &ProviderSnapshot,
    upstream: &str,
    model: &str,
    max_tokens: i64,
    source: &str,
    priority: i64,
) -> Route {
    Route {
        name: name.to_string(),
        upstream_protocol: upstream.to_string(),
        model: model.to_string(),
        default_max_tokens: max_tokens,
        source: source.to_string(),
        priority,
        provider: provider.clone(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{config, db::Database, model, repository::Repository};

    fn repo() -> Repository {
        let dir = tempfile::tempdir().unwrap();
        let mut cfg = config::defaults();
        cfg.apply_data_dir(dir.path().to_str().unwrap());
        let db = Database::open(&cfg).unwrap();
        db.seed_defaults().unwrap();
        std::mem::forget(dir);
        Repository::new(db)
    }

    #[test]
    fn direct_route_wins_before_rules() {
        let repo = repo();
        let now = model::now_string();
        repo.save_provider(&model::Provider {
            id: "p-openai".into(),
            name: "openai".into(),
            protocol: model::PROTOCOL_OPENAI_CHAT.into(),
            vendor: "custom".into(),
            enabled: true,
            created_at: now.clone(),
            updated_at: now.clone(),
            ..Default::default()
        })
        .unwrap();
        repo.save_model(&model::ProviderModel {
            id: "m1".into(),
            provider_id: "p-openai".into(),
            name: "gpt-4o".into(),
            max_tokens: 128000,
            enabled: true,
            created_at: now.clone(),
            updated_at: now.clone(),
        })
        .unwrap();
        repo.save_rule(&model::RoutingRule {
            id: "r1".into(),
            name: "catch all".into(),
            priority: 1,
            match_protocol: model::PROTOCOL_OPENAI_CHAT.into(),
            match_model_pattern: "*".into(),
            target_provider_id: "missing".into(),
            target_model: "missing".into(),
            enabled: true,
            created_at: now.clone(),
            updated_at: now,
            ..Default::default()
        })
        .unwrap();

        let route = RouteResolver::new(repo)
            .resolve(model::PROTOCOL_OPENAI_CHAT, "openai/gpt-4o")
            .unwrap();
        assert_eq!(route.source, "direct");
        assert_eq!(route.model, "gpt-4o");
        assert_eq!(route.default_max_tokens, 128000);
    }
}
