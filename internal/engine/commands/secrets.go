package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// kindSecrets is the kind key of core Secrets.
const kindSecrets = "secrets"

// errRevealGUIOnly refuses secret.reveal for every origin but the user's clicks in the GUI.
var errRevealGUIOnly = errors.New("a Secret's values are shown only to the user, in Cogpit's own window; " +
	"agents and the CLI never receive them")

func registerSecrets(g *registrar) {
	add(g, engine.Spec[engine.RevealParams]{
		ID: engine.RevealCommand, Label: "Show Value", Kinds: []string{kindSecrets}, Danger: engine.DangerRead,
		Verbs: []engine.Verb{{Verb: "get", Resource: kindSecrets}},
		Description: "Shows the user one value of the Secret open in the drawer, in Cogpit's own window (the drawer lists " +
			"the keys with their values masked). Only the user's click in the GUI runs it: agents and the CLI are refused " +
			"and never receive a Secret's value. Reads nothing from the cluster: the drawer fetched the Secret when it opened.",
		Enabled: func(s *engine.Session, p engine.RevealParams) error {
			if p.Ref.Kind != kindSecrets {
				return fmt.Errorf("%w: this action applies to %s", engine.ErrNotApplicable, kindSecrets)
			}
			return s.SecretKeyHeld(p.Ref, p.Key)
		},
		Run: func(ctx context.Context, s *engine.Session, p engine.RevealParams) (any, error) {
			if engine.OriginOf(ctx) != engine.OriginGUI {
				return nil, errRevealGUIOnly
			}
			return s.SecretValue(p.Ref, p.Key)
		},
	})
}
