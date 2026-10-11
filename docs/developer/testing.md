# Testing

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (the MVP's tiers exist; first measured budgets) · **Target:** Lens Desktop 2026.10.51501

The MVP (2026-10-11) brought the first tiers; the table says for each whether it exists, the
`make` target and where it lives. Tiers without code are the plan.

## Tiers

| Tier | Runs where | Needs | What it proves | State |
|---|---|---|---|---|
| Unit | `go test ./...` everywhere | nothing | logic in isolation; `model` and `kubeconfig` are fully testable without I/O | exists: `make test-unit` |
| Race | `go test -race ./...` | nothing | no data races; a concurrent map write would end the process | exists: `make test-unit-race` (CI not set up) |
| Kind-profile goldens | `internal/model` | recorded objects in `testdata/` | an object becomes the row and detail model Lens would show | exists: [`kinds/golden_test.go`](../../internal/model/kinds/golden_test.go), 49 synthetic objects; `-update` regenerates |
| Cache | `internal/cache`, `internal/testkit/integration` | an in-memory source; a fake API server over HTTPS | watches, warm period, error state, transforms, and the API requests each scenario costs | exists; envtest not used yet ([`budget_test.go`](../../internal/testkit/integration/budget_test.go)) |
| Engine | `internal/engine` | the real cache over an in-memory source | commands, snapshots, the drawer, confirmation per origin | exists |
| Command fuzzing | `go test -fuzz` on the registry | the in-memory source | every command, fed adversarial params, returns an error instead of panicking | exists: `FuzzDispatch` |
| MCP | `internal/mcp`, `internal/testkit/integration` | the SDK's in-memory transports and loopback HTTP | tools match the registry; the showcase; the HTTP endpoint rejects missing tokens and foreign hosts and origins | exists ([`showcase_test.go`](../../internal/testkit/integration/showcase_test.go)) |
| Agent | `internal/agent` | a fake OpenAI-compatible server | the tool loop, its limits, redaction of what is sent | not built |
| Cluster | opt-in locally | Docker and `kind` | views, logs, exec, `kubectl` in the terminal against a real kubelet | exists for the MVP: `make test-kind` (build tag `kind`), [`engine_kind_test.go`](../../internal/testkit/integration/engine_kind_test.go), `exec_kind_test.go`, `kind_test.go`; attach, port-forward, node shell and Helm not yet |
| UI | everywhere with cgo | Fyne's `test` package (offscreen) | views render from snapshots; images for visual review | exists: `internal/ui/...`, PNGs in `testdata/` (`-update` rewrites them) |
| Fuzz | locally | `go test -fuzz` | kubeconfig parsing, kind profiles, YAML tokenizer and edits, terminal escape filter | targets exist; no nightly run |
| Performance | locally | Go benchmarks and `cogpit --perf` | the budgets below | exists: `make bench`; no fixed runner |

## Fixtures

- **Recorded objects** are real API responses with names, namespaces, IPs and UIDs replaced by
  synthetic values. Never commit objects from a real cluster unscrubbed: they carry internal
  hostnames, registries and sometimes credentials in annotations or environment variables.
- **Clusters** are created by the tests (`kind`, envtest), never borrowed from a developer's
  kubeconfig. Tests must not read `~/.kube/config`; `kind` writes its kubeconfig to a temporary
  file (`--kubeconfig`), never into the user's.
- **API request counting:** tests wrap the client's transport in a counting round-tripper and
  assert the requests a scenario may cost.

## Performance budgets (proposed, not measured)

Targets to validate against Lens Desktop on the same machine before they are enforced. The heading
keeps its name so that links survive; the "Measured" column holds what Cogpit reached on
2026-10-11 on an Apple M1 Ultra (macOS 26.6.2), without a Lens comparison yet.

| Id | Scenario | Proposed target | Measured (2026-10-11) |
|---|---|---|---|
| P1 | Open a context to the first rows of the Pods list, 1,000-pod cluster | ≤ 1 s | not measured |
| P2 | Sort a 10,000-pod list by a column | ≤ 50 ms | 10.2 ms sort, 11.6 ms command to new snapshot (`BenchmarkSortChange10k`, `BenchmarkSortCommand10k`); 41 µs to apply it to the table without paint |
| P3 | Scroll a 10,000-pod list | every frame ≤ 16 ms | real window: no main-goroutine stall over 16 ms after the first second (jank meter, `cogpit --demo --perf`); a start-up stall of 43–460 ms in the first second ([S1](spike-fyne.md#results)) |
| P4 | Apply a burst of 1,000 watch events per second to an open list | no frame over 16 ms | engine side: 29 snapshots in 3 s at 1,000 events/s (`TestDemoClusterThroughTheEngine`), a 10,000-row snapshot in 2.1–4.6 ms; frames: as P3, the same runs had the churn on |
| P5 | Follow a log stream at 10,000 lines per second | no frame over 16 ms; memory bounded by the scrollback limit | offscreen: 0.15 ms median, 2.3 ms max of work per 100 ms batch at ~9,800 lines/s; 1.5 MiB for 10,000 kept lines (termview logs test); no GPU frames measured |
| P6 | Resident memory with three clusters open on their Pods lists | track only until measured against Lens | the demo cluster with 10,000 pods keeps ~98 MB of its own heap; Cogpit's not measured |
| P7 | Cold start to an interactive window | ≤ 1 s | not measured |
| P8 | Type into a 5 MB YAML document in the editor, highlighting on | keystroke to repaint ≤ 16 ms | 0.015 ms widget work; 26.4 ms offscreen repaint, of which 18.2 ms is the software painter's fixed cost; GPU not measured |
| P9 | A view stays open on an idle cluster for 10 minutes | 0 list requests; only the view's watches | 0 requests in 2 s idle (fake API server) and 3 s idle (kind); 1 streaming watch per kind (`TestBudgetIdleViewAndWarmReopen`, `TestKindIdleViewsCostNoLists`); 10 minutes not run |
| P10 | Reopen a view within the warm period | 0 list requests; rows shown ≤ 100 ms | 0 requests; rows in ~90 µs |
| P11 | An agent asks `ui_state`, `resources_find` and `ui_navigate` for an object in a warm cache | 0 API requests | 0 lists and 0 watches on the source (`TestShowcaseServiceByClusterIP`) |

## Before a pull request

```sh
make lint
make test-unit-race
make test-kind      # when you touched cluster, cache, terminal or engine streams; needs make kind-up
```
