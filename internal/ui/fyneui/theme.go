package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/model"
)

// ThemeMode picks Cogpit's colour palette.
type ThemeMode string

// Theme modes. The zero value means ThemeDark: Cogpit starts dark, as Lens does.
const (
	ThemeDark   ThemeMode = "dark"
	ThemeLight  ThemeMode = "light"
	ThemeSystem ThemeMode = "system" // follow the operating system's light or dark setting
)

// Cogpit's own colour tokens, beyond Fyne's. Widgets look them up through the theme so that no
// widget decides a colour itself (docs/developer/conventions.md, UI).
const (
	colorTopBar       fyne.ThemeColorName = "cogpit.topBar"
	colorStatusBar    fyne.ThemeColorName = "cogpit.statusBar"
	colorSidebar      fyne.ThemeColorName = "cogpit.sidebar"
	colorRowHover     fyne.ThemeColorName = "cogpit.rowHover"
	colorRowSelected  fyne.ThemeColorName = "cogpit.rowSelected"
	colorDrawer       fyne.ThemeColorName = "cogpit.drawer"
	colorDrawerHeader fyne.ThemeColorName = "cogpit.drawerHeader"
	colorDock         fyne.ThemeColorName = "cogpit.dock"
	colorDockTab      fyne.ThemeColorName = "cogpit.dockTab"
	colorMuted        fyne.ThemeColorName = "cogpit.muted"
	colorAccent       fyne.ThemeColorName = "cogpit.accent"
	colorChip         fyne.ThemeColorName = "cogpit.chip"
	colorToast        fyne.ThemeColorName = "cogpit.toast"
	colorLink         fyne.ThemeColorName = "cogpit.link"    // a cell or drawer value that names an object
	colorTooltip      fyne.ThemeColorName = "cogpit.tooltip" // a tooltip's background
)

// Status colour tokens, one per model.Status.
const (
	colorStatusOK          fyne.ThemeColorName = "cogpit.status.ok"
	colorStatusSucceeded   fyne.ThemeColorName = "cogpit.status.succeeded"
	colorStatusPending     fyne.ThemeColorName = "cogpit.status.pending"
	colorStatusWarning     fyne.ThemeColorName = "cogpit.status.warning"
	colorStatusError       fyne.ThemeColorName = "cogpit.status.error"
	colorStatusTerminating fyne.ThemeColorName = "cogpit.status.terminating"
	colorStatusUnknown     fyne.ThemeColorName = "cogpit.status.unknown"
)

// Cogpit's own size tokens. Fyne's sizes (text, padding) are set for density in [cogTheme.Size].
const (
	sizeSidebarWidth  fyne.ThemeSizeName = "cogpit.sidebarWidth"  // the navigator's width
	sizeBarHeight     fyne.ThemeSizeName = "cogpit.barHeight"     // the top bar's height
	sizeStatusHeight  fyne.ThemeSizeName = "cogpit.statusHeight"  // the status bar's height
	sizeCellPadding   fyne.ThemeSizeName = "cogpit.cellPadding"   // horizontal padding inside a table cell
	sizeRowPadding    fyne.ThemeSizeName = "cogpit.rowPadding"    // vertical padding inside a table row
	sizeDrawerLabel   fyne.ThemeSizeName = "cogpit.drawerLabel"   // the name column of the drawer's details
	sizeChipRadius    fyne.ThemeSizeName = "cogpit.chipRadius"    // corner radius of chips and pills
	sizeSortIcon      fyne.ThemeSizeName = "cogpit.sortIcon"      // the sort arrow in a column header
	sizeDotDiameter   fyne.ThemeSizeName = "cogpit.dot"           // connection and status dots
	sizeToastWidth    fyne.ThemeSizeName = "cogpit.toastWidth"    // the width of a toast
	sizeFilterWidth   fyne.ThemeSizeName = "cogpit.filterWidth"   // the namespace filter button
	sizeSearchWidth   fyne.ThemeSizeName = "cogpit.searchWidth"   // the search entry
	sizePickerWidth   fyne.ThemeSizeName = "cogpit.pickerWidth"   // the context picker
	sizePickerHeight  fyne.ThemeSizeName = "cogpit.pickerHeight"  // the context picker
	sizeConfirmWidth  fyne.ThemeSizeName = "cogpit.confirmWidth"  // the confirmation dialog
	sizePopupMaxList  fyne.ThemeSizeName = "cogpit.popupMaxList"  // the tallest list in a popup
	sizeDockTabMaxLen fyne.ThemeSizeName = "cogpit.dockTabMaxLen" // the widest dock tab title
	sizeMenuColumn    fyne.ThemeSizeName = "cogpit.menuColumn"    // the ⋮ column at each row's end
	sizeCellIcon      fyne.ThemeSizeName = "cogpit.cellIcon"      // an icon in a table cell (the warning triangle, ⋮)
	sizeTooltipWidth  fyne.ThemeSizeName = "cogpit.tooltipWidth"  // the widest tooltip; longer text wraps
	sizeTooltipOffset fyne.ThemeSizeName = "cogpit.tooltipOffset" // a tooltip's distance from the pointer
	sizeDialogWidth   fyne.ThemeSizeName = "cogpit.dialogWidth"   // a command's parameter dialog
	sizeHealthBar     fyne.ThemeSizeName = "cogpit.healthBar"     // a health bar's height (Workloads Overview)
)

// palette is one variant's colours. Cogpit's own choice: a blue-slate dark and a cool light, with
// an indigo accent; status hues are spread so that no two tokens are confused at a glance.
type palette struct {
	fyne   map[fyne.ThemeColorName]color.Color
	status map[model.Status]color.Color
}

func hex(rgb uint32) color.NRGBA {
	return color.NRGBA{R: uint8(rgb >> 16), G: uint8(rgb >> 8), B: uint8(rgb), A: 0xff}
}

func alpha(c color.NRGBA, a uint8) color.NRGBA { c.A = a; return c }

var darkPalette = palette{
	fyne: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:          hex(0x15181e),
		theme.ColorNameButton:              hex(0x242a35),
		theme.ColorNameDisabledButton:      hex(0x1d2129),
		theme.ColorNameDisabled:            hex(0x59616f),
		theme.ColorNameError:               hex(0xf2555a),
		theme.ColorNameFocus:               alpha(hex(0x7b86ff), 0x99),
		theme.ColorNameForeground:          hex(0xd9dee7),
		theme.ColorNameForegroundOnError:   hex(0xffffff),
		theme.ColorNameForegroundOnPrimary: hex(0xffffff),
		theme.ColorNameForegroundOnSuccess: hex(0x0b1f14),
		theme.ColorNameForegroundOnWarning: hex(0x1f1405),
		theme.ColorNameHeaderBackground:    hex(0x1b1f27),
		theme.ColorNameHover:               alpha(hex(0xffffff), 0x10),
		theme.ColorNameHyperlink:           hex(0x9aa3ff),
		theme.ColorNameInputBackground:     hex(0x0f1216),
		theme.ColorNameInputBorder:         hex(0x2c323d),
		theme.ColorNameMenuBackground:      hex(0x1e222b),
		theme.ColorNameOverlayBackground:   hex(0x1a1e25),
		theme.ColorNamePlaceHolder:         hex(0x6b7383),
		theme.ColorNamePressed:             alpha(hex(0xffffff), 0x1c),
		theme.ColorNamePrimary:             hex(0x6873f2),
		theme.ColorNameScrollBar:           alpha(hex(0xffffff), 0x40),
		theme.ColorNameScrollBarBackground: alpha(hex(0x000000), 0x00),
		theme.ColorNameSelection:           hex(0x29305a),
		theme.ColorNameSeparator:           hex(0x262b34),
		theme.ColorNameShadow:              alpha(hex(0x000000), 0x66),
		theme.ColorNameSuccess:             hex(0x3dd68c),
		theme.ColorNameWarning:             hex(0xf5a524),

		colorTopBar:       hex(0x0f1217),
		colorStatusBar:    hex(0x0f1217),
		colorSidebar:      hex(0x1a1e25),
		colorRowHover:     hex(0x1e232d),
		colorRowSelected:  hex(0x29305a),
		colorDrawer:       hex(0x191d24),
		colorDrawerHeader: hex(0x20252f),
		colorDock:         hex(0x111419),
		colorDockTab:      hex(0x1d222b),
		colorMuted:        hex(0x8c95a6),
		colorAccent:       hex(0x8a94ff),
		colorChip:         hex(0x262c38),
		colorToast:        hex(0x232834),
		colorLink:         hex(0x9aa3ff),
		colorTooltip:      hex(0x2b313d),
	},
	status: map[model.Status]color.Color{
		model.StatusOK:          hex(0x3dd68c),
		model.StatusSucceeded:   hex(0x2bc4b4),
		model.StatusPending:     hex(0xf2c14e),
		model.StatusWarning:     hex(0xf28c38),
		model.StatusError:       hex(0xf2555a),
		model.StatusTerminating: hex(0xb67cf2),
		model.StatusUnknown:     hex(0x8c95a6),
	},
}

var lightPalette = palette{
	fyne: map[fyne.ThemeColorName]color.Color{
		theme.ColorNameBackground:          hex(0xffffff),
		theme.ColorNameButton:              hex(0xeceff4),
		theme.ColorNameDisabledButton:      hex(0xf1f3f6),
		theme.ColorNameDisabled:            hex(0xa3aab8),
		theme.ColorNameError:               hex(0xd93a3f),
		theme.ColorNameFocus:               alpha(hex(0x4f5bd5), 0x88),
		theme.ColorNameForeground:          hex(0x1e2430),
		theme.ColorNameForegroundOnError:   hex(0xffffff),
		theme.ColorNameForegroundOnPrimary: hex(0xffffff),
		theme.ColorNameForegroundOnSuccess: hex(0xffffff),
		theme.ColorNameForegroundOnWarning: hex(0x1f1405),
		theme.ColorNameHeaderBackground:    hex(0xf5f7fa),
		theme.ColorNameHover:               alpha(hex(0x000000), 0x0c),
		theme.ColorNameHyperlink:           hex(0x4f5bd5),
		theme.ColorNameInputBackground:     hex(0xffffff),
		theme.ColorNameInputBorder:         hex(0xcdd3dd),
		theme.ColorNameMenuBackground:      hex(0xffffff),
		theme.ColorNameOverlayBackground:   hex(0xffffff),
		theme.ColorNamePlaceHolder:         hex(0x8a92a3),
		theme.ColorNamePressed:             alpha(hex(0x000000), 0x17),
		theme.ColorNamePrimary:             hex(0x4f5bd5),
		theme.ColorNameScrollBar:           alpha(hex(0x000000), 0x40),
		theme.ColorNameScrollBarBackground: alpha(hex(0x000000), 0x00),
		theme.ColorNameSelection:           hex(0xdfe3ff),
		theme.ColorNameSeparator:           hex(0xe3e7ee),
		theme.ColorNameShadow:              alpha(hex(0x000000), 0x2e),
		theme.ColorNameSuccess:             hex(0x1f9d5c),
		theme.ColorNameWarning:             hex(0xc77700),

		colorTopBar:       hex(0xeceff4),
		colorStatusBar:    hex(0xeceff4),
		colorSidebar:      hex(0xf4f6f9),
		colorRowHover:     hex(0xf1f4f9),
		colorRowSelected:  hex(0xe2e6ff),
		colorDrawer:       hex(0xfbfcfd),
		colorDrawerHeader: hex(0xf0f2f6),
		colorDock:         hex(0xf6f7f9),
		colorDockTab:      hex(0xffffff),
		colorMuted:        hex(0x5c6578),
		colorAccent:       hex(0x4f5bd5),
		colorChip:         hex(0xeceff4),
		colorToast:        hex(0xffffff),
		colorLink:         hex(0x4f5bd5),
		colorTooltip:      hex(0xfdfdfe),
	},
	status: map[model.Status]color.Color{
		model.StatusOK:          hex(0x1f9d5c),
		model.StatusSucceeded:   hex(0x0f8f86),
		model.StatusPending:     hex(0xb88400),
		model.StatusWarning:     hex(0xd2691e),
		model.StatusError:       hex(0xd93a3f),
		model.StatusTerminating: hex(0x8a4fd8),
		model.StatusUnknown:     hex(0x6b7383),
	},
}

// init files each palette's status colours under their theme names, so that Color is one lookup.
func init() {
	for _, p := range []*palette{&darkPalette, &lightPalette} {
		for s, name := range statusColorNames {
			p.fyne[name] = p.status[s]
		}
	}
}

var statusColorNames = map[model.Status]fyne.ThemeColorName{
	model.StatusOK:          colorStatusOK,
	model.StatusSucceeded:   colorStatusSucceeded,
	model.StatusPending:     colorStatusPending,
	model.StatusWarning:     colorStatusWarning,
	model.StatusError:       colorStatusError,
	model.StatusTerminating: colorStatusTerminating,
	model.StatusUnknown:     colorStatusUnknown,
}

// statusColorName returns the theme colour name of a status token; StatusNone and unknown tokens
// map to the foreground, so that an unexpected token never hides text.
func statusColorName(s model.Status) fyne.ThemeColorName {
	if n, ok := statusColorNames[s]; ok {
		return n
	}
	return theme.ColorNameForeground
}

// StatusColor returns the colour of a status token in the given variant: ok green, succeeded teal,
// pending amber, warning orange, error red, terminating violet, unknown grey. StatusNone returns
// the variant's foreground colour.
func StatusColor(s model.Status, v fyne.ThemeVariant) color.Color {
	p := paletteFor(v)
	if c, ok := p.status[s]; ok {
		return c
	}
	return p.fyne[theme.ColorNameForeground]
}

func paletteFor(v fyne.ThemeVariant) *palette {
	if v == theme.VariantLight {
		return &lightPalette
	}
	return &darkPalette
}

// cogTheme is Cogpit's fyne.Theme: its own palettes and a denser size set on top of Fyne's fonts
// and icons.
type cogTheme struct {
	mode ThemeMode
	base fyne.Theme
}

var _ fyne.Theme = (*cogTheme)(nil)

// newTheme returns the theme for a mode; the zero mode is dark.
func newTheme(mode ThemeMode) *cogTheme {
	if mode == "" {
		mode = ThemeDark
	}
	return &cogTheme{mode: mode, base: theme.DefaultTheme()}
}

// variant resolves the mode against the variant Fyne asks for: only ThemeSystem follows it.
func (t *cogTheme) variant(v fyne.ThemeVariant) fyne.ThemeVariant {
	switch t.mode {
	case ThemeLight:
		return theme.VariantLight
	case ThemeSystem:
		return v
	}
	return theme.VariantDark
}

// Color implements fyne.Theme.
func (t *cogTheme) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	v = t.variant(v)
	if c, ok := paletteFor(v).fyne[n]; ok {
		return c
	}
	return t.base.Color(n, v)
}

// Font implements fyne.Theme.
func (t *cogTheme) Font(s fyne.TextStyle) fyne.Resource { return t.base.Font(s) }

// iconChevronDown is Cogpit's own open-branch chevron, matching Fyne's closed-branch chevron;
// Fyne's tree draws an arrow (MoveDown) there, which reads as "move", not "expanded".
var iconChevronDown = fyne.NewStaticResource("cogpit-chevron-down.svg", []byte(
	`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">`+
		`<path d="M7.41 8.59 12 13.17l4.59-4.58L18 10l-6 6-6-6z"/></svg>`))

// Icon implements fyne.Theme.
func (t *cogTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	if n == theme.IconNameMoveDown {
		return theme.NewThemedResource(iconChevronDown)
	}
	return t.base.Icon(n)
}

// Size implements fyne.Theme. Text is a step smaller than Fyne's default and paddings are tighter:
// a Kubernetes IDE shows many rows, and Lens is compact too.
func (t *cogTheme) Size(n fyne.ThemeSizeName) float32 {
	switch n {
	case theme.SizeNameText:
		return 13
	case theme.SizeNameCaptionText:
		return 11
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameHeadingText:
		return 20
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 6
	case theme.SizeNameLineSpacing:
		return 3
	case theme.SizeNameInlineIcon:
		return 16
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 3
	case theme.SizeNameInputRadius, theme.SizeNameButtonRadius:
		return 5
	case theme.SizeNameSelectionRadius:
		return 4
	case theme.SizeNameDialogRadius:
		return 10
	case theme.SizeNameSplitThickness:
		return 4

	case sizeSidebarWidth:
		return 216
	case sizeBarHeight:
		return 40
	case sizeStatusHeight:
		return 24
	case sizeCellPadding:
		return 10
	case sizeRowPadding:
		return 6
	case sizeDrawerLabel:
		return 128
	case sizeChipRadius:
		return 9
	case sizeSortIcon:
		return 14
	case sizeDotDiameter:
		return 8
	case sizeToastWidth:
		return 360
	case sizeFilterWidth:
		return 190
	case sizeSearchWidth:
		return 240
	case sizePickerWidth:
		return 600
	case sizePickerHeight:
		return 460
	case sizeConfirmWidth:
		return 500
	case sizePopupMaxList:
		return 320
	case sizeDockTabMaxLen:
		return 220
	case sizeMenuColumn:
		return 30
	case sizeCellIcon:
		return 14
	case sizeTooltipWidth:
		return 440
	case sizeTooltipOffset:
		return 14
	case sizeDialogWidth:
		return 420
	case sizeHealthBar:
		return 6
	}
	return t.base.Size(n)
}
