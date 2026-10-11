package fyneui

// actionResults handle what an action run from the row menu or the drawer returns, by command
// id: Copy puts the name it returns on the clipboard. A command listed here gets its errors shown
// as any other; each feature registers its handler in an init of its own file.
var actionResults = map[string]func(sh *shell, res any){}

func init() { actionResults["object.copyName"] = copyName }

// actionDone returns the dispatch callback of an action: its result handler, or nil, which shows
// errors as toasts.
func (sh *shell) actionDone(id string) func(any, error) {
	h, ok := actionResults[id]
	if !ok {
		return nil
	}
	return func(res any, err error) {
		if err != nil {
			sh.showError(err)
			return
		}
		h(sh, res)
	}
}

// copyName puts the name object.copyName returned on the clipboard (Lens's drawer: Copy copies
// the resource's name) and says so.
func copyName(sh *shell, res any) {
	var r struct {
		Name string `json:"name"`
	}
	if err := decode(res, &r); err != nil || r.Name == "" {
		sh.toast("error", "Nothing to copy")
		return
	}
	sh.app.Clipboard().SetContent(r.Name)
	sh.toast("info", "Copied "+r.Name)
}
