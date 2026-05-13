package domain

import "icoo_llm_bridge/internal/constants"

type Route struct {
	Name             string
	UpstreamProtocol constants.Protocol
	Model            string
	DefaultMaxTokens int
	Source           string
	Provider         ProviderSnapshot
}
