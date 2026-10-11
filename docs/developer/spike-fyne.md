# Fyne spike

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (first version: plan, gate criteria and findings from reading the sources; not run yet) · **Target:** Lens Desktop 2026.10.51501

The gate of [D-1](decisions.md#d-1-ui-toolkit): before milestone M2 builds the real UI on Fyne, a
spike builds the parts most likely to fail and measures them. If the gate passes, M2 builds on
Fyne. If it fails, the engine stays in Go and the UI becomes a separate Rust/egui process.

**Status: planned, not run.** It runs after M0 and before M2.

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

Not run yet. Each criterion gets its measured value, the machine, the date and the commit here.

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | First version: plan, gate criteria, findings from the sources |
