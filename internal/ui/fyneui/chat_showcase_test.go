package fyneui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// 14. The agent chat beside the Pods list (showcase_test.go's README pictures): the user asks why
// the selected pod keeps restarting; the agent reads the pod and its log, and answers with a table
// of its containers, the log line and the cause.
func TestShowcaseAgentChat(t *testing.T) {
	showcaseOnly(t)
	p := podNamed(t, podCrashLoop)
	f := kindsFake(t, "pods")
	withUsageCells(t, f, "pods")
	ref := model.ObjectRef{Context: "demo", Kind: "pods", Namespace: p.ns, Name: p.name}
	f.ui.Selected = []model.ObjectRef{ref}
	showcaseTabs(f, "pods", engine.ClusterOverview, "deployments")
	fakeTabCommands(t, f)
	ch := withChat(t, f)
	ch.conns = []engine.ChatConnection{lmStudio(), litellm()}
	restarts := podRestarts(f, p.name)
	conv := showcaseChat(ch, p, restarts)
	f.ui.Chat = true
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the conversation", func() bool {
		return len(sh.kinds) > 0 && sh.chat.convID == conv && len(sh.chat.views) > 0 && sh.diag.total > 0 && len(sh.tabs.shown) == 3
	})
	settle(w)
	scrollRowToTop(sh, len(f.tables["pods"].Rows), rowOf(t, f, "pods", p.name)-19)
	sh.chat.scroll.ScrollToTop()
	sh.chat.updatePin()
	saveShowcase(t, w, "agent_chat.png")
}

// podRestarts is the pod's Restarts cell as the list shows it, so that the answer agrees with it.
func podRestarts(f *fakeEngine, name string) int64 {
	snap := f.tables["pods"]
	col := -1
	for i, c := range snap.Columns {
		if c.ID == "restarts" {
			col = i
		}
	}
	for _, r := range snap.Rows {
		if r.Ref.Name == name && col >= 0 && col < len(r.Cells) {
			return r.Cells[col].Num
		}
	}
	return 0
}

// showcaseChat is a finished turn about a crash-looping pod: the question with the pod attached,
// three reads and the answer.
func showcaseChat(ch *fakeChat, p podSpec, restarts int64) string {
	app := p.containers[0]
	db := strings.Split(p.ns, "-")[0] + "-db"
	podRef := `{"ref":{"kind":"pods","namespace":"` + p.ns + `","name":"` + p.name + `"}}`
	rows := "| `" + app + "` | CrashLoopBackOff | " + strconv.FormatInt(restarts, 10) + " | `Error` (exit code 1) |\n"
	for _, c := range p.containers[1:] {
		rows += "| `" + c + "` | Running, ready | 0 | – |\n"
	}
	answer := "**" + p.name + "** is crash-looping: its container `" + app + "` exits with code 1 a few seconds after " +
		"each start, because it cannot reach its database.\n\n" +
		"| Container | State | Restarts | Last exit |\n|---|---|---:|---|\n" + rows + "\n" +
		"Its log says why, on every start:\n\n```\ndial tcp 10.96.14.7:5432: connect: connection refused\n```\n\n" +
		"1. The Service `" + db + "` has **no endpoints**: its selector `app=" + db + "` matches no running pod.\n" +
		"2. The StatefulSet `" + db + "` was scaled to `0` an hour ago.\n\n" +
		"Scaling `" + db + "` back to one replica should fix it. Shall I?"
	sent := &engine.ChatSent{Requests: 4, Bytes: 51_377, Messages: 8, ToolResults: 3, Tokens: 13_206, Endpoint: "localhost:1234", Private: true}
	return ch.add("demo", "Why does "+p.name+" keep restarting?",
		engine.ChatItem{Kind: engine.ChatItemUser, Text: "Why does this pod keep restarting?",
			Attached: "Pods · " + p.ns + "/" + p.name, Sent: sent},
		engine.ChatItem{Kind: engine.ChatItemTool, Tool: &engine.ChatToolCall{CallID: "s1", Command: "object.get", Label: "Get Object",
			Summary: "Pod " + p.ns + "/" + p.name, State: engine.ChatToolDone, Args: podRef}},
		engine.ChatItem{Kind: engine.ChatItemTool, Tool: &engine.ChatToolCall{CallID: "s2", Command: "pod.logs", Label: "Logs",
			Summary: p.name + " · " + app, State: engine.ChatToolDone, Args: podRef}},
		engine.ChatItem{Kind: engine.ChatItemTool, Tool: &engine.ChatToolCall{CallID: "s3", Command: "dock.read", Label: "Read Tab Output",
			Summary: "Logs: " + p.name + ", the last 200 lines", State: engine.ChatToolDone}},
		engine.ChatItem{Kind: engine.ChatItemAssistant, Text: answer},
	)
}
