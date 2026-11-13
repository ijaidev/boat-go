package credentials

import "github.com/ijaidev/boat-go/model"

// CreateProfileRequest is the body of POST /credential-profiles.
//
// Name is required and must be unique among the owner's profiles. Org makes the profile belong to
// that organization; leave it empty for a personal profile. CopyFrom starts the new profile with a
// copy of that profile's credentials; leave it empty for an empty profile.
type CreateProfileRequest struct {
	Name     string `json:"name"`
	Org      string `json:"org,omitempty"`
	CopyFrom string `json:"copyFrom,omitempty"`
}

// UpdateProfileRequest is the body of PUT /credential-profiles/{profile}.
//
// IsDefault sends true to make this profile its owner's default. The API ignores false; make another
// profile the default instead. Access applies to organization profiles only and only its owner may
// change it.
type UpdateProfileRequest struct {
	Name      string                      `json:"name,omitempty"`
	IsDefault *bool                       `json:"isDefault,omitempty"`
	Access    *model.ResourceAccessUpdate `json:"access,omitempty"`
}

// SetKeyRequest is the body of PUT /credential-profiles/{profile}/keys/{credential} and
// POST /credential-profiles/{profile}/accounts/{credential}.
//
// Value is the provider API key and is write-only: the API never returns it and the SDK never logs
// it. KeepCurrent keeps the key being replaced as another account, ready to switch back to.
type SetKeyRequest struct {
	Value       string `json:"value"`
	KeepCurrent *bool  `json:"keepCurrent,omitempty"`
}

// SetCustomAPIRequest is the body of PUT /credential-profiles/{profile}/custom-apis/{name}.
//
// Value is the secret the sandbox receives as the variable named in the path. BaseURL is the public
// HTTPS base URL the sandbox receives as <NAME>_BASE_URL. Both are write-only in the API.
type SetCustomAPIRequest struct {
	Value   string `json:"value"`
	BaseURL string `json:"baseUrl"`
}

// GetProxyRequest filters GET /ai-proxy.
//
// Org selects the billing wallet: an organization id, an organization name, or "personal". It is
// sent as the org query parameter and wins over the X-Boat-Org header set by core.WithOrg.
type GetProxyRequest struct {
	Org string
}

// SetProxyRequest is the body and query of PUT /ai-proxy.
//
// Enabled turns the AI credentials proxy on or off for the account in scope. Org selects the billing
// wallet, as in GetProxyRequest; it is a query parameter and is not part of the JSON body.
//
// Turning the proxy on makes Boat charge the wallet for each UTC day the proxy is on and per proxied
// request, and replaces real provider keys in sandboxes with placeholders. Treat this as a
// security-sensitive and billable change.
type SetProxyRequest struct {
	Enabled bool   `json:"enabled"`
	Org     string `json:"-"`
}

// UsageRequest filters GET /ai-usage.
//
// Days is the number of UTC days of history, today included. Zero omits the parameter and uses the
// server default of 30; any other value is forwarded and must be from 1 to 365. Profile counts that
// profile's requests instead of a wallet's; only the profile's managers can read it. Org selects the
// billing wallet when Profile is empty.
type UsageRequest struct {
	Days    int
	Profile string
	Org     string
}

// SandboxUsageRequest filters GET /sandboxes/{sandboxId}/ai-usage.
//
// Days is the number of UTC days of history, today included. Zero omits the parameter and uses the
// server default of 30; any other value is forwarded and must be from 1 to 365.
type SandboxUsageRequest struct {
	Days int
}

// SetSandboxAccessRequest is the body of PUT /sandboxes/{sandboxId}/ai-access.
//
// Blocked true makes the AI proxy refuse the sandbox's placeholders. It has no effect on a sandbox
// that holds real provider keys.
type SetSandboxAccessRequest struct {
	Blocked bool `json:"blocked"`
}
