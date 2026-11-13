package environments

import (
	"github.com/ijaidev/boat-go/core"
	"github.com/ijaidev/boat-go/model"
)

// ListRequest filters GET /environments.
//
// Org is the organization id, name or "personal". Environment routes only honor the org query
// parameter; core.WithOrg does not apply.
type ListRequest struct {
	Org string
}

// CreateRequest is the body of POST /environments.
//
// Name is unique across the account or organization and may contain letters, digits, dot, dash and
// underscore, up to 64 characters.
type CreateRequest struct {
	Name string `json:"name"`

	// Org selects the organization through the org query parameter.
	Org string `json:"-"`
}

// RepositorySelection selects one repository for an environment version.
type RepositorySelection struct {
	// RepositoryID is the repository databaseId from ListRepos.
	RepositoryID string `json:"repositoryId"`
	// BaseBranch defaults to main on the server.
	BaseBranch *string `json:"baseBranch,omitempty"`
	// SetupScript runs when a sandbox starts from this version.
	SetupScript *string `json:"setupScript,omitempty"`
	// SetupBlocking makes the sandbox wait for SetupScript before it reports ready.
	SetupBlocking *bool `json:"setupBlocking,omitempty"`
}

// UpdateRequest is the body of PUT /environments/{environmentId}.
//
// Every set field is written into a new immutable version; omitted fields are left unchanged.
// CredentialProfileID has three states: nil omits it, core.Some("profile") sets it, and
// core.Null[string]() clears it to the owner's default profile. A non-nil SecretFiles or
// Repositories slice replaces that list in full, including an empty slice.
type UpdateRequest struct {
	Name                   *string                     `json:"name,omitempty"`
	IsDefault              *bool                       `json:"isDefault,omitempty"`
	SafeForThirdParties    *bool                       `json:"safeForThirdParties,omitempty"`
	PassGithub             *bool                       `json:"passGithub,omitempty"`
	PassSecrets            *bool                       `json:"passSecrets,omitempty"`
	PassSandboxCredentials *bool                       `json:"passSandboxCredentials,omitempty"`
	PassAgentsCredentials  *bool                       `json:"passAgentsCredentials,omitempty"`
	CredentialProfileID    *core.Optional[string]      `json:"credentialProfileId,omitempty"`
	ProxyAiCredentials     *bool                       `json:"proxyAiCredentials,omitempty"`
	Access                 *model.ResourceAccessUpdate `json:"access,omitempty"`
	EnvContents            *string                     `json:"envContents,omitempty"`
	SecretFiles            *[]model.SecretFile         `json:"secretFiles,omitempty"`
	Repositories           *[]RepositorySelection      `json:"repositories,omitempty"`

	// Org selects the organization through the org query parameter.
	Org string `json:"-"`
}

// UpgradeRequest is the body of POST /environments/{environmentId}/upgrade.
//
// An empty AgentIDs upgrades every active sandbox on an older version of the environment.
type UpgradeRequest struct {
	AgentIDs []string `json:"agentIds,omitempty"`

	// Org selects the organization through the org query parameter.
	Org string `json:"-"`
}

// SecretFileRequest is the body of PUT /environments/{environmentId}/secret-files.
//
// Contents is a secret. The SDK never logs it or adds it to error text.
type SecretFileRequest struct {
	// Path is relative to the sandbox workspace, for example .env or repo/.env.
	Path     string `json:"path"`
	Contents string `json:"contents"`

	// Org selects the organization through the org query parameter.
	Org string `json:"-"`
}

// AddRepoRequest is the body of POST /environments/{environmentId}/repos.
type AddRepoRequest struct {
	// RepositoryID is the repository databaseId from ListRepos.
	RepositoryID string `json:"repositoryId"`
	// BaseBranch defaults to main on the server.
	BaseBranch *string `json:"baseBranch,omitempty"`

	// Org selects the organization through the org query parameter.
	Org string `json:"-"`
}

// ListReposRequest filters GET /repos.
//
// Sync asks the server to refresh its repository list from GitHub before answering. Limit, Cursor
// and Sort belong to the /repos cursor pagination; Query is the case-insensitive name filter sent as
// q. Selected filters to selected or unselected repositories.
type ListReposRequest struct {
	Sync     *bool
	Limit    int
	Cursor   string
	Sort     string
	Query    string
	Selected *bool
}

// SelectRepoRequest is the body of POST /repos.
//
// Selecting an already-selected repository updates its base branch instead of returning a conflict.
type SelectRepoRequest struct {
	// RepositoryID is the repository databaseId from ListRepos.
	RepositoryID string `json:"repositoryId"`
	// BaseBranch defaults to main on the server.
	BaseBranch *string `json:"baseBranch,omitempty"`
}
