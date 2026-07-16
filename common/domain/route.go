package domain

import "github.com/issueye/icoo_proxy/common/constants"

type Route struct {
	Name             string
	UpstreamProtocol constants.Protocol
	Model            string
	DefaultMaxTokens int
	Source           string
	Provider         ProviderSnapshot
}
