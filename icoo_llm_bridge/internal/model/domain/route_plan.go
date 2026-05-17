package domain

import "icoo_llm_bridge/internal/constants"

type RoutePlan struct {
	DownstreamProtocol constants.Protocol
	RequestedModel     string
	Candidates         []RouteCandidate
}

func (p RoutePlan) First() (RouteCandidate, bool) {
	if len(p.Candidates) == 0 {
		return RouteCandidate{}, false
	}
	return p.Candidates[0], true
}

type RouteCandidate struct {
	Name             string
	UpstreamProtocol constants.Protocol
	Model            string
	DefaultMaxTokens int
	Source           string
	Priority         int
	Provider         ProviderSnapshot
	Endpoint         ProviderEndpointSnapshot
	Credential       ProviderCredentialSnapshot
}

func (c RouteCandidate) Route() Route {
	provider := c.Provider
	if c.Endpoint.BaseURL != "" {
		provider.BaseURL = c.Endpoint.BaseURL
	}
	if c.Credential.APIKey != "" {
		provider.APIKey = c.Credential.APIKey
	}
	return Route{
		Name:             c.Name,
		UpstreamProtocol: c.UpstreamProtocol,
		Model:            c.Model,
		DefaultMaxTokens: c.DefaultMaxTokens,
		Source:           c.Source,
		Provider:         provider,
	}
}

type ProviderEndpointSnapshot struct {
	ID         string
	ProviderID string
	BaseURL    string
	Priority   int
	Weight     int
	Enabled    bool
}

type ProviderCredentialSnapshot struct {
	ID         string
	ProviderID string
	APIKey     string
	Enabled    bool
}
