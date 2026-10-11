package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// commandIcons are the icons of the drawer's action toolbar, by command id, from Fyne's theme.
// Lens's drawer shows its actions as icons (capture, 2026-10-11: a node's Cordon is a pause icon,
// Uncordon a play icon); the glyphs are Fyne's, chosen for Cogpit. A command without an entry gets
// genericCommandIcon; its tooltip names it either way.
var commandIcons = map[string]fyne.ThemeIconName{
	"pod.attach":          theme.IconNameMailAttachment,
	"pod.shell":           theme.IconNameComputer,
	"node.shell":          theme.IconNameComputer,
	"pod.evict":           theme.IconNameContentRemove,
	"pod.logs":            theme.IconNameList,
	"workload.logs":       theme.IconNameList,
	"pod.portForward":     theme.IconNameMailForward,
	"service.portForward": theme.IconNameMailForward,
	"portforward.open":    theme.IconNameVisibility,
	"portforward.edit":    theme.IconNameDocumentCreate,
	"portforward.stop":    theme.IconNameMediaStop,
	"portforward.remove":  theme.IconNameDelete,
	"node.cordon":         theme.IconNameMediaPause,
	"node.uncordon":       theme.IconNameMediaPlay,
	"node.drain":          theme.IconNameLogout,
	"deployment.scale":    theme.IconNameViewFullScreen,
	"statefulset.scale":   theme.IconNameViewFullScreen,
	"replicaset.scale":    theme.IconNameViewFullScreen,
	"deployment.restart":  theme.IconNameViewRefresh,
	"statefulset.restart": theme.IconNameViewRefresh,
	"daemonset.restart":   theme.IconNameViewRefresh,
	"cronjob.trigger":     theme.IconNameMediaSkipNext,
	"cronjob.suspend":     theme.IconNameMediaPause,
	"cronjob.resume":      theme.IconNameMediaPlay,
	"object.edit":         theme.IconNameDocumentCreate,
	"object.delete":       theme.IconNameDelete,
}

// genericCommandIcon is the icon of a command without its own.
const genericCommandIcon = theme.IconNameSettings

// commandIcon returns the toolbar icon of a command.
func commandIcon(id string) fyne.Resource {
	if name, ok := commandIcons[id]; ok {
		return theme.Icon(name)
	}
	return theme.Icon(genericCommandIcon)
}
