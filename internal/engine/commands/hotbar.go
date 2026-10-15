package commands

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The hotbar (G13; engine/hotbar.go): the clusters the user pinned to the bar at the window's left
// edge, in banks of up to twelve. Every command but hotbar.list changes Cogpit's own preferences
// only (the hotbar's file under Cogpit's configuration directory), never a cluster and never a
// kubeconfig, so they are UI commands that agents run freely (D-2) and the audit log records;
// hotbar.openCluster connects, as cluster.connect does. Removing a bank that holds clusters asks
// first in the GUI (Cogpit's own dialog, inferred); the result returns the removed bank, so an
// agent can put it back.

// HotbarResult is the result of the hotbar commands: the hotbar as the user sees it now.
type HotbarResult struct {
	Hotbar engine.HotbarState `json:"hotbar"`
}

// BankRemoved is hotbar.removeBank's result: the bank removed, with its clusters, and the hotbar.
type BankRemoved struct {
	Removed engine.HotbarBank  `json:"removed"`
	Hotbar  engine.HotbarState `json:"hotbar"`
}

type hotbarClusterParams struct {
	Context string `json:"context,omitempty" jsonschema:"context id as cluster_contexts lists it (a bare context name works when one file only defines it); empty or omitted: the connected context"`
	Bank    int    `json:"bank,omitempty" jsonschema:"bank number as the hotbar's switcher shows it, from 1 (⇧F1); 0 or omitted: the bank shown now"`
}

type hotbarMoveParams struct {
	Context  string `json:"context" jsonschema:"context id of a tile in the bank, as ui_state's hotbar lists it"`
	Position int    `json:"position" jsonschema:"the tile's new place in the bank, from 1 (the top, F1); the tiles between move one place"`
	Bank     int    `json:"bank,omitempty" jsonschema:"bank number as the hotbar's switcher shows it, from 1 (⇧F1); 0 or omitted: the bank shown now"`
}

type hotbarSlotParams struct {
	Slot int `json:"slot" jsonschema:"the tile's place in the bank, from 1 (the top, F1) to 12 (F12)"`
	Bank int `json:"bank,omitempty" jsonschema:"bank number as the hotbar's switcher shows it, from 1 (⇧F1); 0 or omitted: the bank shown now"`
}

type hotbarBankParams struct {
	Bank int `json:"bank,omitempty" jsonschema:"bank number as the hotbar's switcher shows it, from 1 (⇧F1); 0 or omitted: the bank shown now"`
}

type hotbarSwitchParams struct {
	Bank int `json:"bank" jsonschema:"number of the bank to show, from 1 (⇧F1) to 12 (⇧F12)"`
}

type hotbarCreateParams struct {
	Name string `json:"name,omitempty" jsonschema:"the new bank's name, one line of at most 64 characters; empty or omitted: Bank <n>"`
}

type hotbarRenameParams struct {
	Name string `json:"name" jsonschema:"the bank's new name, one line of at most 64 characters"`
	Bank int    `json:"bank,omitempty" jsonschema:"bank number as the hotbar's switcher shows it, from 1 (⇧F1); 0 or omitted: the bank shown now"`
}

// hotbarContext resolves a tile command's context: the connected one when none is given, and a
// bare context name to the id the sources give it, as cluster.connect does.
func hotbarContext(s *engine.Session, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = s.UIState().Context
		if name == "" {
			return "", errors.New("no cluster connected: name a context")
		}
		return name, nil
	}
	return s.ResolveContext(name)
}

// hotbarResult is what a hotbar change returns: the hotbar as the user sees it.
func hotbarResult(s *engine.Session) (any, error) {
	v, err := s.HotbarView()
	if err != nil {
		return nil, err
	}
	return HotbarResult{Hotbar: v}, nil
}

// prefsOnly ends the description of every command that changes the hotbar.
const prefsOnly = " Changes only Cogpit's own hotbar preferences (a file in Cogpit's configuration directory), never a cluster or a kubeconfig."

func registerHotbar(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "hotbar.list", Label: "Show Hotbar", MenuPath: "Hotbar/Show Hotbar", Danger: engine.DangerRead,
		Description: "Returns the hotbar, the bar of pinned clusters at the left edge of Cogpit's window: its banks (numbered from 1, " +
			"as the switcher shows them) with their names and tiles, top to bottom, and the bank shown now (current). Each tile has " +
			"its context id, its short label, the state of a connection to it (Cogpit holds one at a time), missing when no " +
			"kubeconfig source offers the context any more, and exec when its credential plugin needs the user's approval. " +
			"ui_state carries the same.",
		Enabled: func(s *engine.Session, _ struct{}) error { _, err := s.HotbarState(); return err },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return hotbarResult(s)
		},
	})

	add(g, engine.Spec[hotbarClusterParams]{
		ID: "hotbar.addCluster", Label: "Add to Hotbar", MenuPath: "Hotbar/Add to Hotbar", Danger: engine.DangerUI,
		Description: "Pins a cluster to the hotbar: a tile for the context at the end of a bank, which F1–F12 open by its place. " +
			"A bank holds at most 12 clusters, and a context once. The context must be one cluster_contexts lists." + prefsOnly,
		Enabled: func(s *engine.Session, p hotbarClusterParams) error {
			st, err := s.HotbarState()
			if err != nil {
				return err
			}
			id, err := hotbarContext(s, p.Context)
			if err != nil {
				return err
			}
			return st.CheckAdd(p.Bank, id)
		},
		Run: func(_ context.Context, s *engine.Session, p hotbarClusterParams) (any, error) {
			id, err := hotbarContext(s, p.Context)
			if err != nil {
				return nil, err
			}
			if err := offered(s, id); err != nil {
				return nil, err
			}
			if _, err := s.ChangeHotbar(func(h *engine.HotbarState) error { return h.AddCluster(p.Bank, id) }); err != nil {
				return nil, err
			}
			return hotbarResult(s)
		},
	})

	add(g, engine.Spec[hotbarClusterParams]{
		ID: "hotbar.removeCluster", Label: "Remove from Hotbar", MenuPath: "Hotbar/Remove from Hotbar", Danger: engine.DangerUI,
		Description: "Unpins a cluster: removes the context's tile from a bank; the tiles below it move up one place. The context " +
			"and its kubeconfig stay as they are, and a connection to it stays open." + prefsOnly,
		Enabled: func(s *engine.Session, p hotbarClusterParams) error {
			st, err := s.HotbarState()
			if err != nil {
				return err
			}
			id, err := hotbarContext(s, p.Context)
			if err != nil {
				return err
			}
			return st.CheckInBank(p.Bank, id)
		},
		Run: func(_ context.Context, s *engine.Session, p hotbarClusterParams) (any, error) {
			id, err := hotbarContext(s, p.Context)
			if err != nil {
				return nil, err
			}
			if _, err := s.ChangeHotbar(func(h *engine.HotbarState) error { return h.RemoveCluster(p.Bank, id) }); err != nil {
				return nil, err
			}
			return hotbarResult(s)
		},
	})

	add(g, engine.Spec[hotbarMoveParams]{
		ID: "hotbar.moveCluster", Label: "Move on Hotbar", MenuPath: "Hotbar/Move", Danger: engine.DangerUI,
		Description: "Moves a tile to another place in its bank (from 1, the top), which changes the F-key that opens it; the " +
			"tiles between move one place." + prefsOnly,
		Enabled: func(s *engine.Session, p hotbarMoveParams) error {
			st, err := s.HotbarState()
			if err != nil {
				return err
			}
			id, err := s.ResolveContext(p.Context)
			if err != nil {
				return err
			}
			return st.CheckMove(p.Bank, id, p.Position)
		},
		Run: func(_ context.Context, s *engine.Session, p hotbarMoveParams) (any, error) {
			id, err := s.ResolveContext(p.Context)
			if err != nil {
				return nil, err
			}
			if _, err := s.ChangeHotbar(func(h *engine.HotbarState) error { return h.MoveCluster(p.Bank, id, p.Position) }); err != nil {
				return nil, err
			}
			return hotbarResult(s)
		},
	})

	add(g, engine.Spec[hotbarSlotParams]{
		ID: "hotbar.openCluster", Label: "Open Hotbar Cluster", MenuPath: "Hotbar/Open Cluster", Shortcut: "F1–F12", Danger: engine.DangerUI,
		Description: "Opens the cluster of a hotbar tile, as clicking it or pressing its F-key does (F1 the top tile of the bank " +
			"shown): connects to its context and shows its last used tab or its Pods view, replacing the current connection, exactly as " +
			"cluster_connect does; a tile of the cluster connected already changes nothing. Connecting only reads from the " +
			"cluster. A tile whose context no kubeconfig source offers any more is disabled. Returns {context, state, error}.",
		Enabled: func(s *engine.Session, p hotbarSlotParams) error {
			_, err := openableTile(s, p)
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p hotbarSlotParams) (any, error) {
			t, err := openableTile(s, p)
			if err != nil {
				return nil, err
			}
			if u := s.UIState(); u.Context == t.Context && (u.Connection.State.Live() || u.Connection.State == engine.Connecting) {
				return ConnectResult{Context: u.Context, State: u.Connection.State, TLSUnchecked: u.Connection.TLSUnchecked}, nil
			}
			err = s.Connect(ctx, t.Context)
			u := s.UIState()
			if err != nil && u.Connection.State != engine.ConnectError {
				return nil, err // refused before connecting: no connector, closed, or superseded
			}
			return connectResult(u), nil
		},
	})

	add(g, engine.Spec[hotbarSwitchParams]{
		ID: "hotbar.switchBank", Label: "Switch Hotbar Bank", MenuPath: "Hotbar/Switch Bank", Shortcut: "Shift+F1–F12", Danger: engine.DangerUI,
		Description: "Shows another bank of the hotbar, as the switcher's arrows or ⇧F1–⇧F12 do; F1–F12 then open that bank's " +
			"clusters." + prefsOnly,
		Enabled: func(s *engine.Session, p hotbarSwitchParams) error {
			st, err := s.HotbarState()
			if err != nil {
				return err
			}
			return st.SwitchBank(p.Bank) // on a copy
		},
		Run: func(_ context.Context, s *engine.Session, p hotbarSwitchParams) (any, error) {
			if _, err := s.ChangeHotbar(func(h *engine.HotbarState) error { return h.SwitchBank(p.Bank) }); err != nil {
				return nil, err
			}
			return hotbarResult(s)
		},
	})

	add(g, engine.Spec[hotbarCreateParams]{
		ID: "hotbar.createBank", Label: "New Hotbar Bank", MenuPath: "Hotbar/New Bank", Danger: engine.DangerUI,
		Description: "Adds an empty bank after the others and shows it; the hotbar holds at most 12 banks, the ones ⇧F1–⇧F12 " +
			"reach." + prefsOnly,
		Enabled: func(s *engine.Session, _ hotbarCreateParams) error {
			st, err := s.HotbarState()
			if err != nil {
				return err
			}
			return st.CheckCreateBank()
		},
		Run: func(_ context.Context, s *engine.Session, p hotbarCreateParams) (any, error) {
			if _, err := s.ChangeHotbar(func(h *engine.HotbarState) error { _, err := h.CreateBank(p.Name); return err }); err != nil {
				return nil, err
			}
			return hotbarResult(s)
		},
	})

	add(g, engine.Spec[hotbarRenameParams]{
		ID: "hotbar.renameBank", Label: "Rename Hotbar Bank", MenuPath: "Hotbar/Rename Bank", Danger: engine.DangerUI,
		Description: "Renames a bank of the hotbar; the switcher shows the name as its tooltip." + prefsOnly,
		Enabled: func(s *engine.Session, p hotbarRenameParams) error {
			st, err := s.HotbarState()
			if err != nil {
				return err
			}
			_, err = st.Bank(p.Bank)
			return err
		},
		Run: func(_ context.Context, s *engine.Session, p hotbarRenameParams) (any, error) {
			if _, err := s.ChangeHotbar(func(h *engine.HotbarState) error { return h.RenameBank(p.Bank, p.Name) }); err != nil {
				return nil, err
			}
			return hotbarResult(s)
		},
	})

	add(g, engine.Spec[hotbarBankParams]{
		ID: "hotbar.removeBank", Label: "Remove Hotbar Bank", MenuPath: "Hotbar/Remove Bank", Danger: engine.DangerUI,
		Description: "Removes a bank of the hotbar with its tiles; the banks after it move up one number, and the hotbar keeps " +
			"at least one bank. The clusters and their kubeconfigs stay as they are. Cogpit's window asks the user first when " +
			"the bank holds clusters, and so does an agent's request; the result returns the removed bank, so it can be put " +
			"back." + prefsOnly,
		// An agent's removal of a bank with clusters waits for the user, as the window asks (A65).
		AgentsAsk: func(s *engine.Session, p hotbarBankParams) bool { return len(bankClusters(s, p.Bank)) > 0 },
		Detail: func(s *engine.Session, p hotbarBankParams) string {
			if c := bankClusters(s, p.Bank); len(c) > 0 {
				return "The bank holds " + strings.Join(c, ", ") + "; their kubeconfigs stay."
			}
			return ""
		},
		Enabled: func(s *engine.Session, p hotbarBankParams) error {
			st, err := s.HotbarState()
			if err != nil {
				return err
			}
			return st.CheckRemoveBank(p.Bank)
		},
		Run: func(_ context.Context, s *engine.Session, p hotbarBankParams) (any, error) {
			var removed engine.HotbarBank
			if _, err := s.ChangeHotbar(func(h *engine.HotbarState) (err error) { removed, err = h.RemoveBank(p.Bank); return err }); err != nil {
				return nil, err
			}
			v, err := s.HotbarView()
			if err != nil {
				return nil, err
			}
			return BankRemoved{Removed: removed, Hotbar: v}, nil
		},
	})
}

// bankClusters returns the contexts of a bank's tiles; none for a bank that does not exist.
func bankClusters(s *engine.Session, bank int) []string {
	st, err := s.HotbarState()
	if err != nil {
		return nil
	}
	b, err := st.Bank(bank)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(b.Clusters))
	for _, t := range b.Clusters {
		out = append(out, t.Context)
	}
	return out
}

// offered refuses a context that no kubeconfig source offers (and that is not the connected one):
// a tile is pinned for a context the user can connect to.
func offered(s *engine.Session, id string) error {
	if id == s.UIState().Context {
		return nil
	}
	ctxs, err := s.Contexts()
	for _, c := range ctxs {
		if c.Name == id {
			return nil
		}
	}
	if err != nil {
		return fmt.Errorf("the contexts cannot be listed: %w", err)
	}
	return fmt.Errorf("no context %s: cluster_contexts lists the ones Cogpit offers", id)
}

// openableTile is the tile hotbar.openCluster opens: it exists, and its context is offered.
func openableTile(s *engine.Session, p hotbarSlotParams) (engine.HotbarTile, error) {
	v, err := s.HotbarView()
	if err != nil {
		return engine.HotbarTile{}, err
	}
	t, err := v.Tile(p.Bank, p.Slot)
	if err != nil {
		return engine.HotbarTile{}, err
	}
	if t.Missing {
		return engine.HotbarTile{}, fmt.Errorf("%s is no longer offered by any kubeconfig source; its tile stays until it is removed", t.Context)
	}
	return t, nil
}
