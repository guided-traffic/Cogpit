# Beyond Lens

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-12 · **Change:** minor (everyday use: the status bar link done, copying from the drawer partial) · **Target:** Lens Desktop 2026.10.51501

What Cogpit does that Lens Desktop does not, or does only in part. These rows are Cogpit's own
targets, decided by the owner; they do not count toward the parity numbers in
[`target-app-parity.md`](target-app-parity.md), which measure Cogpit against Lens. Method:
[`README.md`](README.md).

**Summary (measured, 2026-10-12):** 19 of 36 rows exist: 11 done and 8 partial, in MCP control,
cache transparency, crash recovery (the unfinished drain) and everyday use (the status bar link
done, copying from the drawer partial). Remaining (estimated): 30–60 h for MCP control
([G17](gaps.md#g17-mcp-control)), 40–80 h for the built-in agent
([G21](gaps.md#g21-built-in-agent)); updates, cache transparency and crash recovery are inside
[G14](gaps.md#g14-packaging-signing-and-updates-macos), [G3](gaps.md#g3-cache-and-the-resource-model),
[G7](gaps.md#g7-resource-actions) and [G11](gaps.md#g11-helm).

## MCP control of UI and cluster

[D-2](../developer/decisions.md#d-2-agent-interface-scope). Lens Desktop's MCP server is paid and
read-only: `list-clusters`, `connect-cluster`, `disconnect-cluster`, `kubectl`, and Prometheus tools
([Lens MCP server](https://docs.lenshq.io/k8slens/mcp-server/)).

| Feature | MCP tool | Status | Evidence |
|---|---|---|---|
| Every registry command is an MCP tool, with its schema and annotations | (generated) | done | [`mcp/tools.go`](../../internal/mcp/tools.go), `TestToolsMirrorRegistry` in [`tools_test.go`](../../internal/mcp/tools_test.go); 41 tools (2026-10-11, was 21) |
| Read the UI state: context, view, namespaces, filter, selection, drawer object, dock tabs | `ui_state` | done | [`commands/ui.go`](../../internal/engine/commands/ui.go), `TestShowcaseServiceByClusterIP` |
| Read the rows the user sees in the current view | `view_rows` | missing | – |
| List the kinds the cluster serves and the user may list | `kinds_list` | partial: served kinds, each with `listable` from the access reviews (2026-10-11); absent while the reviews have not answered, and for a namespaced kind no reviewed namespace allows when the user may list more than 32 namespaces | [`commands/resources.go`](../../internal/engine/commands/resources.go), [`accessreviews.go`](../../internal/engine/accessreviews.go); `TestKindsListListable`, `TestBudgetAccessReviews`, `TestKindAccessReviews` |
| Find objects by a searchable field (`clusterIP`, `podIP`, `node`, `image`, `label`) | `resources_find` | done | [`commands/resources.go`](../../internal/engine/commands/resources.go), `TestShowcaseServiceByClusterIP`, `TestDemoClusterThroughTheEngine` |
| List objects of a kind, bounded | `resources_list` | done | [`commands/resources.go`](../../internal/engine/commands/resources.go), `TestKindEngineWorkflows` |
| Read one object (YAML, events, related objects) from the cache | `object_get` | partial: YAML (redacted) and events; no related objects | [`commands/resources.go`](../../internal/engine/commands/resources.go) |
| Open a view, set namespaces and filter | `ui_openView`, `ui_setNamespace`, `ui_setFilter` | done | [`commands/ui.go`](../../internal/engine/commands/ui.go) and its tests |
| Select objects and open the drawer | `ui_select` | done | [`commands/ui.go`](../../internal/engine/commands/ui.go) |
| Navigate to an object: open its view, select it, scroll to it, open the drawer | `ui_navigate` | done | `TestShowcaseServiceByClusterIP`; the GUI scrolls to the row (`TestNavigateScrollsOnceSnapshotHasRow`, `TestRenderDemoShowcase`, [PNG](../../internal/ui/fyneui/testdata/demo_showcase.png)) |
| Open and read dock tabs (logs, terminal output, editor) | `dock_open`, `dock_read` | partial: agents open tabs with `pod_logs`, `pod_shell`, `terminal_open`, `object_edit`, `resource_createTab` and close them with `dock_close`; they read and set editor tabs' text with `editor_get` and `editor_setText`; logs and terminal output they cannot read | [`commands/dock.go`](../../internal/engine/commands/dock.go) |
| Run cluster actions with the GUI's verbs and danger levels | every action command | partial: Restart (Deployment); Delete, Edit (`editor_diff`, `object_dryRun`, `object_apply`) and Create (`object_create`) for every kind | [`commands/actions.go`](../../internal/engine/commands/actions.go) |
| Mutating commands from agents wait for the user's acceptance unless allowed for the context | (engine) | partial: the engine and the GUI dialog do; the allowance has no UI | `TestDispatchConfirmationPolicy`, [`fyneui/confirm.go`](../../internal/ui/fyneui/confirm.go) |
| Destructive commands from agents always wait for the user's confirmation in the GUI | (engine) | done | `TestDispatchConfirmationPolicy`, `TestDeleteGoesThroughConfirmation`, MCP tests for declined and headless calls |
| Audit log of every agent-initiated command, readable in the GUI | (engine) | partial: written to `<user config dir>/Cogpit/audit.log`; not readable in the GUI | [`engine/audit.go`](../../internal/engine/audit.go) |
| Streamable HTTP on `127.0.0.1` with a bearer token; `Host` and `Origin` checks | (transport) | done | [`mcp/http.go`](../../internal/mcp/http.go), `TestHTTPGuard`, `TestHTTPEndToEnd` |
| `cogpit-cli mcp` stdio bridge for stdio-only agents | (transport) | missing | – |

## Built-in agent

[D-10](../developer/decisions.md#d-10-built-in-agent). Lens Desktop's "Ask AI" (paid) launches an
external AI command-line tool with the cluster's context ([Ask AI](https://docs.lenshq.io/k8slens/ask-ai/)).

| Feature | Status | Evidence |
|---|---|---|
| Chat tab beside the views, running while the user keeps working | missing | – |
| Knows what the user sees: the current view, selection and drawer object are part of every turn | missing | – |
| Answers questions about the selected object from the cache (YAML, events, related objects, logs on request) | missing | – |
| Navigates the UI for the user ("show me the Service with ClusterIP 1.2.3.4") | missing | – |
| Runs cluster actions under the confirmation rules of D-2 | missing | – |
| LLM through an OpenAI-compatible Chat Completions endpoint with tool calling and streaming: LiteLLM, Open WebUI or any compatible server | missing | – |
| Settings: base URL, API key in the operating system's keychain, model picked from the endpoint's model list | missing | – |
| Limits per turn: tool calls, tokens, time | missing | – |
| Data sent to the endpoint is visible to the user; Secret values are never sent unless allowed for that request | missing | – |
| ⌘I / Ctrl+I opens the chat for the current cluster (Lens's Ask AI shortcut) | missing | – |

## Updates

[D-4](../developer/decisions.md#d-4-packaging-and-updates). Lens Desktop updates its exe, dmg and
AppImage builds in place; APT and RPM installs update by hand
([install](https://docs.lenshq.io/k8slens/getting-started/install-lens/)).

| Feature | Status | Evidence |
|---|---|---|
| One click updates the macOS app through Sparkle 2; MSI, portable zip, AppImage and tar.xz with Windows and Linux (M11) | missing | – |
| Package-manager installs: the button runs or shows the manager's command | missing | – |
| Signed update manifest verified with a key built into the app, before anything is replaced | missing | – |

## Cache transparency

[D-13](../developer/decisions.md#d-13-cache).

| Feature | Status | Evidence |
|---|---|---|
| Diagnostics view: API requests per cluster by verb and resource, open and warm watches, objects and memory per kind | partial: totals in the status bar (requests, lists, watches, informers, snapshots); no view | [`fyneui/bars.go`](../../internal/ui/fyneui/bars.go), [`engine/diagnostics.go`](../../internal/engine/diagnostics.go) |

## Crash recovery

[D-11](../developer/decisions.md#d-11-crash-stance).

| Feature | Status | Evidence |
|---|---|---|
| On start, Helm releases left `pending-install`, `pending-upgrade` or `pending-rollback` by Cogpit are shown with "finish" and "roll back" | missing | – |
| On start, nodes Cogpit cordoned for a drain it did not finish are shown with "continue drain" and "uncordon" | done (2026-10-11): on connecting the context, a banner names the node and offers Finish drain, Uncordon and Dismiss; agents read the same list with `node_pendingDrains` | [`drains.go`](../../internal/engine/drains.go) (a journal written before the cordon), `TestKindDrainCrashFinishUncordon` (three-node kind cluster: a simulated crash, the offer on the next session, finish, uncordon), the banner's offscreen PNGs |
| Unsaved editor content restored from its local draft | missing | – |

## Everyday use

The owner's requirements of 2026-10-12 that Lens Desktop does not have.

| Feature | Status | Evidence |
|---|---|---|
| Every value in the detail drawer can be selected, and has a copy icon on hover; an entry of a value with several entries is copied in YAML syntax; a Secret's value only once revealed, and the clipboard cleared after 60 seconds ([D-25](../developer/decisions.md#d-25-copying-values)) | partial (2026-10-12): text lines (names, values, table cells, titles, events), event messages and revealed Secret values select by drag or double click and copy with ⌘C or Ctrl+C; every value, table cell and chip shows a copy icon on hover; entries of labels, annotations and other lists copy as `key: value` or a YAML scalar; every chip owns the room between itself and its icon, so the pointer reaches the icon of a label chip, the last chip of a row and a wrapped one; after a copy the icon is a checkmark for 1.5 s (2026-10-12); a masked Secret value has no icon, a revealed one copies and the clipboard is cleared after 60 seconds if it still holds it, a later copy cancels that, closing the window clears at once; no command. Not drag-selectable: links, chips and status pills (the icon copies them); a selection stays within one line | [`selection.go`](../../internal/ui/fyneui/selection.go), [`copy.go`](../../internal/ui/fyneui/copy.go); `TestDrawerCopyIconOnHover`, `TestDrawerCopiesEntriesInYAML`, `TestCopyIconReachableOnLabelChips`, `TestCopyIconReachableOnWrappedChips`, `TestCopyIconTurnsIntoACheckmark`, `TestYAMLEntry`, `TestDrawerTextIsSelectable`, `TestDrawerLongNamesSelectUnderTheEllipsis`, `TestSecretValueCopyOnlyOnceRevealed`, `TestSecretClipboardKeepsWhatReplacedIt`, `TestSecretTextSelectionCopiesAsASecret`, `TestCopyHasNoCommand` ([H-25](../security/credentials.md#h-25-a-copied-secret-value-reaches-the-clipboard)) |
| A link to the owner at the status bar's left end, opening hans-fischer.com in the browser on a click, a GUI link and not a command ([D-23](../developer/decisions.md#d-23-a-link-to-the-owner-in-the-status-bar)) | done (2026-10-12): "Need a Kubernetes Expert? Visit: hans-fischer.com", shown before connecting and when connected; a click opens `https://hans-fischer.com` and nothing is sent before; when the bar is too narrow the link is shortened and then hidden with its divider, and the status items keep their room | [`bars.go`](../../internal/ui/fyneui/bars.go) (`statusLayout`, `openOwnerLink`); `TestStatusBarOwnerLink`, `TestStatusBarOwnerLinkIsNoCommand`, `TestStatusBarOwnerLinkGivesWayFirst`; [PNG](../../internal/ui/fyneui/testdata/shell_dark.png) (the status bar's left end) |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-12 | minor | Everyday use: the status bar link (D-23) done, copying values (D-25) partial; recounted by the lead: 19 of 36 rows, 11 done and 8 partial |
| 2026-10-12 | minor | Everyday use: copying values (D-25) and the status bar link (D-23), both missing; 17 of 36 rows |
| 2026-10-11 | minor | `kinds_list` reports per kind whether the user may list it (access reviews); still partial |
| 2026-10-11 | minor | Crash recovery: an unfinished drain is offered on the next connect (done); 17 of 34 rows |
| 2026-10-11 | minor | G9: agents edit, check, apply, create and delete objects of every kind, and read and set editor tabs |
| 2026-10-11 | minor | The MVP: MCP tools, transport, navigation and confirmation rows; the status bar's counters |
| 2026-10-11 | major | First version: MCP control, built-in agent, updates, cache transparency, crash recovery |
