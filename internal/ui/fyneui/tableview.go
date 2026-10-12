package fyneui

import (
	"image/color"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Default column widths by column id, in Fyne units, for columns whose profile gives no width
// hint. They are data about Kubernetes columns (an age is short, a message is long), so they sit
// next to the table and not in the theme.
var defaultColumnWidths = map[string]float32{
	"name": 250, "namespace": 130, "containers": 92, "cpu": 80, "memory": 86, "restarts": 76,
	"controlledBy": 130, "node": 120, "qos": 96, "age": 56, "status": 130, "conditions": 240,
	"pods": 72, "replicas": 86, "type": 100, "clusterIP": 120, "ports": 140, "externalIP": 120,
	"selector": 200, "labels": 260, "message": 300, "involvedObject": 210, "source": 140,
	"count": 60, "lastSeen": 172, "roles": 120, "version": 104, "taints": 64, "disk": 76,
}

var defaultWidthByType = map[model.ColumnType]float32{
	model.ColumnText: 140, model.ColumnNumber: 84, model.ColumnAge: 64,
}

// tableView is the centre of the window: the view header (title and count, namespace filter,
// search) above the resource table, or a placeholder while there is nothing to show.
type tableView struct {
	sh *shell

	title, count, syncing *textLine
	csvButton             *widget.Button // beside the count: download the view as CSV (tableheader.go)
	errLine               *textLine
	nsButton              *widget.Button
	nsBox                 *fyne.Container // the namespace filter's slot; hidden on cluster-scoped views
	search                *widget.Entry
	searchBox             *fyne.Container
	heading, tools        *fyne.Container // the header's left and right parts

	// A page (the Workloads Overview) shows its own part between the header and the table: the
	// kinds' tiles and the events' header, which then holds the count and the search.
	overview                   *overviewPanel
	eventsTitle                *textLine
	eventsHeading, eventsTools *fyne.Container
	pageBox                    *fyne.Container
	pageMode                   bool
	table                      *resourceTable
	placeholder                *fyne.Container
	placeholderText            *textLine
	connectButton              *widget.Button
	create                     *createButton // the floating "+" (createbutton.go)
	root                       fyne.CanvasObject

	snap    *engine.TableSnapshot
	cols    []int  // visible column → index in snap.Columns
	colsKey string // the view and column ids, to notice a column change
	now     time.Time

	cells      []*tableCell // every cell the table has shown, pooled ones included
	hoverRow   int
	hoverCol   int
	kbOn       bool // the table has the keyboard focus (tablekeys.go)
	kbRow      int  // the cell the keyboard is on
	kbCol      int
	hoverCell  *tableCell    // the cell under the pointer or the keyboard highlight
	pointer    fyne.Position // the pointer over the table, in window coordinates (tooltips)
	pointerIn  bool
	icons      map[cellIconKey]fyne.Resource // the cells' icons, one per icon and colour
	menu       *widget.PopUpMenu             // the row menu last opened
	menuModel  *fyne.Menu                    // and its items
	kindSet    map[string]bool               // the kinds links may lead to (serves)
	kindSetSrc *kindInfo
	kindSetLen int
	selected   []model.ObjectRef
	pending    *model.ObjectRef // a selected object to scroll to once a snapshot contains it

	userWidths map[string]float32 // widths the user dragged, by view and column id
	setWidths  []float32          // the width the shell last gave each visible column
	fitWidth   float32            // the frame width the columns were last fitted to
	fitting    bool               // set while fitColumns sets widths, which are not the user's

	searchTimer *time.Timer
	filterBusy  int  // filter changes typed but not yet acknowledged by the engine
	syncing2    bool // set while the search entry is updated from engine state

	nsPopup  *widget.PopUp
	nsChecks []*widget.Check
	nsSync   bool

	colPopup  *widget.PopUp // the column list of the header's ⋮ (tableheader.go)
	colChecks []*widget.Check
	colIDs    []string // the column id of each check
	colView   string   // the view the list was opened for
	colSync   bool
}

func newTableView(sh *shell) *tableView {
	tv := &tableView{sh: sh, hoverRow: -1, hoverCol: -1, userWidths: map[string]float32{}, icons: map[cellIconKey]fyne.Resource{}}

	tv.title = newTextLine("", theme.ColorNameForeground, theme.SizeNameSubHeadingText, fyne.TextStyle{Bold: true})
	tv.count = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	tv.syncing = newTextLine("", colorAccent, theme.SizeNameCaptionText, fyne.TextStyle{})
	tv.errLine = newTextLine("", colorStatusError, theme.SizeNameCaptionText, fyne.TextStyle{})
	tv.errLine.Hide()

	tv.nsButton = widget.NewButtonWithIcon("All namespaces", theme.MenuDropDownIcon(), tv.openNamespaces)
	tv.nsButton.IconPlacement = widget.ButtonIconTrailingText
	tv.nsButton.Alignment = widget.ButtonAlignLeading
	tv.search = widget.NewEntry()
	tv.search.SetPlaceHolder("Search…")
	tv.search.SetIcon(theme.SearchIcon())
	tv.search.OnChanged = tv.searchChanged

	tv.table = newResourceTable(tv)

	tv.placeholderText = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	tv.connectButton = widget.NewButtonWithIcon("Connect to a cluster…", theme.LoginIcon(), func() { sh.picker.show() })
	tv.connectButton.Importance = widget.HighImportance
	tv.placeholder = container.NewCenter(container.NewVBox(
		container.NewCenter(tv.placeholderText), container.NewCenter(tv.connectButton)))

	tv.csvButton = widget.NewButtonWithIcon("", theme.DownloadIcon(), tv.exportCSV)
	tv.csvButton.Importance = widget.LowImportance
	tv.csvButton.Hide()
	tv.heading = container.NewHBox(tv.title, tv.count, tv.csvButton, tv.syncing)
	tv.nsBox = sized(sizeFilterWidth, "", tv.nsButton)
	tv.searchBox = sized(sizeSearchWidth, "", tv.search)
	tv.tools = container.NewHBox(tv.nsBox, tv.searchBox)
	header := container.NewStack(newSurface(theme.ColorNameBackground),
		inset(theme.SizeNameInnerPadding, theme.SizeNamePadding,
			container.NewBorder(nil, nil, tv.heading, tv.tools)))

	tv.overview = newOverviewPanel(sh)
	tv.eventsTitle = newTextLine("Events", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	tv.eventsHeading, tv.eventsTools = container.NewHBox(), container.NewHBox()
	tv.pageBox = container.NewVBox(tv.overview.root, newDivider(false),
		container.NewStack(newSurface(theme.ColorNameBackground),
			inset(theme.SizeNameInnerPadding, theme.SizeNamePadding,
				container.NewBorder(nil, nil, tv.eventsHeading, tv.eventsTools))))
	tv.pageBox.Hide()
	frame := box(&stretchLayout{tv: tv}, tv.table)
	tv.create = newCreateButton(sh)
	body := container.NewStack(newSurface(theme.ColorNameBackground), frame, tv.placeholder, tv.create.floating())
	tv.root = container.NewBorder(
		container.NewVBox(header, tv.pageBox, inset(theme.SizeNameInnerPadding, "", tv.errLine)),
		nil, nil, nil, body)
	tv.updatePlaceholder()
	return tv
}

// setPageLayout arranges the header for a page or a kind's view. On the Workloads Overview the
// header holds the page's title and the namespace filter, which applies to the whole page; the
// tiles follow, then the events' own header with their count, download button and search.
func (tv *tableView) setPageLayout(page bool) {
	if page == tv.pageMode {
		return
	}
	tv.pageMode = page
	if page {
		tv.heading.Objects = []fyne.CanvasObject{tv.title}
		tv.tools.Objects = []fyne.CanvasObject{tv.nsBox}
		tv.eventsHeading.Objects = []fyne.CanvasObject{tv.eventsTitle, tv.count, tv.csvButton, tv.syncing}
		tv.eventsTools.Objects = []fyne.CanvasObject{tv.searchBox}
		tv.pageBox.Show()
	} else {
		tv.eventsHeading.Objects, tv.eventsTools.Objects = nil, nil
		tv.heading.Objects = []fyne.CanvasObject{tv.title, tv.count, tv.csvButton, tv.syncing}
		tv.tools.Objects = []fyne.CanvasObject{tv.nsBox, tv.searchBox}
		tv.pageBox.Hide()
	}
	for _, c := range []*fyne.Container{tv.heading, tv.tools, tv.eventsHeading, tv.eventsTools} {
		c.Refresh()
	}
}

// viewChanged follows a change of the open view: titles, placeholder, search hint.
func (tv *tableView) viewChanged() {
	tv.pending = nil
	tv.hoverRow, tv.hoverCol, tv.hoverCell = -1, -1, nil
	tv.sh.tip.hide()
	page := engine.IsPage(tv.sh.ui.View)
	tv.setPageLayout(page)
	if page {
		tv.overview.request()
	}
	tv.updateTitle()
	tv.updatePlaceholder()
	tv.updateNamespaceButton()
}

// rowNoun names the open view's rows in hints: "pods"; a page's table lists events.
func (tv *tableView) rowNoun() string {
	if engine.IsPage(tv.sh.ui.View) {
		return kindEvents // the kind key reads as the noun
	}
	return strings.ToLower(tv.sh.kindTitle(tv.sh.ui.View))
}

// updateTitle shows the view's title and item count, "Pods 18 items", and names the view in the
// search hint; kinds.list may bring the title after the view opened.
func (tv *tableView) updateTitle() {
	tv.search.SetPlaceHolder("Search " + tv.rowNoun() + "…")
	title := tv.sh.kindTitle(tv.sh.ui.View)
	if tv.snap != nil && tv.snap.Title != "" {
		title = tv.snap.Title
	}
	tv.title.SetText(title)
	if tv.snap == nil {
		tv.count.SetText("")
		tv.syncing.SetText("")
		tv.csvButton.Hide()
		return
	}
	tv.count.SetText(itemCount(len(tv.snap.Rows), tv.snap.Total))
	tv.csvButton.Show()
	if tv.snap.Synced {
		tv.syncing.SetText("")
	} else {
		tv.syncing.SetText("syncing…")
	}
}

// updatePlaceholder shows the placeholder instead of the table when there is nothing to show.
func (tv *tableView) updatePlaceholder() {
	ui := tv.sh.ui
	msg := ""
	connect := false
	switch {
	case ui.Connection.State == engine.Connecting:
		msg = "Connecting to " + ui.Context + "…"
	case ui.Connection.State != engine.Connected:
		msg, connect = "No cluster connected.", true
		if ui.Connection.State == engine.ConnectError && ui.Connection.Error != "" {
			msg = "Could not connect to " + ui.Context + ": " + ui.Connection.Error
		}
	case ui.View == "":
		msg = "Choose a view in the sidebar."
	case tv.snap == nil || (!tv.snap.Synced && len(tv.snap.Rows) == 0 && tv.snap.Error == ""):
		msg = "Loading " + tv.rowNoun() + "…"
	case len(tv.snap.Rows) == 0 && (tv.snap.Filter != "" || len(tv.snap.Namespaces) > 0 || (tv.groupMode() && len(ui.Groups) > 0)):
		msg = "No " + tv.rowNoun() + " match the filter."
	case len(tv.snap.Rows) == 0:
		msg = "No " + tv.rowNoun() + " found."
	}
	tv.placeholderText.SetText(msg)
	if connect {
		tv.connectButton.Show()
	} else {
		tv.connectButton.Hide()
	}
	if msg == "" {
		tv.placeholder.Hide()
	} else {
		tv.placeholder.Show()
	}
	tv.create.update()
}

// setSnapshot shows a new snapshot. A replaced snapshot costs one table refresh: the table asks
// for the visible cells only, and each cell update is O(1).
func (tv *tableView) setSnapshot(snap *engine.TableSnapshot) {
	if snap == tv.snap && snap != nil {
		return
	}
	newView := snap != nil && (tv.snap == nil || tv.snap.View != snap.View)
	tv.snap = snap
	key := ""
	if snap != nil {
		var b strings.Builder
		b.WriteString(snap.View)
		for _, c := range snap.Columns {
			b.WriteByte('|')
			b.WriteString(c.ID)
			if c.Hidden {
				b.WriteByte('-')
			}
		}
		key = b.String()
	}
	if key != tv.colsKey {
		tv.colsKey = key
		tv.cols = tv.cols[:0]
		if snap != nil {
			for i, c := range snap.Columns {
				if !c.Hidden {
					tv.cols = append(tv.cols, i)
				}
			}
		}
		tv.setWidths = nil
		tv.fitWidth = 0
		tv.fitColumns(tv.table.Size().Width)
	}
	if snap != nil && snap.Error != "" {
		tv.errLine.SetText(snap.Error)
		tv.errLine.Show()
	} else {
		tv.errLine.Hide()
	}
	tv.now = tv.sh.now()
	tv.updateTitle()
	tv.updatePlaceholder()
	tv.updateNamespaceButton() // cluster-scoped views hide the namespace filter
	tv.syncColumnChecks()
	if newView {
		tv.table.resetKeyboard() // a view starts at its first cell, scrolled to the top
	}
	tv.table.Refresh()
	tv.resolvePending()
}

// refreshAges re-renders the visible cells so that age columns advance.
func (tv *tableView) refreshAges() {
	if tv.snap == nil {
		return
	}
	tv.now = tv.sh.now()
	tv.table.Refresh()
}

// stateChanged follows namespace, filter and sort changes made anywhere (the user, an agent).
func (tv *tableView) stateChanged() {
	tv.syncSearch()
	tv.updateNamespaceButton()
	tv.syncNamespaceChecks()
}

// selectionChanged re-renders the selected rows and remembers the first selected object as a
// scroll target: an agent's ui.navigate selects an object that may be far down the table, or in
// a view whose snapshot has not arrived yet.
func (tv *tableView) selectionChanged(sel []model.ObjectRef) {
	tv.selected = slices.Clone(sel)
	if len(sel) > 0 {
		ref := sel[0]
		tv.pending = &ref
	} else {
		tv.pending = nil
	}
	for _, c := range tv.cells {
		c.setSelected(tv.rowSelected(c.row))
	}
	tv.resolvePending()
}

// resolvePending scrolls to the pending object once the shown snapshot contains it.
func (tv *tableView) resolvePending() {
	if tv.pending == nil || tv.snap == nil || tv.snap.View != tv.pending.Kind {
		return
	}
	i := tv.snap.IndexOf(*tv.pending)
	if i < 0 {
		return
	}
	tv.pending = nil
	tv.scrollTo(i)
}

// scrollTo scrolls a row into view and refreshes the cells: ScrollTo alone moves the offset but
// leaves the old cells in place until the next refresh.
func (tv *tableView) scrollTo(row int) {
	tv.layoutTable()
	tv.table.ScrollTo(widget.TableCellID{Row: row, Col: 0})
	tv.table.Refresh()
}

// layoutTable lays the table out for its current row count. Fyne 2.8 lays out a table's scroller
// only when the table is resized, and the scroller clamps its offset against its content's laid-out
// size: right after the row count grew (a new view, a cleared filter), ScrollTo sets the table's
// offset to the row but the scroller clamps its own to the old, smaller content and the cells are
// drawn outside the viewport. Resizing by a point and back forces the layout.
func (tv *tableView) layoutTable() {
	sz := tv.table.Size()
	if sz.IsZero() {
		return
	}
	tv.table.Resize(sz.Add(fyne.NewSize(0, 1)))
	tv.table.Resize(sz)
}

func (tv *tableView) rowSelected(row int) bool {
	if tv.snap == nil || row < 0 || row >= len(tv.snap.Rows) || len(tv.selected) == 0 {
		return false
	}
	ref := tv.snap.Rows[row].Ref
	for _, s := range tv.selected {
		if s.Same(ref) {
			return true
		}
	}
	return false
}

// length, createCell and updateCell are the table's data callbacks. The table has one column
// more than the view: the ⋮ that opens each row's actions menu.
func (tv *tableView) length() (int, int) {
	if tv.snap == nil {
		return 0, len(tv.cols)
	}
	return len(tv.snap.Rows), len(tv.cols) + 1
}

// menuColumn is the table column of the ⋮.
func (tv *tableView) menuColumn() int { return len(tv.cols) }

func (tv *tableView) createCell() fyne.CanvasObject { return newTableCell(tv) }

// updateCell fills one visible cell. It is O(1): an index into the snapshot, an age formatted
// against the refresh's clock, and text and colour set on canvas objects the cell reuses.
func (tv *tableView) updateCell(id widget.TableCellID, o fyne.CanvasObject) {
	c := o.(*tableCell)
	if !c.tracked {
		// Cells are registered on their first update, not on creation: Table.Refresh creates a
		// template cell to measure each time, and those are never shown or updated.
		c.tracked = true
		tv.cells = append(tv.cells, c)
	}
	if tv.snap == nil || id.Row < 0 || id.Row >= len(tv.snap.Rows) || id.Col < 0 || id.Col > len(tv.cols) {
		c.set(id.Row, id.Col, cellContent{}, false, false, false)
		return
	}
	var content cellContent
	if id.Col == tv.menuColumn() {
		content.kind = cellMenu
	} else {
		row := &tv.snap.Rows[id.Row]
		ci := tv.cols[id.Col]
		col := &tv.snap.Columns[ci]
		var cell model.Cell
		if ci < len(row.Cells) {
			cell = row.Cells[ci]
		}
		status := cell.Status
		if status == model.StatusNone && col.ID == "status" {
			status = tv.snap.Rows[id.Row].Status
		}
		content = cellContent{text: model.CellText(*col, cell, tv.now), status: status, link: cell.Link, parts: cell.Parts}
		if content.link != nil && !tv.serves(content.link.Kind) {
			content.link = nil // nowhere to go: plain text
		}
		if col.Type == model.ColumnWarning {
			content.kind = cellWarning
		}
	}
	hover := id.Row == tv.hoverRow
	if c.set(id.Row, id.Col, content, tv.rowSelected(id.Row), hover, hover && id.Col == tv.hoverCol) && tv.sh.tip.owner == c {
		tv.sh.tip.hide() // the cell under the tooltip shows something else now
	}
	c.setFocused(tv.keyboardOn(id.Row, id.Col))
}

func (tv *tableView) createHeader() fyne.CanvasObject {
	return newHeaderCell(tv.headerTapped)
}

// updateHeader fills a column header and notices widths the user dragged: Fyne has no column
// resize callback, but it resizes the header before updating it.
func (tv *tableView) updateHeader(id widget.TableCellID, o fyne.CanvasObject) {
	h := o.(*headerCell)
	if id.Row != -1 || tv.snap == nil || id.Col < 0 || id.Col >= len(tv.cols) {
		// The ⋮ column's header is the ⋮ that shows and hides columns.
		h.setMenu(id.Col, id.Row == -1 && tv.snap != nil && id.Col == tv.menuColumn())
		return
	}
	col := tv.snap.Columns[tv.cols[id.Col]]
	sort := 0
	if s := tv.snap.Sort; s.Column == col.ID {
		sort = 1
		if s.Desc {
			sort = -1
		}
	}
	title := col.Title
	if col.Type == model.ColumnWarning {
		title = "" // Lens's warning column has no title; Title names it for agents
	}
	h.set(id.Col, title, sort)
	if !tv.fitting && id.Col < len(tv.setWidths) && id.Col < len(tv.cols) {
		if w := o.Size().Width; w > 0 && w != tv.setWidths[id.Col] {
			tv.setWidths[id.Col] = w
			tv.userWidths[tv.snap.View+"/"+col.ID] = w
		}
	}
}

// headerTapped sorts by a column; tapping the sorted column again reverses the order. The ⋮ at the
// end of the header row opens the column list.
func (tv *tableView) headerTapped(col int) {
	if tv.snap != nil && col == tv.menuColumn() {
		tv.openColumns()
		return
	}
	if tv.snap == nil || col < 0 || col >= len(tv.cols) {
		return
	}
	id := tv.snap.Columns[tv.cols[col]].ID
	cur := tv.sh.ui.Sort
	tv.sh.dispatch("ui.sort", sortParams{Column: id, Desc: cur.Column == id && !cur.Desc}, nil)
}

// rowTapped selects a row's object, which opens the drawer.
func (tv *tableView) rowTapped(id widget.TableCellID) {
	tv.table.UnselectAll() // the engine holds the selection; Fyne's own would block a second tap
	if tv.snap == nil || id.Row < 0 || id.Row >= len(tv.snap.Rows) {
		return
	}
	tv.setHoverCell(id.Row, id.Col)
	tv.sh.tip.hide()
	ref := tv.snap.Rows[id.Row].Ref
	tv.sh.dispatch("ui.select", selectParams{Refs: []model.ObjectRef{ref}}, nil)
}

// setHover moves the row highlight that follows the pointer and the keyboard.
func (tv *tableView) setHover(row int) { tv.setHoverCell(row, -1) }

// setHoverCell moves the row highlight and the hovered cell: a link underlines, and a cell that
// is cut off or a warning icon asks for its tooltip. It visits the cells the table has made (the
// visible ones and its pool), never the rows.
func (tv *tableView) setHoverCell(row, col int) {
	if row == tv.hoverRow && col == tv.hoverCol {
		return
	}
	oldRow := tv.hoverRow
	tv.hoverRow, tv.hoverCol = row, col
	for _, c := range tv.cells {
		if c.row == oldRow || c.row == row {
			c.setHover(c.row == row, c.row == row && c.col == col)
		}
	}
	tv.hoverCell = tv.cellUnderPointer(row, col)
	tv.requestTip()
}

// cellUnderPointer returns the cell of a row and column that the pointer is on. The table's pool
// keeps cells with the ids they last showed, so the id alone may name a cell that is not on
// screen; the pointer decides. Only the cells with that id are measured.
func (tv *tableView) cellUnderPointer(row, col int) *tableCell {
	if !tv.pointerIn || row < 0 {
		return nil
	}
	drv := fyne.CurrentApp().Driver()
	for _, c := range tv.cells {
		if c.row != row || c.col != col || !c.Visible() {
			continue
		}
		p, s := drv.AbsolutePositionForObject(c), c.Size()
		if tv.pointer.X >= p.X && tv.pointer.X < p.X+s.Width && tv.pointer.Y >= p.Y && tv.pointer.Y < p.Y+s.Height {
			return c
		}
	}
	return nil
}

// requestTip asks for the tooltip of the cell under the pointer, or hides the tooltip.
func (tv *tableView) requestTip() {
	if tv.hoverCell == nil || !tv.pointerIn {
		tv.sh.tip.hide()
		return
	}
	text := tv.hoverCell.tipText()
	if text == "" {
		tv.sh.tip.hide() // the pointer left the cell with the tooltip for one without
		return
	}
	tv.sh.tip.hover(tv.hoverCell, text, tv.pointer)
}

// cellTapped handles a click on a body cell: the ⋮ opens the row's actions menu below it, a link
// is followed (and the row is not selected), any other click selects the row through the table.
func (tv *tableView) cellTapped(c *tableCell, e *fyne.PointEvent) {
	tv.sh.tip.hide()
	if tv.snap == nil || c.row < 0 || c.row >= len(tv.snap.Rows) {
		return
	}
	drv := fyne.CurrentApp().Driver()
	switch {
	case c.content.kind == cellMenu:
		tv.openRowMenu(c.row, drv.AbsolutePositionForObject(c).AddXY(0, c.Size().Height))
	case c.onLink(e.Position):
		nsColumn := c.col < len(tv.cols) && tv.snap.Columns[tv.cols[c.col]].ID == "namespace"
		tv.sh.followLink(*c.content.link, tv.snap.Rows[c.row].Ref.Context, nsColumn)
	default:
		if cnv := drv.CanvasForObject(tv.table); cnv != nil {
			cnv.Focus(tv.table)
		}
		tv.table.Select(widget.TableCellID{Row: c.row, Col: c.col})
	}
}

// columnWidth is a visible column's width: what the user dragged, else the profile's hint, else
// the default for the column id or type.
func (tv *tableView) columnWidth(i int) float32 {
	col := tv.snap.Columns[tv.cols[i]]
	if w, ok := tv.userWidths[tv.snap.View+"/"+col.ID]; ok {
		return w
	}
	if col.Width > 0 {
		return col.Width
	}
	if w, ok := defaultColumnWidths[col.ID]; ok {
		return w
	}
	if w, ok := defaultWidthByType[col.Type]; ok {
		return w
	}
	return defaultWidthByType[model.ColumnText]
}

// flexColumn is the visible column that takes the free width: the name, as in Lens; for views
// without a visible name (Events hides it) the message; else the first column.
func (tv *tableView) flexColumn() int {
	for _, id := range []string{"name", "message"} {
		for i, ci := range tv.cols {
			if tv.snap.Columns[ci].ID == id {
				return i
			}
		}
	}
	return 0
}

// fitColumns gives every column its width and stretches the flex column to fill the frame, as
// Lens does, unless the user sized it.
func (tv *tableView) fitColumns(width float32) {
	if tv.snap == nil || len(tv.cols) == 0 || width <= 0 {
		return
	}
	if width == tv.fitWidth && len(tv.setWidths) == len(tv.cols)+1 {
		return
	}
	tv.fitWidth = width
	th := tv.table.Theme()
	gap := th.Size(theme.SizeNamePadding)
	flex := tv.flexColumn()
	widths := make([]float32, len(tv.cols)+1)
	widths[len(tv.cols)] = th.Size(sizeMenuColumn)
	others := widths[len(tv.cols)] + gap
	for i := range tv.cols {
		widths[i] = tv.columnWidth(i)
		if i != flex {
			others += widths[i] + gap
		}
	}
	col := tv.snap.Columns[tv.cols[flex]]
	if _, user := tv.userWidths[tv.snap.View+"/"+col.ID]; !user {
		if fill := width - others - th.Size(theme.SizeNameScrollBar); fill > widths[flex] {
			widths[flex] = fill
		}
	}
	if len(tv.setWidths) != len(widths) {
		tv.setWidths = make([]float32, len(widths))
	}
	// Each SetColumnWidth refreshes the table, and the header updates in between see columns
	// that are not set yet; fitting tells updateHeader that none of this is the user dragging.
	tv.fitting = true
	defer func() { tv.fitting = false }()
	for i, w := range widths {
		if tv.setWidths[i] != w {
			tv.setWidths[i] = w
			tv.table.SetColumnWidth(i, w)
		}
	}
}

// stretchLayout lays the table out at full size and refits the columns when its width changes.
type stretchLayout struct{ tv *tableView }

func (l *stretchLayout) MinSize(objs []fyne.CanvasObject) fyne.Size { return objs[0].MinSize() }

func (l *stretchLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	objs[0].Move(fyne.NewPos(0, 0))
	objs[0].Resize(size)
	l.tv.fitColumns(size.Width)
}

// searchChanged debounces typing: the filter goes to the engine once the user pauses.
func (tv *tableView) searchChanged(text string) {
	if tv.syncing2 {
		return
	}
	if tv.searchTimer != nil && tv.searchTimer.Stop() {
		tv.filterBusy--
	}
	tv.filterBusy++
	tv.searchTimer = time.AfterFunc(searchDebounce, func() {
		tv.sh.dispatch("ui.setFilter", filterParams{Filter: text}, func(_ any, err error) {
			tv.filterBusy--
			tv.sh.showError(err)
			tv.syncSearch()
		})
	})
}

// syncSearch shows the engine's filter in the search entry, unless the user is still typing.
func (tv *tableView) syncSearch() {
	if tv.filterBusy > 0 || tv.search.Text == tv.sh.ui.Filter {
		return
	}
	tv.syncing2 = true
	tv.search.SetText(tv.sh.ui.Filter)
	tv.syncing2 = false
}

// updateNamespaceButton labels the namespace filter: "All namespaces", one name, or a count; on
// the Definitions view the same button is the API group filter ("All groups").
func (tv *tableView) updateNamespaceButton() {
	all, _, sel := tv.filterChoices()
	text := all
	switch {
	case len(sel) == 1:
		text = sel[0]
	case len(sel) > 1:
		noun := "namespaces"
		if tv.groupMode() {
			noun = "groups"
		}
		text = formatCount(int64(len(sel))) + " " + noun
	}
	if tv.nsButton.Text != text {
		tv.nsButton.SetText(text)
	}
	// Lens shows no namespace filter on a cluster-scoped view (captures of Nodes, Ingress Classes,
	// Cluster Roles and Cluster Role Bindings, 2026-10-11), and a group filter on Definitions.
	if tv.groupMode() || tv.viewNamespaced() {
		tv.nsBox.Show()
	} else {
		tv.nsBox.Hide()
	}
}

// groupMode reports whether the filter button is the Definitions view's API group filter.
func (tv *tableView) groupMode() bool { return engine.FiltersByGroup(tv.sh.ui.View) }

// filterChoices returns what the filter button offers: the "all" entry's label, the names to
// choose from, and the engine's selection. Namespaces come from the cluster; API groups from its
// custom resource kinds (kinds.list), plus any group the selection holds.
func (tv *tableView) filterChoices() (all string, names, selected []string) {
	if !tv.groupMode() {
		return "All namespaces", tv.sh.namespaces, tv.sh.ui.Namespaces
	}
	set := map[string]bool{}
	for _, k := range tv.sh.kinds {
		if isCustomKind(k.Kind) {
			set[apiGroupOf(k)] = true
		}
	}
	for _, g := range tv.sh.ui.Groups {
		set[g] = true
	}
	for g := range set {
		names = append(names, g)
	}
	slices.Sort(names)
	return "All groups", names, tv.sh.ui.Groups
}

// viewNamespaced reports whether the open view's kind is namespaced: from its snapshot, before
// the first one from kinds.list, and true when neither knows (the filter then stays where it is
// until the view's snapshot says otherwise).
func (tv *tableView) viewNamespaced() bool {
	if tv.snap != nil && tv.snap.View == tv.sh.ui.View {
		return tv.snap.Namespaced
	}
	for _, k := range tv.sh.kinds {
		if k.Kind == tv.sh.ui.View {
			return k.Namespaced
		}
	}
	return true
}

// namespacesLoaded refreshes an open namespace popup with the newly loaded names.
func (tv *tableView) namespacesLoaded() {
	if tv.nsPopup != nil && tv.nsPopup.Visible() {
		tv.nsPopup.Hide()
		tv.openNamespaces()
	}
}

// openNamespaces shows the namespace filter: "All namespaces" and one check per namespace (on
// Definitions: "All groups" and one per API group). Checks toggle without closing the popup, as
// Lens's filter does.
func (tv *tableView) openNamespaces() {
	all, choices, _ := tv.filterChoices()
	names := append([]string{""}, choices...)
	tv.nsChecks = tv.nsChecks[:0]
	list := container.NewVBox()
	for _, ns := range names {
		label := ns
		if ns == "" {
			label = all
		}
		c := widget.NewCheck(label, func(on bool) { tv.toggleNamespace(ns, on) })
		tv.nsChecks = append(tv.nsChecks, c)
		list.Add(c)
	}
	tv.syncNamespaceChecks()
	th := tv.table.Theme()
	scroll := container.NewVScroll(list)
	h := list.MinSize().Height
	if maxH := th.Size(sizePopupMaxList); h > maxH {
		h = maxH
	}
	scroll.SetMinSize(fyne.NewSize(th.Size(sizeFilterWidth)+th.Size(sizeCellPadding)*2, h))
	c := fyne.CurrentApp().Driver().CanvasForObject(tv.nsButton)
	if c == nil {
		return
	}
	tv.nsPopup = widget.NewPopUp(scroll, c)
	tv.nsPopup.ShowAtRelativePosition(fyne.NewPos(0, tv.nsButton.Size().Height), tv.nsButton)
	if len(tv.sh.namespaces) == 0 && !tv.groupMode() {
		tv.sh.loadNamespaces()
	}
}

// syncNamespaceChecks makes the popup's checks show the engine's namespaces (or groups).
func (tv *tableView) syncNamespaceChecks() {
	if len(tv.nsChecks) == 0 {
		return
	}
	_, _, sel := tv.filterChoices()
	tv.nsSync = true
	for i, c := range tv.nsChecks {
		want := len(sel) == 0
		if i > 0 {
			want = slices.Contains(sel, c.Text)
		}
		if c.Checked != want {
			c.SetChecked(want)
		}
	}
	tv.nsSync = false
}

// toggleNamespace changes the namespace filter: "All" clears it, a namespace is added or removed.
// On Definitions it changes the API group filter the same way.
func (tv *tableView) toggleNamespace(ns string, on bool) {
	if tv.nsSync {
		return
	}
	_, _, cur := tv.filterChoices()
	next := []string{}
	switch {
	case ns == "":
		// "All namespaces": checking it clears the selection; unchecking it changes nothing.
		if !on {
			tv.syncNamespaceChecks()
			return
		}
	case on:
		next = append(slices.Clone(cur), ns)
	default:
		for _, n := range cur {
			if n != ns {
				next = append(next, n)
			}
		}
	}
	if tv.groupMode() {
		tv.sh.dispatch("ui.setGroups", groupsParams{Groups: next}, nil)
		return
	}
	tv.sh.dispatch("ui.setNamespace", namespacesParams{Namespaces: next}, nil)
}

// resourceTable is widget.Table with the hooks Fyne does not offer as callbacks: the pointer's
// position (for tooltips), the pointer leaving the table (clears the row highlight and the
// tooltip), scrolling (closes the tooltip), and Escape, which closes the drawer while the table
// has the focus.
type resourceTable struct {
	widget.Table
	tv     *tableView
	keying bool          // set while a key moves Fyne's highlight (tablekeys.go)
	shift  bool          // Shift is held, for ⇧Tab
	offset fyne.Position // the scroll offset, as the scroller last reported it
}

func newResourceTable(tv *tableView) *resourceTable {
	t := &resourceTable{tv: tv}
	t.Length = tv.length
	t.CreateCell = tv.createCell
	t.UpdateCell = tv.updateCell
	t.ShowHeaderRow = true
	t.CreateHeader = tv.createHeader
	t.UpdateHeader = tv.updateHeader
	// Fyne's dividers draw a grid and its selection marks one cell; Cogpit's cells draw row
	// lines and a full-row highlight themselves. HideSeparators hides both of Fyne's.
	t.HideSeparators = true
	t.OnSelected = tv.rowTapped
	t.OnHighlighted = tv.highlighted
	t.ExtendBaseWidget(t)
	return t
}

// CreateRenderer adds a scroll hook to the table's: a scroll moves the cells under the pointer,
// so the tooltip closes; the offset is kept for finding the column separator under a double click.
func (t *resourceTable) CreateRenderer() fyne.WidgetRenderer {
	r := t.Table.CreateRenderer()
	for _, o := range r.Objects() {
		if s, ok := o.(*container.Scroll); ok && s.OnScrolled != nil {
			inner := s.OnScrolled
			s.OnScrolled = func(p fyne.Position) {
				inner(p)
				t.offset = p
				t.tv.sh.tip.hide()
			}
		}
	}
	return r
}

// MouseIn records the pointer for tooltips.
func (t *resourceTable) MouseIn(ev *desktop.MouseEvent) {
	t.tv.pointer, t.tv.pointerIn = ev.AbsolutePosition, true
	t.Table.MouseIn(ev)
}

// MouseMoved records the pointer; the table's hover callback then moves the highlight.
func (t *resourceTable) MouseMoved(ev *desktop.MouseEvent) {
	t.tv.pointer, t.tv.pointerIn = ev.AbsolutePosition, true
	t.Table.MouseMoved(ev)
	t.tv.requestTip()
}

// MouseOut clears the row highlight and the tooltip when the pointer leaves the table.
func (t *resourceTable) MouseOut() {
	t.Table.MouseOut()
	t.tv.pointerIn = false
	t.tv.setHover(-1)
	t.tv.sh.tip.hide()
}

// cellKind says what a body cell draws.
type cellKind uint8

const (
	cellText    cellKind = iota // text, possibly a link or in coloured parts
	cellWarning                 // Lens's warning icon; the text is the icon's tooltip
	cellMenu                    // the ⋮ that opens the row's actions menu
)

// cellContent is what one body cell shows. Its fields are values or pointers into the immutable
// snapshot, so building and comparing one allocates nothing.
type cellContent struct {
	text   string
	status model.Status
	link   *model.ObjectRef
	parts  *[]model.CellPart
	kind   cellKind
}

// tableCell is one body cell: text in the cell's status colour (a link in the link colour, a cell
// in coloured parts run by run), or an icon (the warning triangle, the ⋮), and a background and
// bottom line that reach into the table's cell gaps so that a row reads as one band.
//
// The cell handles its own taps, so that a click on a link follows it instead of selecting the
// row; any other click selects the row through the table, as before. A right-click opens the
// row's actions menu.
type tableCell struct {
	widget.BaseWidget
	tv    *tableView
	bg    *canvas.Rectangle
	text  *canvas.Text   // the text, or the first part
	more  []*canvas.Text // the further parts of a cell in coloured parts; grown on demand
	runX  []float32      // each part's x offset, set by fit
	icon  *canvas.Image  // made when the cell first shows an icon
	line  *canvas.Rectangle
	under *canvas.Rectangle // a hovered link's underline; made when the cell first shows a link
	ring  *canvas.Rectangle // the keyboard's focus ring; made when the keyboard first reaches the cell
	objs  int               // bumped when an object is added, so the renderer lists it

	content   cellContent
	full      string // the cell text; text.Text may be a shortened form of it
	fitWidth  float32
	truncated bool // the shown text is shorter than full: the tooltip shows full
	iconRes   fyne.Resource
	selected  bool
	hover     bool // the pointer or the keyboard is on the row
	linkHover bool // the pointer is on this cell, which is a link
	focused   bool // the keyboard is on this cell (tablekeys.go)
	tracked   bool // registered in tableView.cells
	row, col  int
}

var (
	_ fyne.Tappable          = (*tableCell)(nil)
	_ fyne.SecondaryTappable = (*tableCell)(nil)
	_ desktop.Cursorable     = (*tableCell)(nil)
)

func newTableCell(tv *tableView) *tableCell {
	c := &tableCell{
		tv:   tv,
		bg:   canvas.NewRectangle(color.Transparent),
		text: canvas.NewText("", color.Transparent),
		line: canvas.NewRectangle(color.Transparent),
		row:  -1, col: -1,
	}
	c.ExtendBaseWidget(c)
	return c
}

// set gives the cell new content; unchanged content costs a few comparisons. It reports whether
// the content changed.
func (c *tableCell) set(row, col int, content cellContent, selected, hover, linkHover bool) bool {
	c.row, c.col = row, col
	changed := false
	if content != c.content {
		c.content, c.full, c.fitWidth = content, content.text, -1
		changed = true
	}
	linkHover = linkHover && content.link != nil
	if changed || selected != c.selected || hover != c.hover || linkHover != c.linkHover {
		c.selected, c.hover, c.linkHover = selected, hover, linkHover
		c.apply()
	}
	return changed
}

func (c *tableCell) setSelected(on bool) {
	if on != c.selected {
		c.selected = on
		c.apply()
	}
}

// setFocused rings the cell while the keyboard is on it.
func (c *tableCell) setFocused(on bool) {
	if on == c.focused {
		return
	}
	c.focused = on
	if c.ring == nil {
		c.ring = canvas.NewRectangle(color.Transparent)
		c.objs++
	}
	c.Refresh() // applies the ring's colour and lays it out; two cells per key press
}

// setHover marks the row as hovered and, for a link, the cell itself.
func (c *tableCell) setHover(row, cell bool) {
	cell = cell && c.content.link != nil
	if row != c.hover || cell != c.linkHover {
		c.hover, c.linkHover = row, cell
		c.apply()
	}
}

// Tapped implements fyne.Tappable: follow a link, open the menu of the ⋮, else select the row.
func (c *tableCell) Tapped(e *fyne.PointEvent) { c.tv.cellTapped(c, e) }

// TappedSecondary implements fyne.SecondaryTappable: a right-click opens the row's actions menu
// (Lens since 2025.10.230725).
func (c *tableCell) TappedSecondary(e *fyne.PointEvent) { c.tv.openRowMenu(c.row, e.AbsolutePosition) }

// Cursor implements desktop.Cursorable: a hand over a link the pointer is on and over the ⋮.
func (c *tableCell) Cursor() desktop.Cursor {
	if c.linkHover || c.content.kind == cellMenu {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// onLink reports whether a point in the cell lies on its link text.
func (c *tableCell) onLink(p fyne.Position) bool {
	if c.content.link == nil || c.text.Text == "" {
		return false
	}
	x := c.text.Position().X
	return p.X >= x && p.X <= x+c.text.Size().Width
}

// tipText is the cell's tooltip: a warning's reason, or the full text of a cut-off cell.
func (c *tableCell) tipText() string {
	switch {
	case c.content.kind == cellWarning:
		return c.full
	case c.content.kind == cellText && c.truncated:
		return c.full
	}
	return ""
}

// apply recolours and refits the cell's canvas objects.
func (c *tableCell) apply() {
	th := c.Theme()
	switch {
	case c.selected:
		c.bg.FillColor = themeColor(th, colorRowSelected)
	case c.hover:
		c.bg.FillColor = themeColor(th, colorRowHover)
	default:
		c.bg.FillColor = color.Transparent
	}
	size := th.Size(theme.SizeNameText)
	fg := statusColorName(c.content.status)
	if c.content.link != nil {
		fg = colorLink
	}
	if c.selected && (fg == colorLink || fg == theme.ColorNameForeground) {
		fg = colorRowSelectedText // the plain and link colours are too faint on the selection
	}
	c.text.Color = themeColor(th, fg)
	c.text.TextSize = size
	if c.content.parts != nil && c.content.kind == cellText {
		parts := *c.content.parts
		for len(c.more) < len(parts)-1 {
			c.more = append(c.more, canvas.NewText("", color.Transparent))
		}
		if len(parts) > 0 {
			c.text.Color = themeColor(th, statusColorName(parts[0].Status))
		}
		for i, t := range c.more {
			t.TextSize = size
			if i+1 < len(parts) {
				t.Color = themeColor(th, statusColorName(parts[i+1].Status))
			}
		}
	}
	if c.under == nil && c.content.link != nil {
		c.under = canvas.NewRectangle(color.Transparent)
		c.objs++
	}
	if c.under != nil {
		c.under.FillColor = themeColor(th, colorLink)
		if c.linkHover {
			c.under.Show()
		} else {
			c.under.Hide()
		}
		c.under.Refresh()
	}
	c.line.FillColor = themeColor(th, theme.ColorNameSeparator)
	if c.ring != nil {
		c.ring.FillColor = color.Transparent
		c.ring.StrokeColor = themeColor(th, colorAccent)
		c.ring.StrokeWidth = th.Size(theme.SizeNameInputBorder)
		c.ring.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
		if c.focused {
			c.ring.Show()
		} else {
			c.ring.Hide()
		}
		c.ring.Refresh()
	}
	c.setIcon(c.tv.cellIcon(c.content))
	c.fit(th)
	c.bg.Refresh()
	c.text.Refresh()
	for _, t := range c.more {
		t.Refresh()
	}
}

// setIcon shows res, or hides the icon for nil. The image is refreshed (its SVG rasterised) only
// when the resource changes; the resources are shared per icon and colour (tableView.cellIcon).
func (c *tableCell) setIcon(res fyne.Resource) {
	if res == nil {
		if c.icon != nil && c.icon.Visible() {
			c.icon.Hide()
		}
		return
	}
	if c.icon == nil {
		c.icon = canvas.NewImageFromResource(nil)
		c.icon.FillMode = canvas.ImageFillContain
		c.objs++
	}
	if res != c.iconRes {
		c.iconRes = res
		c.icon.Resource = res
		c.icon.Refresh()
	}
	if !c.icon.Visible() {
		c.icon.Show()
	}
}

// fit shows the text, shortened with an ellipsis when it is wider than the cell; a cell in parts
// shows its runs side by side and shortens the run that reaches the edge. Measuring is cached by
// Fyne per string; the shortening runs only when the text or the width changed. It reports
// whether the shown text changed.
func (c *tableCell) fit(th fyne.Theme) bool {
	avail := c.Size().Width - 2*th.Size(sizeCellPadding)
	if avail == c.fitWidth {
		return false
	}
	c.fitWidth = avail
	if c.content.kind != cellText {
		c.truncated = false
		changed := c.text.Text != ""
		c.text.Text = ""
		c.hideMore(0)
		return changed
	}
	if c.content.parts == nil || len(*c.content.parts) == 0 {
		shown := ellipsize(c.full, avail, c.text.TextSize, c.text.TextStyle)
		c.truncated = shown != c.full
		c.hideMore(0)
		if shown == c.text.Text {
			return false
		}
		c.text.Text = shown
		return true
	}
	parts := *c.content.parts
	space := fyne.MeasureText(" ", c.text.TextSize, c.text.TextStyle).Width
	c.runX = c.runX[:0]
	x := float32(0)
	c.truncated = false
	for i, p := range parts {
		t := c.text
		if i > 0 {
			t = c.more[i-1]
		}
		c.runX = append(c.runX, x)
		w := fyne.MeasureText(p.Text, t.TextSize, t.TextStyle).Width
		switch {
		case avail-x <= 0:
			t.Text, c.truncated = "", true
		case x+w > avail:
			t.Text, c.truncated = ellipsize(p.Text, avail-x, t.TextSize, t.TextStyle), true
		default:
			t.Text = p.Text
		}
		x += w + space
	}
	c.hideMore(len(parts) - 1)
	return true
}

// hideMore empties the part texts from index n on.
func (c *tableCell) hideMore(n int) {
	for i := n; i < len(c.more); i++ {
		c.more[i].Text = ""
	}
}

// ellipsize returns s, or its longest prefix plus "…" that fits in width.
func ellipsize(s string, width, size float32, style fyne.TextStyle) string {
	if s == "" || width <= 0 || fyne.MeasureText(s, size, style).Width <= width {
		return s
	}
	runes := []rune(s)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if fyne.MeasureText(string(runes[:mid])+"…", size, style).Width <= width {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return string(runes[:lo]) + "…"
}

// CreateRenderer implements fyne.Widget.
func (c *tableCell) CreateRenderer() fyne.WidgetRenderer {
	c.apply()
	return &tableCellRenderer{c: c}
}

type tableCellRenderer struct {
	c     *tableCell
	objs  []fyne.CanvasObject
	nmore int
	gen   int
}

func (r *tableCellRenderer) Layout(size fyne.Size) {
	c := r.c
	th := c.Theme()
	gap := th.Size(theme.SizeNamePadding)
	c.bg.Move(fyne.NewPos(-gap/2, -gap/2))
	c.bg.Resize(fyne.NewSize(size.Width+gap, size.Height+gap))
	c.line.Move(fyne.NewPos(-gap/2, size.Height+gap/2-1))
	c.line.Resize(fyne.NewSize(size.Width+gap, 1))
	if c.ring != nil {
		c.ring.Move(fyne.NewPos(-gap/2, -gap/2))
		c.ring.Resize(fyne.NewSize(size.Width+gap, size.Height+gap))
	}
	if c.fit(th) {
		c.text.Refresh() // a column resize shortened or lengthened the text
		for _, t := range c.more {
			t.Refresh()
		}
	}
	pad := th.Size(sizeCellPadding)
	ts := c.text.MinSize()
	y := (size.Height - ts.Height) / 2
	c.text.Move(fyne.NewPos(pad, y))
	c.text.Resize(ts)
	for i, t := range c.more {
		if i+1 < len(c.runX) && t.Text != "" {
			t.Move(fyne.NewPos(pad+c.runX[i+1], y))
			t.Resize(t.MinSize())
		}
	}
	if c.under != nil {
		c.under.Move(fyne.NewPos(pad, y+ts.Height-2))
		c.under.Resize(fyne.NewSize(ts.Width, 1))
	}
	if c.icon != nil {
		is := th.Size(sizeCellIcon)
		c.icon.Move(fyne.NewPos((size.Width-is)/2, (size.Height-is)/2))
		c.icon.Resize(fyne.NewSize(is, is))
	}
}

func (r *tableCellRenderer) MinSize() fyne.Size {
	th := r.c.Theme()
	h := fyne.MeasureText("Mg", th.Size(theme.SizeNameText), fyne.TextStyle{}).Height
	return fyne.NewSize(2*th.Size(sizeCellPadding)+h, h+2*th.Size(sizeRowPadding))
}

func (r *tableCellRenderer) Refresh() {
	r.c.fitWidth = -1
	r.c.apply()
	r.Layout(r.c.Size())
	r.c.line.Refresh()
}

// Objects returns the cell's canvas objects; the list is rebuilt only when the cell made a new
// one (a part, its first icon or underline).
func (r *tableCellRenderer) Objects() []fyne.CanvasObject {
	c := r.c
	if r.objs == nil || r.nmore != len(c.more) || r.gen != c.objs {
		r.nmore, r.gen = len(c.more), c.objs
		r.objs = append(r.objs[:0], c.bg, c.line, c.text)
		if c.under != nil {
			r.objs = append(r.objs, c.under)
		}
		if c.icon != nil {
			r.objs = append(r.objs, c.icon)
		}
		for _, t := range c.more {
			r.objs = append(r.objs, t)
		}
		if c.ring != nil {
			r.objs = append(r.objs, c.ring)
		}
	}
	return r.objs
}

func (r *tableCellRenderer) Destroy() {}

// cellIcon returns the icon a cell shows: the warning triangle in the cell's status colour, the ⋮
// in the muted colour, or nil. The themed resources are made once per icon and colour, so that
// updating cells allocates nothing and a cell's image is re-rasterised only when its icon changes.
func (tv *tableView) cellIcon(content cellContent) fyne.Resource {
	var key cellIconKey
	switch content.kind {
	case cellWarning:
		if content.status == model.StatusNone {
			return nil
		}
		key = cellIconKey{theme.IconNameWarning, statusColorName(content.status)}
	case cellMenu:
		key = cellIconKey{theme.IconNameMoreVertical, colorMuted}
	default:
		return nil
	}
	if res, ok := tv.icons[key]; ok {
		return res
	}
	res := themedIcon(theme.Icon(key.icon), key.color)
	tv.icons[key] = res
	return res
}

// cellIconKey names a table icon in a colour.
type cellIconKey struct {
	icon  fyne.ThemeIconName
	color fyne.ThemeColorName
}

// headerCell is a column header: the title, an arrow on the sorted column, and a short divider at
// its right edge that marks where the column can be dragged wider. Tapping it sorts. The header of
// the ⋮ column shows a ⋮ instead, which opens the list of columns to show and hide.
type headerCell struct {
	widget.BaseWidget
	onTap func(col int)
	col   int
	title string
	sort  int  // 1 ascending, -1 descending, 0 not sorted
	menu  bool // the ⋮ at the end of the header row

	text  *canvas.Text
	arrow *iconImage
	sep   *canvas.Rectangle
	line  *canvas.Rectangle
}

func newHeaderCell(onTap func(int)) *headerCell {
	h := &headerCell{
		onTap: onTap, col: -1,
		text:  canvas.NewText("", color.Transparent),
		arrow: newIconImage(),
		sep:   canvas.NewRectangle(color.Transparent),
		line:  canvas.NewRectangle(color.Transparent),
	}
	h.ExtendBaseWidget(h)
	return h
}

func (h *headerCell) set(col int, title string, sort int) {
	h.col = col
	if title == h.title && sort == h.sort && !h.menu {
		return
	}
	h.title, h.sort, h.menu = title, sort, false
	h.Refresh()
}

// setMenu makes the cell the ⋮ of the header row (on), or an empty header.
func (h *headerCell) setMenu(col int, on bool) {
	h.col = col
	if on == h.menu && h.title == "" && h.sort == 0 {
		return
	}
	h.title, h.sort, h.menu = "", 0, on
	h.Refresh()
}

// Cursor implements desktop.Cursorable: a hand over the ⋮.
func (h *headerCell) Cursor() desktop.Cursor {
	if h.menu {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// Tapped implements fyne.Tappable.
func (h *headerCell) Tapped(*fyne.PointEvent) {
	if h.onTap != nil {
		h.onTap(h.col)
	}
}

// CreateRenderer implements fyne.Widget.
func (h *headerCell) CreateRenderer() fyne.WidgetRenderer {
	r := &headerCellRenderer{h: h}
	r.Refresh()
	return r
}

type headerCellRenderer struct{ h *headerCell }

func (r *headerCellRenderer) Layout(size fyne.Size) {
	h := r.h
	th := h.Theme()
	gap := th.Size(theme.SizeNamePadding)
	pad := th.Size(sizeCellPadding)
	ts := h.text.MinSize()
	h.text.Move(fyne.NewPos(pad, (size.Height-ts.Height)/2))
	h.text.Resize(ts)
	is := th.Size(sizeSortIcon)
	if h.menu {
		is = th.Size(sizeCellIcon)
		h.arrow.img.Move(fyne.NewPos((size.Width-is)/2, (size.Height-is)/2))
	} else {
		h.arrow.img.Move(fyne.NewPos(pad+ts.Width+gap/2, (size.Height-is)/2))
	}
	h.arrow.img.Resize(fyne.NewSize(is, is))
	h.sep.Move(fyne.NewPos(size.Width+gap/2-0.5, size.Height/4))
	h.sep.Resize(fyne.NewSize(1, size.Height/2))
	h.line.Move(fyne.NewPos(-gap/2, size.Height-1))
	h.line.Resize(fyne.NewSize(size.Width+gap, 1))
}

func (r *headerCellRenderer) MinSize() fyne.Size {
	th := r.h.Theme()
	h := fyne.MeasureText("Mg", th.Size(theme.SizeNameText), fyne.TextStyle{Bold: true}).Height
	return fyne.NewSize(2*th.Size(sizeCellPadding)+h, h+2*th.Size(sizeRowPadding))
}

func (r *headerCellRenderer) Refresh() {
	h := r.h
	th := h.Theme()
	h.text.Text = h.title
	h.text.TextSize = th.Size(theme.SizeNameText)
	h.text.TextStyle = fyne.TextStyle{Bold: true}
	h.text.Color = themeColor(th, colorMuted)
	h.sep.FillColor = themeColor(th, theme.ColorNameSeparator)
	h.line.FillColor = themeColor(th, theme.ColorNameSeparator)
	var arrow fyne.Resource
	switch h.sort {
	case 1:
		arrow = theme.MenuDropUpIcon()
	case -1:
		arrow = theme.MenuDropDownIcon()
	}
	if h.menu {
		h.arrow.set(th, theme.MoreVerticalIcon(), colorMuted)
	} else {
		h.arrow.set(th, arrow, colorAccent)
	}
	r.Layout(h.Size())
	h.text.Refresh()
	h.sep.Refresh()
	h.line.Refresh()
}

func (r *headerCellRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.h.line, r.h.sep, r.h.text, r.h.arrow.img}
}

func (r *headerCellRenderer) Destroy() {}
