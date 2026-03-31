# Review Follow-up

This document tracks the P1-P3 follow-up items we decided to keep from the external review. It is intended to stay current while work is in progress.

## Status Legend

- `planned`: not started yet
- `in_progress`: currently being worked on
- `done`: implemented and verified
- `deferred`: intentionally postponed

## Work Queue

| Order | Priority | Item | Status | Notes |
| --- | --- | --- | --- | --- |
| 1 | P1 | Deduplicate `SkipMode` | done | `bootstrap.SkipMode` now aliases the engine type instead of redefining it. |
| 2 | P1 | Split `Engine.Tick()` into smaller helpers | done | Session completion, tick short-circuiting, scheduler advancement, and event dispatch are now separated. |
| 3 | P1 | Add frontend lint baseline | done | Added ESLint flat config, lint script, and cleaned reported issues to a passing baseline. |
| 4 | P2 | Add minimal frontend test baseline | done | Added Vitest with initial bridge/helper coverage and a passing test script. |
| 5 | P2 | Improve `logx` test isolation | done | Added explicit logger reset helpers and coverage proving clean re-initialization in tests. |
| 6 | P2 | Revisit `Fatalf` exit strategy | done | Removed process exit from `logx`; entrypoints now log and exit explicitly. |
| 7 | P2 | Log `runtime.Close()` failures during init cleanup | done | Early cleanup paths now log close failures instead of dropping them silently. |
| 8 | P2 | Consider constructor injection for notifier | done | `Engine` now receives notifier at construction, so bootstrap wiring is complete at creation time. |
| 9 | P3 | Split frontend `useSettings` responsibilities | done | Broke notification and reminder management into internal hooks while keeping the external hook contract stable. |
| 10 | P3 | Plan `history.db` migration strategy | done | Added a concrete migration/versioning plan document for future schema changes. |

## Update Log

- 2026-03-31: Created this tracking document and started item 1 (`SkipMode` deduplication).
- 2026-03-31: Completed `SkipMode` deduplication and verified with `go test ./internal/backend/runtime/engine ./internal/backend/bootstrap ./internal/app`.
- 2026-03-31: Completed `Engine.Tick()` refactor into smaller helpers. Ran the same Go test suite and did a post-change elegance review to confirm the split reduced branching and repeated state-reset logic without changing behavior.
- 2026-03-31: Completed frontend lint baseline work with a passing `npm run lint`. Kept the rules practical for the current codebase, and resolved the remaining warnings with small code cleanups instead of suppressing them.
- 2026-03-31: Completed minimal frontend test baseline work with `Vitest`, plus passing `npm test -- --run` and `npm run lint`.
- 2026-03-31: Completed `logx` test isolation work. Added reset helpers used only by tests and verified fresh logger initialization with `go test ./internal/logx ./internal/backend/runtime/engine ./internal/backend/bootstrap ./internal/app`.
- 2026-03-31: Completed `Fatalf` exit-strategy cleanup. `logx` now only logs, while `main` decides when to call `os.Exit(1)`. Verified with package tests plus `go test ./...`.
- 2026-03-31: Completed init-cleanup error logging for runtime/history close paths. Verified with targeted package tests plus `go test ./...`.
- 2026-03-31: Completed notifier constructor-injection cleanup. This simplified bootstrap wiring and removed the partially initialized `Engine` state after construction. Verified with targeted tests plus `go test ./...`.
- 2026-03-31: Completed `useSettings` responsibility split by extracting notification and reminder-management concerns into focused internal hooks. Verified with `npm run lint` and `npm test -- --run`.
- 2026-03-31: Added `docs/historydb-migration-plan.md` to define the first safe versioning/migration path for `history.db`.
