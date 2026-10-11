package fyneui

import (
	"fmt"

	"github.com/guided-traffic/cogpit/internal/model"
)

// itemActionDialogs are the GUI's own dialogs for item actions, by command id: a command listed
// here opens its dialog instead of running at once or getting the schema-generated one (the
// port-forward dialog for pod.portForward). Each feature registers its dialog in an init of its
// own file, so that features never edit a shared table.
var itemActionDialogs = map[string]func(sh *shell, ref model.ObjectRef, a model.ItemAction){}

// runItemAction runs a drawer item's action (model.ItemAction): its own dialog if one is
// registered, else as the row menu runs a command: a schema-generated dialog, prefilled with the
// action's params, when the command asks for more, else a dispatch of the params.
func (sh *shell) runItemAction(ref model.ObjectRef, a model.ItemAction) {
	if open, ok := itemActionDialogs[a.Command]; ok {
		open(sh, ref, a)
		return
	}
	cmd, ok := sh.eng.Registry().Get(a.Command)
	if !ok {
		sh.toast("error", fmt.Sprintf("%s: unknown command %q", a.Label, a.Command))
		return
	}
	params := a.Params
	if len(params) == 0 {
		params = cmd.ObjectParams(ref, "")
	}
	if len(cmd.DialogFields()) > 0 {
		sh.openParamDialog(cmd, ref, params)
		return
	}
	sh.dispatch(cmd.ID, params, nil)
}
