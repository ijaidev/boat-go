# Examples

Each example is a complete `main` package and is compiled by `go build ./examples/...`. They make
real API calls only when run.

| Example | What it shows | Environment |
|---|---|---|
| [quickstart](quickstart/main.go) | Create, wait, run one command, delete | `BOAT_API_KEY` |
| [stream](stream/main.go) | NDJSON command output and frames | `BOAT_API_KEY`, `BOAT_SANDBOX_ID` |
| [webhook](webhook/main.go) | Verify a lifecycle webhook before parsing it | `BOAT_WEBHOOK_SECRET` |
| [jobs](jobs/main.go) | Create, push, exec, usage and delete in one call | `BOAT_API_KEY` |

Run one with:

```bash
BOAT_API_KEY=boat_... go run ./examples/quickstart
```

Never commit a real API key, webhook secret, desktop URL or signed file URL.
