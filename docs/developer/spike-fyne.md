# Fyne spike

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (the spike ran as the local MVP; results S1–S8 recorded) · **Target:** Lens Desktop 2026.10.51501

The gate of [D-1](decisions.md#d-1-ui-toolkit): before milestone M2 builds the real UI on Fyne, a
spike builds the parts most likely to fail and measures them. If the gate passes, M2 builds on
Fyne. If it fails, the engine stays in Go and the UI becomes a separate Rust/egui process.

**Status: run on 2026-10-11 as the local MVP** (the MVP builds the spike's pieces on the planned
packages, see [Results](#results)). The gate decision is the owner's.

## What the spike builds

A throwaway app on the planned packages (`model`, `cache`, `engine`, `mcp`, `ui/fyneui`) with two
data sources:

- **demo:** a synthetic cluster in memory with 10,000 pods, a few hundred services with known
  cluster IPs, and a configurable churn of watch-like events (default 1,000 per second);
- **kind:** a real `kind` cluster through client-go informers, with its kubeconfig in a separate
  file so that the developer's `~/.kube/config` stays untouched.

And four pieces of UI:

1. the **Pods table** with sortable headers, a namespace filter and a text filter;
2. the **detail drawer** with a **YAML editor** (highlighting, line numbers, typing, copy and
   paste);
3. a **terminal** dock tab: a local shell with a per-terminal kubeconfig, and a shell in a pod;
4. **MCP control:** an external MCP client reads `ui_state`, calls
   `resources_find {kind: services, field: clusterIP, value: …}` and `ui_navigate`, and the GUI
   follows.

## Gate criteria

Measured on the slowest supported platform available, with the GPU renderer (a real window), not
only with Fyne's offscreen test driver.

| # | Criterion | Pass when | Budget |
|---|---|---|---|
| S1 | 10,000-row table under 1,000 events per second | no frame over 16 ms while scrolling and while events arrive | P3, P4 |
| S2 | Sort by a column on 10,000 rows | ≤ 50 ms from click to sorted view | P2 |
| S3 | Column widths | users can resize columns by dragging, or a workaround of comparable cost is shown | – |
| S4 | YAML editor on a 5 MB document | keystroke to repaint ≤ 16 ms with highlighting; memory proportional to the document, not to its cells | P8 |
| S5 | Terminal | local shell with its own `KUBECONFIG`, pod exec, resize, copy and paste; on macOS, Linux (X11 and Wayland) and Windows | – |
| S6 | MCP control | an MCP client navigates the GUI to an object; widget updates happen on Fyne's main goroutine without warnings | – |
| S7 | Packaging | `fyne package` builds the app for macOS, Windows and Linux on CI runners | – |
| S8 | Accessibility (recorded, not gating) | what a screen reader announces for the table and the drawer with `-tags accessibility` | – |

**Pass:** S1–S7 pass. **Fail:** any of S1, S2, S4 or S5 fails without a workaround the spike
demonstrates. S3 and S7 failures are recorded and weighed.

## Known before the spike

Verified on 2026-10-11 by reading the sources of the pinned versions:

| Finding | Source | Consequence for the spike |
|---|---|---|
| `widget.Table` supports header rows (`ShowHeaderRow`, `CreateHeader`, `UpdateHeader`), sticky rows and columns, `SetColumnWidth` | `fyne.io/fyne/v2@v2.8.1/widget/table.go` | sorting by header click is straightforward; whether users can drag column widths was not found in the changelog (criterion S3) |
| `widget.TextGrid` renders only the visible rows, but stores every character as a `TextGridCell` and computes its minimum size by walking all rows | `widget/textgrid.go` | a 5 MB YAML must not be loaded into one TextGrid; the editor keeps its own lines and fills a grid the size of the viewport |
| Fyne's accessibility support is "currently off by default, use `-tags accessibility` to test" | Fyne changelog (v2.8.x) | criterion S8 records the state; no gate |
| Wayland is supported by default and picked at runtime | Fyne changelog | test both X11 and Wayland in S5 |
| `fyne-io/terminal`'s `RunLocalShell` starts the shell with the process's own environment | `term_unix.go` | per-terminal `KUBECONFIG` needs Cogpit's own PTY and `RunWithConnection` |
| `fyne-io/terminal` uses ConPTY on Windows | `term_windows.go` | the same approach is available to Cogpit's own PTY |
| `fyne-io/terminal` has no tagged releases; commit `46cd12724ff3` (2026-10-09) requires an unreleased Fyne, commit `91063bb29d69` (2026-09-27) builds with Fyne 2.8.1 | its `go.mod` history | pin `91063bb29d69` |
| Widget updates from other goroutines go through `fyne.Do` or `fyne.DoAndWait` | `fyne.io/fyne/v2@v2.8.1/thread.go` | engine events reach the UI only through `fyne.Do` |

## Effort

16–24 agent-hours.

## Results

Run on 2026-10-11 on an **Apple M1 Ultra (20 cores), macOS 26.6.2**, Go 1.27.2, Fyne v2.8.1,
fyne-io/terminal `91063bb29d69`, at commit `4a3401a`. "Offscreen" means Fyne's `test` driver
with its CPU software painter, which re-rasterises every frame and costs about 18 ms for an empty
1440×900 frame on its own, so offscreen frame times say nothing about the GPU renderer the gate
asks for. The GPU measurement is the one real-window run at the end (S1).

| # | Result | Value and evidence |
|---|---|---|
| S1 | **fails as written: a start-up stall; passes after the first second** | Real window with the GPU renderer, 1440×900, `cogpit --demo --perf <d>`: the jank meter posts a callback to the main goroutine every 16 ms and records how late it runs (Fyne has no frame callback, so this is a proxy for frame time), while the table auto-scrolls 3 rows per frame through 10,000 pods and the demo cluster sends 1,000 watch events per second. Four runs: p50 0.02 ms in every run; p99 7.5–273 ms; max 43–460 ms; with stall timestamps (two runs), every stall falls in the first second after the rows appear (43 ms at 0.06 s and 21 ms at 0.84 s; 336 ms at 0.00 s and 43 ms at 0.06 s), and none over 16 ms in the remaining 39 s and 19 s. The start-up stall (likely the first frames of a full table: cell creation and glyph uploads) is not analysed yet ([`perf.go`](../../internal/ui/fyneui/perf.go)) |
| S2 | **pass** | Sort 10,000 rows by another column: 10.2 ms (`BenchmarkSortChange10k`), and 11.6 ms from the `ui.sort` command to the new snapshot's event on an idle view (`BenchmarkSortCommand10k`), both in [`internal/engine/bench_test.go`](../../internal/engine/bench_test.go); applying a 10,000-row snapshot to the table: 41 µs without paint (`BenchmarkTableRefresh10k` in [`internal/ui/fyneui/bench_test.go`](../../internal/ui/fyneui/bench_test.go)). Budget P2 is 50 ms |
| S3 | **pass** | Fyne 2.8.1's table resizes a column when the user drags the gap between two header cells (`Cursor()` returns the resize cursor there, `Dragged` calls `SetColumnWidth`); `TestColumnResizeByDraggingHeaderSeparator` in [`internal/ui/fyneui/resize_test.go`](../../internal/ui/fyneui/resize_test.go) drags 60 points and the column widens by 60. Caveats: the grab zone is only the 4-point gap, and Fyne has no resize callback (the shell reads the width back) |
| S4 | **pass for memory; latency on the GPU not measured** | Own line storage and a viewport-sized set of `canvas.Text` runs ([`internal/ui/fyneui/yamlview`](../../internal/ui/fyneui/yamlview/yamlview.go)): a keystroke into a 5 MB document costs 0.015 ms of widget work (edit, tokenize, place); keystroke to offscreen repaint 26.4 ms, of which 18.2 ms is the software painter's fixed cost for the frame; `SetText` of 5 MB plus first paint 31.2 ms. Memory: 0.63 bytes per document byte on top of the text, against 25.8 bytes per byte (129 MB) for a TextGrid holding the same document |
| S5 | **pass on macOS, on a patched widget** | Local shell on Cogpit's own PTY with `KUBECONFIG=<own file>:<the user's files>`: `kubectl config current-context` and the namespace come from the terminal's file, which is 0600, holds no credential and is removed on close (`TestStartLocalBindsKubectlToContextAndNamespace` in [`internal/terminal/localshell_test.go`](../../internal/terminal/localshell_test.go)); pod exec over WebSocket with SPDY fallback against kind (`TestKindPodExec`); resize (`stty size` → 45 123); kubectl in Cogpit's terminal lists the kind cluster's pods (`TestKindEngineWorkflows`); ⌘C/⌘V, double-click and drag selection (offscreen, synthetic events). The pinned fyne-io/terminal froze (a tab at the right margin) or crashed (delete and insert characters, scroll regions) on output a container can send, and changed the process directory on OSC 7; Cogpit builds against a patched copy ([`third_party/fyne-terminal`](../../third_party/fyne-terminal/PATCHES.md)) that survives every known crashing input, with an escape filter ([`termview/sanitize.go`](../../internal/ui/fyneui/termview/sanitize.go)) in front (`TestRawWidgetHazards`, `TestTerminalFilterNeutralisesHazards`); see [H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input). Linux (X11, Wayland) and Windows not run |
| S6 | **pass** | An MCP client over loopback HTTP calls `resources_find` and `ui_navigate`; the engine's UI state and the next table snapshot follow ([`showcase_test.go`](../../internal/testkit/integration/showcase_test.go)), and the Fyne shell over the real engine and demo cluster opens Services, selects the row, scrolls to it and opens the drawer, with all widget updates through `fyne.Do` (`TestRenderDemoShowcase`, [PNG](../../internal/ui/fyneui/testdata/demo_showcase.png)). In a real window, Claude Code (`claude -p` with this server as its only MCP server, `--strict-mcp-config`) answered "show me the Service with ClusterIP 10.96.100.100 in Cogpit" by calling `resources_find` and `ui_navigate` against the kind cluster; Cogpit's UI state then had Services open, `web` selected and its drawer open (read back over MCP; both calls in the audit log) |
| S7 | **partial** | `make app` runs `fyne package` (fyne.io/tools v1.7.3) and produces an ad-hoc signed `Cogpit.app` for arm64 that `codesign --verify` accepts. Not done: a universal (arm64 + x86_64) build, CI runners, Windows and Linux packages (CI was out of scope for the MVP) |
| S8 | **recorded** | With `-tags accessibility` (builds on macOS with AppKit), only `Container`, `Label`, `Button` and `Hyperlink` implement `fyne.Accessible`, and the tree walk does not descend into widget renderers; the table and the detail drawer therefore announce nothing. The tree is rebuilt only on window show and `SetContent` |

**Gate reading (for the owner's decision).** S1 fails as the criterion is written: a stall of
43–460 ms in the first second after a 10,000-row view fills; after that, no frame over 16 ms while
scrolling under 1,000 events per second. Whether that is a fail of the gate (no workaround is
demonstrated yet) or a start-up defect to fix is the owner's call. S2, S3, S5 and S6 pass; S4 passes
for memory, but its keystroke latency on the GPU is unmeasured; S7 is partial (local macOS build
only). S5 holds only because Cogpit builds against a patched copy of fyne-io/terminal
(`third_party/fyne-terminal`): the pinned upstream commit froze or crashed on container output.

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | Results S1–S8 from the local MVP on an M1 Ultra |
| 2026-10-11 | major | First version: plan, gate criteria, findings from the sources |
