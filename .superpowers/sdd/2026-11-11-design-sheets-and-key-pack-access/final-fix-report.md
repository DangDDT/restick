# Final backend review fixes

## Changes

- `services/api/internal/studio/sheets.go`: after a count-guarded INSERT affects zero rows, remove the upload's temporary source directory, repeat the `(design_id, checksum_sha256)` lookup, and return the concurrent winner. Return `ErrSheetLimit` only when that lookup finds no sheet; preserve real lookup errors instead of misreporting a limit.
- `services/api/internal/interfaces/http/admin_keys.go`: decode `pack_ids` into `*[]string` and reject missing/nil values with HTTP 400 `bad_request` (`pack_ids is required`) before calling `SetKeyPacks`. An explicit empty array still clears grants. Existing 404/422 mapping is unchanged.
- No caller signatures, dependencies, schema, migrations, production test hooks, or other packages changed. The existing key-packs OpenAPI PUT route already documents HTTP 400, so no contract change was necessary.

## Regression tests

- `TestSaveSheetConcurrentDuplicateAtLimit` in `internal/studio/sheets_concurrency_test.go`: seed 19 sheets, then run two identical new uploads against real SQLite. A test-only driver connector pauses both INSERTs until both initial duplicate lookups have missed, guaranteeing the reviewed interleaving rather than relying on scheduling. Assert both uploads succeed with the same nonempty sheet ID, exactly 20 design-scoped database rows, and exactly 20 source directories. No production hook or mock database is used.
- `TestAdminKeyPackGrantBodyRequired` in `internal/interfaces/http/admin_test.go`: separate cases reject `{}`, `{"pack_ids":null}`, and top-level `null` with HTTP 400 `bad_request`. Each begins with one grant and one pre-existing removal marker and verifies both grants and complete removal-marker values are unchanged. The explicit `{"pack_ids":[]}` case succeeds, clears grants, returns an empty array, and records the newly removed grant.
- Existing `TestAdminKeyPackGrants` continues covering unknown-key 404 and unknown-pack 422.

## Red/green evidence

Commands below ran from `services/api`.

Before production changes:

```sh
go test ./internal/studio ./internal/interfaces/http -run 'TestSaveSheetConcurrentDuplicateAtLimit|TestAdminKeyPackGrantBodyRequired' -count=1
```

Exit 1. Relevant exact output excerpts (random IDs omitted by selecting only these lines):

```text
--- FAIL: TestSaveSheetConcurrentDuplicateAtLimit (0.18s)
FAIL	github.com/eykh/restick/api_svc/internal/studio	0.734s
--- FAIL: TestAdminKeyPackGrantBodyRequired (0.83s)
    --- FAIL: TestAdminKeyPackGrantBodyRequired/missing_pack_ids (0.22s)
        admin_test.go:1611: PUT = 200 {"pack_ids":[]}
            , want 400
    --- FAIL: TestAdminKeyPackGrantBodyRequired/null_pack_ids (0.20s)
        admin_test.go:1611: PUT = 200 {"pack_ids":[]}
            , want 400
    --- FAIL: TestAdminKeyPackGrantBodyRequired/null_document (0.21s)
        admin_test.go:1611: PUT = 200 {"pack_ids":[]}
            , want 400
FAIL	github.com/eykh/restick/api_svc/internal/interfaces/http	1.389s
FAIL
```

The sheet failure reported one successful sheet ID and one `studio: design already has the maximum number of sheets` error. Each malformed-body case additionally failed the unchanged-grants/removals assertions, proving the original destructive behavior was exercised. The explicit-clear case passed before and after the fix.

After production changes, the same command exited 0:

```text
ok  	github.com/eykh/restick/api_svc/internal/studio	0.599s
ok  	github.com/eykh/restick/api_svc/internal/interfaces/http	1.272s
```

Formatting command (exit 0, no output):

```sh
gofmt -w internal/studio/sheets.go internal/studio/sheets_concurrency_test.go internal/interfaces/http/admin_keys.go internal/interfaces/http/admin_test.go
```

## Scope and self-review

Reviewed the scoped diff and the complete newly written concurrency regression once. Changes are limited to the two findings and their regression tests. The test barrier wraps real driver INSERT execution; it is confined to `_test.go`. Web QA and unrelated working-tree changes remain owned by other agents and were not touched. No data migration is needed for this not-live repository.

## Required whole-API verification

Executed from `services/api`, with the repository processor environment prepended to PATH:

```sh
env PATH="/Users/doduongtamdang/Organized/Development/Projects/eykh/restick/.venv/bin:$PATH" go vet ./... && env PATH="/Users/doduongtamdang/Organized/Development/Projects/eykh/restick/.venv/bin:$PATH" go test -race ./...
```

Completed successfully. `go vet ./...` produced no output and succeeded (the `&&` then ran the race suite). Complete race-suite output:

```text
ok  	github.com/eykh/restick/api_svc/cmd/restick	17.766s
ok  	github.com/eykh/restick/api_svc/internal/assets	(cached)
ok  	github.com/eykh/restick/api_svc/internal/auth	(cached)
ok  	github.com/eykh/restick/api_svc/internal/infrastructure/config	(cached)
ok  	github.com/eykh/restick/api_svc/internal/integration	34.801s
ok  	github.com/eykh/restick/api_svc/internal/interfaces/http	87.987s
ok  	github.com/eykh/restick/api_svc/internal/jobs	9.806s
ok  	github.com/eykh/restick/api_svc/internal/maintenance	(cached)
ok  	github.com/eykh/restick/api_svc/internal/packs	(cached)
ok  	github.com/eykh/restick/api_svc/internal/storage	(cached)
ok  	github.com/eykh/restick/api_svc/internal/studio	17.996s
```

Reported wall time: 91.18 seconds. All 11 packages passed; no race reports. No backend concerns remain within this assignment. Web QA was intentionally not exercised by this backend-only task.
