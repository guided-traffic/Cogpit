package commands_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// fakeLoginItem is the operating system's login item: a status, the calls made, and what the next
// register answers.
type fakeLoginItem struct {
	mu          sync.Mutex
	status      string
	calls       []string
	registerErr error
	afterReg    string // the status a register leaves; empty: on
}

func (f *fakeLoginItem) Status() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.status
}

func (f *fakeLoginItem) Register() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "register")
	if f.registerErr != nil {
		return f.registerErr
	}
	f.status = engine.LoginItemOn
	if f.afterReg != "" {
		f.status = f.afterReg
	}
	return nil
}

func (f *fakeLoginItem) Unregister() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "unregister")
	f.status = engine.LoginItemOff
	return nil
}

func (f *fakeLoginItem) OpenSettings() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, "settings")
	return nil
}

func (f *fakeLoginItem) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls)
}

// Start at login is the operating system's state, shown as it reports it: unknown until read;
// the user turns it on and off in the window; agents and the CLI may turn it off and are refused
// turning it on, before macOS is asked; only the user opens System Settings; nothing is kept in
// Cogpit's preferences file.
func TestStartAtLogin(t *testing.T) {
	li := &fakeLoginItem{status: engine.LoginItemOff}
	prefs := engine.NewPreferencesStore(t.TempDir())
	h := newHarness(t, seed(), engine.Options{LoginItem: li, Preferences: prefs})

	if st := h.s.UIState().LoginItem; st == nil || st.Status != engine.LoginItemUnknown {
		t.Fatalf("at start %+v, want offered and not read", st)
	}
	if len(li.called()) != 0 {
		t.Errorf("calls at start %v", li.called())
	}
	var got engine.LoginItemState
	h.must(t, engine.OriginExternalAgent, "preferences.loginItem", struct{}{}, &got)
	if got.Status != engine.LoginItemOff || h.s.UIState().LoginItem.Status != engine.LoginItemOff {
		t.Errorf("read %+v", got)
	}

	for _, o := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		err := h.run(t, o, "preferences.setStartAtLogin", map[string]any{"enabled": true}, nil)
		if !errors.Is(err, engine.ErrStartAtLoginGUIOnly) {
			t.Errorf("%s turning it on: %v", o, err)
		}
	}
	if len(li.called()) != 0 {
		t.Fatalf("macOS asked for a refused origin: %v", li.called())
	}

	h.must(t, engine.OriginGUI, "preferences.setStartAtLogin", map[string]any{"enabled": true}, &got)
	if got.Status != engine.LoginItemOn || h.s.UIState().LoginItem.Status != engine.LoginItemOn {
		t.Errorf("after turning it on %+v", got)
	}
	h.must(t, engine.OriginExternalAgent, "preferences.setStartAtLogin", map[string]any{"enabled": false}, &got)
	if got.Status != engine.LoginItemOff || !slices.Equal(li.called(), []string{"register", "unregister"}) {
		t.Errorf("an agent turning it off: %+v, calls %v", got, li.called())
	}

	err := h.run(t, engine.OriginExternalAgent, "preferences.openLoginItems", struct{}{}, nil)
	var gui *engine.GUIOnlyError
	if !errors.As(err, &gui) || !errors.Is(err, engine.ErrLoginItemsGUIOnly) {
		t.Errorf("an agent opening System Settings: %v", err)
	}
	h.must(t, engine.OriginGUI, "preferences.openLoginItems", struct{}{}, nil)
	if c := li.called(); c[len(c)-1] != "settings" {
		t.Errorf("calls %v", c)
	}

	h.must(t, engine.OriginGUI, "preferences.set", map[string]any{"theme": "light"}, nil) // the file is written
	b, err := os.ReadFile(prefs.Path())
	if err != nil || strings.Contains(strings.ToLower(string(b)), "login") {
		t.Errorf("the preferences file (%v): %s", err, b)
	}
}

// A register that leaves the item waiting for the user's approval in System Settings succeeded;
// one that fails says why and shows the status as it is; where no login item is wired in, nothing
// is offered.
func TestStartAtLoginOutcomes(t *testing.T) {
	li := &fakeLoginItem{status: engine.LoginItemOff, afterReg: engine.LoginItemNeedsApproval}
	h := newHarness(t, seed(), engine.Options{LoginItem: li})
	var got engine.LoginItemState
	h.must(t, engine.OriginGUI, "preferences.setStartAtLogin", map[string]any{"enabled": true}, &got)
	if got.Status != engine.LoginItemNeedsApproval {
		t.Errorf("waiting for approval: %+v", got)
	}

	li.mu.Lock()
	li.status, li.registerErr = engine.LoginItemNotFound, errors.New("Invalid code signature")
	li.mu.Unlock()
	err := h.run(t, engine.OriginGUI, "preferences.setStartAtLogin", map[string]any{"enabled": true}, nil)
	if err == nil || !strings.Contains(err.Error(), "Invalid code signature") ||
		h.s.UIState().LoginItem.Status != engine.LoginItemNotFound {
		t.Errorf("a failed register: %v, shown %+v", err, h.s.UIState().LoginItem)
	}

	none := newHarness(t, seed(), engine.Options{})
	if none.s.UIState().LoginItem != nil {
		t.Error("offered without a login item")
	}
	for id, params := range map[string]any{"preferences.loginItem": struct{}{}, "preferences.openLoginItems": struct{}{},
		"preferences.setStartAtLogin": map[string]any{"enabled": false}} {
		var dis *engine.DisabledError
		if err := none.run(t, engine.OriginGUI, id, params, nil); !errors.As(err, &dis) || dis.Reason != engine.ErrNoLoginItem.Error() {
			t.Errorf("%s without a login item: %v", id, err)
		}
	}
}
