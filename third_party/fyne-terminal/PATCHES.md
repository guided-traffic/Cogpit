# Cogpit's patches to fyne-io/terminal

This directory is `github.com/fyne-io/terminal` at commit `46cd12724ff3` (2026-10-09; moved from
`91063bb29d69` on 2026-10-12, whose two upstream commits since touch none of the patched code),
without its `cmd/` and `img/` directories, used by Cogpit through a `replace` directive in Cogpit's `go.mod`.
Terminal output comes from containers, which anyone who can deploy to a cluster controls; the
upstream widget crashed or froze Cogpit's whole process on some of it, on Fyne's main goroutine,
where nothing can recover. Each patch is marked `Cogpit patch:` in the code. They are candidates
for upstream; drop the fork once upstream has equivalent fixes.

| File | Patch |
|---|---|
| `output.go` | A tab stops at the right margin and never wraps; unclamped, the tab loop wrapped to column 0 before reaching its stop and never ended |
| `escape.go` | Delete characters (`CSI P`) does nothing when the cursor is past the row's cells; it sliced out of range |
| `escape.go` | Insert characters (`CSI @`) creates the cursor's row and pads it to the cursor first; it indexed a missing row or sliced out of range |
| `escape.go` | Scroll margins (`CSI r`) are clamped to the screen, and a region whose top is not above its bottom resets to the full screen; a bottom of 0 made scrolling index out of range |
| `output.go` | `scrollUp` and `scrollDown` grow the rows to the region and ignore an invalid region |
| `output.go` | `ESC 7`, `ESC 8`, `ESC D` and `ESC M` run on Fyne's main goroutine like the CSI sequences, instead of the reader goroutine; the restored cursor goes through `moveCursor`, which clamps it to the screen |
| `osc.go` | OSC 7 (the shell reporting its directory) records the directory in the configuration instead of changing the whole process's working directory |
| `escape_test.go` | `TestScrollBack_Tmux` expects the 5 rows of its screen rather than the 46 an unclamped region grew the grid to |

`go.mod` requires Fyne v2.8.1, the latest release, where upstream's asks for an unreleased Fyne
(`v2.8.2-0.20261009111023-0a528b3a387b`); its indirect requirements are the ones `go mod tidy`
derives from that, so the copy never raises a module in Cogpit's build above what Fyne v2.8.1
needs. The code builds and its tests pass on v2.8.1 (2026-10-12), `TestExitCode*` included, which
hung before upstream's `c8f30fa`. Licence: BSD-3-Clause, see [`LICENSE`](LICENSE).
