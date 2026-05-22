use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

#[derive(Clone, Default)]
pub struct RequestTracker {
    counts: Arc<Mutex<HashMap<String, i64>>>,
}

impl RequestTracker {
    pub fn acquire(&self, rule_id: &str) {
        let mut counts = self.counts.lock().unwrap();
        *counts.entry(rule_id.to_string()).or_default() += 1;
    }

    pub fn release(&self, rule_id: &str) {
        let mut counts = self.counts.lock().unwrap();
        if let Some(value) = counts.get_mut(rule_id) {
            *value -= 1;
            if *value <= 0 {
                counts.remove(rule_id);
            }
        }
    }

    pub fn active_count(&self, rule_id: &str) -> i64 {
        *self.counts.lock().unwrap().get(rule_id).unwrap_or(&0)
    }
}
