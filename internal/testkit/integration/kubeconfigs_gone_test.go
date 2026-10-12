package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestConnectionToASourceClosesWhenItsFileIsDeleted: connected through a real connector to a fake
// API server by an added file; deleting the file closes the connection within a poll, the state
// says why, and the connect log (cluster.connectLog, as an agent reads it) ends in that reason
// without the token.
func TestConnectionToASourceClosesWhenItsFileIsDeleted(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: sourceToken})
	defer srv.Close()
	s, _, _ := sourcesSession(t)
	file := srv.WriteKubeconfig(t, "fake", "team-a")
	id := "fake [" + filepath.Base(file) + "]"
	if err := sourcesDispatch(t, s, engine.OriginGUI, "kubeconfig.addFile", map[string]string{"path": file}, nil); err != nil {
		t.Fatal(err)
	}
	if err := sourcesDispatch(t, s, engine.OriginGUI, "cluster.connect", map[string]string{"context": id}, nil); err != nil {
		t.Fatal(err)
	}
	if b := s.Backend(); b == nil || b.SourceFile != file {
		t.Fatalf("backend %+v", b)
	}
	stop := s.WatchKubeconfigs(5 * time.Millisecond)
	defer stop()
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool { return s.Backend() == nil }) {
		t.Fatal("the connection stayed open after its file was deleted")
	}
	var log commands.ConnectLogResult
	if err := sourcesDispatch(t, s, engine.OriginExternalAgent, "cluster.connectLog", map[string]string{"context": id}, &log); err != nil {
		t.Fatal(err)
	}
	out, _ := json.Marshal(log)
	n := len(log.Lines)
	if n < 3 || log.Lines[n-1].Level != "ERROR" || !strings.Contains(log.Lines[n-1].Text, "is gone") || strings.Contains(string(out), sourceToken) {
		t.Errorf("connect log %s", out)
	}
	if st := s.UIState().Connection; st.State != engine.Disconnected || st.Error != log.Lines[n-1].Text {
		t.Errorf("connection %+v", st)
	}
}
