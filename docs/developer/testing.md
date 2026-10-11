# Testing

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (Go test tiers; API-load budgets P9–P11 for the cache) · **Target:** Lens Desktop 2026.10.51501

No test exists yet (2026-10-11). This page is the plan that M0 and the milestones after it
implement; when a tier lands, its row gains the `make` target that runs it and the files that hold
it.

## Tiers

| Tier | Runs where | Needs | What it proves |
|---|---|---|---|
| Unit | `go test ./...` everywhere | nothing | logic in isolation; `model` and `kubeconfig` are fully testable without I/O |
| Race | `go test -race ./...` in CI | nothing | no data races; a concurrent map write would end the process |
| Kind-profile goldens | `internal/model` | recorded objects in `testdata/` | an object becomes the row and detail model Lens would show |
| Cache | `internal/cache` | envtest (a real API server and etcd, no kubelet) | watches, warm period, relist on `410 Gone`, transforms, and the API requests each scenario costs |
| Engine | `internal/engine` | client-go fake clients | commands send the right requests, honour permissions, enforce confirmation per origin |
| Command fuzzing | `go test -fuzz` on the registry | fake clients | every command, fed adversarial params and object states, returns an error instead of panicking (PhotoCraft's `panic_hunt`) |
| MCP | `internal/mcp` | the SDK's in-memory transports | tools match the registry; `ui_state`, `resources_find`, `ui_navigate` and the confirmation flow work; the HTTP endpoint rejects missing tokens and foreign origins |
| Agent | `internal/agent` | a fake OpenAI-compatible server | the tool loop, its limits, redaction of what is sent |
| Cluster | Linux CI, opt-in locally | Docker and `kind` | logs, exec, attach, port-forward, node shell and Helm against a real kubelet |
| UI | every OS in CI | Fyne's `test` package (offscreen) | views render from snapshots; images for visual review |
| Fuzz | nightly | `go test -fuzz` | kubeconfig parsing, YAML editing, terminal escape handling |
| Performance | nightly, fixed runner | Go benchmarks and the app's benchmark mode | the budgets below |

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

Targets to validate against Lens Desktop on the same machine before they are enforced; they become
the scorecard's budget file with milestone M0.

| Id | Scenario | Proposed target |
|---|---|---|
| P1 | Open a context to the first rows of the Pods list, 1,000-pod cluster | ≤ 1 s |
| P2 | Sort a 10,000-pod list by a column | ≤ 50 ms |
| P3 | Scroll a 10,000-pod list | every frame ≤ 16 ms |
| P4 | Apply a burst of 1,000 watch events per second to an open list | no frame over 16 ms |
| P5 | Follow a log stream at 10,000 lines per second | no frame over 16 ms; memory bounded by the scrollback limit |
| P6 | Resident memory with three clusters open on their Pods lists | track only until measured against Lens |
| P7 | Cold start to an interactive window | ≤ 1 s |
| P8 | Type into a 5 MB YAML document in the editor, highlighting on | keystroke to repaint ≤ 16 ms |
| P9 | A view stays open on an idle cluster for 10 minutes | 0 list requests; only the view's watches |
| P10 | Reopen a view within the warm period | 0 list requests; rows shown ≤ 100 ms |
| P11 | An agent asks `ui_state`, `resources_find` and `ui_navigate` for an object in a warm cache | 0 API requests |

## Before a pull request (once the code exists)

```sh
make lint
make test-unit-race
make test
```
