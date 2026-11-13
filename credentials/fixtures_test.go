package credentials_test

// profileJSON is a fake credential profile. Every value is synthetic.
const profileJSON = `{
	"id": "prof_example",
	"name": "work",
	"isDefault": false,
	"owner": {"id": "usr_example", "kind": "personal", "name": "example", "aiProxyEnabled": true},
	"canManage": true,
	"connected": ["openai-api-key"],
	"accounts": {"openai-api-key": [{"id": "acct_1", "label": "ending 1234", "active": true}]},
	"customApis": ["STRIPE_KEY"],
	"createdAt": "2026-10-01T12:00:00Z",
	"updatedAt": "2026-10-09T08:00:00Z"
}`

// accountsJSON is a fake account list. Labels are not secrets.
const accountsJSON = `{
	"ok": true,
	"type": "credential_account.list",
	"success": true,
	"profileId": "prof_example",
	"credential": "openai-api-key",
	"accounts": [
		{"id": "acct_1", "label": "ending 1234", "active": true},
		{"id": "acct_2", "label": "ending 5678", "active": false, "addedAt": "2026-10-02T00:00:00Z"}
	]
}`

// proxyJSON is a fake AI proxy settings response.
const proxyJSON = `{
	"ok": true,
	"type": "ai_proxy.info",
	"success": true,
	"accountId": "usr_example",
	"org": false,
	"enabled": true,
	"enabledAt": "2026-10-09T08:00:00Z",
	"pricing": {"dailyDollars": 1, "dollarsPer1000Requests": 1},
	"proxyUrl": "https://proxy.example.com"
}`

// usageJSON is a fake account AI usage response.
const usageJSON = `{
	"ok": true,
	"type": "ai_usage",
	"accountId": "usr_example",
	"profileId": null,
	"since": "2026-10-01",
	"sandboxes": [{
		"sandboxId": "bx_example",
		"name": "example",
		"blocked": false,
		"usage": [{
			"day": "2026-10-10",
			"credential": "openai-api-key",
			"model": "example-model",
			"requests": 2,
			"errors": 0,
			"inputTokens": 10,
			"outputTokens": 20,
			"cacheReadTokens": 0,
			"cacheWriteTokens": 0
		}],
		"totals": {"requests": 2, "errors": 0, "inputTokens": 10, "outputTokens": 20, "cacheReadTokens": 0, "cacheWriteTokens": 0},
		"dollars": 0.01
	}],
	"usage": [{
		"day": "2026-10-10",
		"credential": "openai-api-key",
		"model": "example-model",
		"requests": 2,
		"errors": 0,
		"inputTokens": 10,
		"outputTokens": 20,
		"cacheReadTokens": 0,
		"cacheWriteTokens": 0
	}],
	"totals": {"requests": 2, "errors": 0, "inputTokens": 10, "outputTokens": 20, "cacheReadTokens": 0, "cacheWriteTokens": 0},
	"dollars": 0.01
}`

// sandboxUsageJSON is a fake sandbox AI usage response.
const sandboxUsageJSON = `{
	"ok": true,
	"type": "sandbox.ai_usage",
	"sandboxId": "bx_example",
	"blocked": false,
	"proxied": true,
	"credentialProfileId": null,
	"since": "2026-09-11",
	"usage": [],
	"totals": {"requests": 0, "errors": 0, "inputTokens": 0, "outputTokens": 0, "cacheReadTokens": 0, "cacheWriteTokens": 0},
	"dollars": 0
}`

const accessJSON = `{"ok": true, "type": "sandbox.ai_access_updated", "success": true, "sandboxId": "bx_example", "blocked": true}`
