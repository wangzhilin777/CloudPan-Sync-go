# CloudPan-Sync-go

CloudPan Sync Go rebuild workspace.

## Current Status

- Phase 1 scaffold is in place.
- The app now has:
  - Go module bootstrap
  - config loading
  - structured logging
  - minimal HTTP server
  - embedded SQLite migration runner
  - provider registry for 10 target providers
  - rebuild plan document in `docs/01-GO_REBUILD_PLAN.md`

## Run

```powershell
go run ./cmd/cloudpan-sync
```

Default server address is `:8080`.
