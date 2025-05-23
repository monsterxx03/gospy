package termui

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/monsterxx03/gospy/pkg/proc"
)

type TopUI struct {
	app          *tview.Application
	table        *tview.Table
	titleView    *tview.TextView
	memStatsView *tview.TextView
	searchView   *tview.InputField
	pid          int
	interval     int
	memReader    proc.ProcessMemReader
	suspended    bool
	refreshChan  chan struct{}
	lastMemStat  *proc.MemStat
	searchFilter string
	flex         *tview.Flex
	lastUpdate   time.Time
	lastDuration time.Duration
	showDead     bool
}

func (t *TopUI) updateHelpText(help *tview.TextView) {
	baseHelp := "[yellow]Press [white]q[green] to quit, [white]r[green] to refresh, [white]s[green] to suspend/resume, [white]/[green] to search"
	if t.searchFilter != "" {
		baseHelp += fmt.Sprintf(" [white]| [green]Current filter: [white]%q", t.searchFilter)
	} else {
		baseHelp += " [white]| [green]No active filter"
	}
	help.SetText(baseHelp)
}

func NewTopUI(pid, interval int, showDead bool, memReader proc.ProcessMemReader) *TopUI {
	app := tview.NewApplication()
	table := tview.NewTable()
	table.SetBorders(false).
		SetFixed(1, 0).
		SetBorder(false)

	ui := &TopUI{
		app:       app,
		table:     table,
		pid:       pid,
		interval:  interval,
		memReader: memReader,
		showDead:  showDead,
	}

	// Create title view
	ui.titleView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)

	return ui
}

func (t *TopUI) Run() error {
	// Set table headers
	t.table.SetCell(0, 0, tview.NewTableCell("Count").
		SetAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow).
		SetBackgroundColor(tcell.ColorDarkSlateGray))
	t.table.SetCell(0, 1, tview.NewTableCell("Status").
		SetAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow).
		SetBackgroundColor(tcell.ColorDarkSlateGray))
	t.table.SetCell(0, 2, tview.NewTableCell("Function").
		SetAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorYellow).
		SetBackgroundColor(tcell.ColorDarkSlateGray))

	// Add help text
	help := tview.NewTextView().
		SetDynamicColors(true)
	t.updateHelpText(help)

	// Create memory stats view
	t.memStatsView = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)

	// Create search input
	t.searchView = tview.NewInputField().
		SetLabel("Search: ").
		SetFieldBackgroundColor(tcell.ColorDefault).
		SetChangedFunc(func(text string) {
			t.searchFilter = text
		}).
		SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEsc || key == tcell.KeyEnter {
				t.flex.RemoveItem(t.searchView)
				t.app.SetFocus(t.table)
				go t.app.QueueUpdateDraw(
					func() {
						t.updateHelpText(help)
						t.refreshChan <- struct{}{}
					},
				)
			}
		})

	// Create layout
	t.flex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(t.titleView, 1, 1, false).    // Title row
		AddItem(t.memStatsView, 4, 1, false). // Memory stats
		AddItem(t.table, 0, 1, true).         // Table content
		AddItem(help, 1, 1, false)            // Help text

	// Set up refresh control
	t.refreshChan = make(chan struct{})
	ticker := time.NewTicker(time.Duration(t.interval) * time.Second)
	defer ticker.Stop()

	// Initial update - use direct call since app isn't running yet
	t.update()

	// Handle keyboard input
	t.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// If search view has focus, only handle escape
		if t.app.GetFocus() == t.searchView {
			switch event.Key() {
			case tcell.KeyEsc:
				t.app.SetFocus(t.table)
				return nil
			default:
				// Let the input field handle all other keys
				return event
			}
		}

		// Normal mode key handling
		switch event.Rune() {
		case 'q':
			t.app.Stop()
			return nil
		case 'r':
			go t.update()
			return nil
		case 's':
			t.suspended = !t.suspended
			if t.suspended {
				t.titleView.SetText(fmt.Sprintf("%s [red](PAUSED)", t.titleView.GetText(true)))
			} else {
				t.refreshChan <- struct{}{} // Trigger immediate refresh
			}
			return nil
		case '/':
			t.searchView.SetText(t.searchFilter) // Keep current filter when reopening
			t.flex.AddItem(t.searchView, 1, 1, false)
			t.app.SetFocus(t.searchView)
			return nil
		}
		return event
	})

	// Start refresh loop
	go func() {
		for {
			select {
			case <-ticker.C:
				if !t.suspended && t.app != nil {
					t.app.QueueUpdateDraw(t.update)
				}
			case <-t.refreshChan:
				if t.app != nil {
					t.app.QueueUpdateDraw(t.update)
				}
			}
		}
	}()

	// Run application
	return t.app.SetRoot(t.flex, true).Run()
}

func (t *TopUI) fetchData() (*proc.Runtime, *proc.MemStat, []proc.G, error) {
	start := time.Now()
	defer func() {
		t.lastDuration = time.Since(start)
	}()

	// Get runtime info
	rt, err := t.memReader.RuntimeInfo()
	if err != nil {
		rt = &proc.Runtime{GoVersion: fmt.Sprintf("error: %v", err)}
	}

	memStat, err := t.memReader.MemStat()
	if err != nil {
		memStat = t.lastMemStat // Use last known stats if error
	} else {
		t.lastMemStat = memStat
	}

	goroutines, err := t.memReader.Goroutines(t.showDead)
	if err != nil {
		return nil, nil, nil, err
	}

	return rt, memStat, goroutines, nil
}

func (t *TopUI) renderGoroutines(goroutines []proc.G) {
	// Group goroutines
	groups := make(map[string]*struct {
		count  int
		status map[string]int
	})

	for _, g := range goroutines {
		funcName := g.StartFuncName
		if funcName == "" {
			funcName = "unknown"
		}

		// Apply search filter
		if t.searchFilter != "" && !strings.Contains(strings.ToLower(funcName), strings.ToLower(t.searchFilter)) {
			continue
		}

		if groups[funcName] == nil {
			groups[funcName] = &struct {
				count  int
				status map[string]int
			}{
				status: make(map[string]int),
			}
		}
		groups[funcName].count++
		groups[funcName].status[g.Status]++
	}

	// Clear and setup table headers
	t.table.Clear()
	t.table.SetCell(0, 0, tview.NewTableCell("Count").
		SetAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow).
		SetBackgroundColor(tcell.ColorDarkSlateGray))
	t.table.SetCell(0, 1, tview.NewTableCell("Status").
		SetAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorYellow).
		SetBackgroundColor(tcell.ColorDarkSlateGray))
	t.table.SetCell(0, 2, tview.NewTableCell("Function").
		SetAlign(tview.AlignLeft).
		SetTextColor(tcell.ColorYellow).
		SetBackgroundColor(tcell.ColorDarkSlateGray))

	// Group goroutines by function name
	type goroutineGroup struct {
		funcName string
		count    int
		status   map[string]int
	}
	var sortedGroups []goroutineGroup
	for funcName, info := range groups {
		sortedGroups = append(sortedGroups, goroutineGroup{
			funcName: funcName,
			count:    info.count,
			status:   info.status,
		})
	}

	// Sort by count in descending order
	sort.Slice(sortedGroups, func(i, j int) bool {
		return sortedGroups[i].count > sortedGroups[j].count
	})

	row := 1
	for _, group := range sortedGroups {
		// Build status string sorted alphabetically
		var statusParts []string
		// Get sorted status names
		statusNames := make([]string, 0, len(group.status))
		for status := range group.status {
			statusNames = append(statusNames, status)
		}
		sort.Strings(statusNames)

		// Build parts in sorted order
		for _, status := range statusNames {
			statusParts = append(statusParts, fmt.Sprintf("%s:%d", status, group.status[status]))
		}
		statusStr := strings.Join(statusParts, " ")

		t.table.SetCell(row, 0, tview.NewTableCell(fmt.Sprintf("%d", group.count)))
		t.table.SetCell(row, 1, tview.NewTableCell(statusStr))
		t.table.SetCell(row, 2, tview.NewTableCell(group.funcName))
		row++
	}
}

func (t *TopUI) renderTitle(rt *proc.Runtime, goroutineCount int) {
	uptime := fmt.Sprintf(" [white]| [cyan]Uptime: %s", proc.FormatDuration(rt.Uptime()))
	title := fmt.Sprintf("[yellow]PID: %d [white]| [green]Go: %s [white]| [blue]Goroutines: %d [white]| [purple]Refresh: %ds [white]| [orange]Update: %v%s",
		t.pid, rt.GoVersion, goroutineCount, t.interval, t.lastDuration.Round(time.Microsecond), uptime)
	t.titleView.SetText(title)
}

func (t *TopUI) renderMemStats(memStat *proc.MemStat, goroutines []proc.G, ps []proc.P) {
	lastGC := "never"
	if memStat.LastGC > 0 {
		lastGC = proc.FormatDuration(time.Since(time.Unix(0, int64(memStat.LastGC)))) + " ago"
	}

	// Calculate processor status distribution
	pStatusCounts := make(map[string]int)
	for _, p := range ps {
		pStatusCounts[p.Status]++
	}
	// Calculate goroutine status distribution
	statusCounts := make(map[string]int)
	for _, g := range goroutines {
		statusCounts[g.Status]++
	}

	// Build goroutine status string
	var gStatusParts []string
	var gStatusNames []string
	for status := range statusCounts {
		gStatusNames = append(gStatusNames, status)
	}
	sort.Strings(gStatusNames)
	for _, status := range gStatusNames {
		gStatusParts = append(gStatusParts, fmt.Sprintf("%s:%d", status, statusCounts[status]))
	}
	gStatusStr := strings.Join(gStatusParts, " ")

	// Build processor status string
	var pStatusParts []string
	var pStatusNames []string
	for status := range pStatusCounts {
		pStatusNames = append(pStatusNames, status)
	}
	sort.Strings(pStatusNames)
	for _, status := range pStatusNames {
		pStatusParts = append(pStatusParts, fmt.Sprintf("%s:%d", status, pStatusCounts[status]))
	}
	pStatusStr := strings.Join(pStatusParts, " ")

	gcStats := fmt.Sprintf(
		"[yellow]GC Stats: [white]Last: %s | Total Pause: %s | Count: %d\n"+
			"[yellow]Recent Pauses: [white]%s, %s, %s\n"+
			"[yellow]Goroutine Status: [white]%s\n"+
			"[yellow]Processor Status: [white]%s",
		lastGC,
		proc.FormatDuration(time.Duration(memStat.PauseTotalNs)),
		memStat.NumGC,
		proc.FormatDuration(time.Duration(memStat.PauseNs[0])),
		proc.FormatDuration(time.Duration(memStat.PauseNs[1])),
		proc.FormatDuration(time.Duration(memStat.PauseNs[2])),
		gStatusStr,
		pStatusStr,
	)
	t.memStatsView.SetText(gcStats)
}

func (t *TopUI) update() {
	// Fetch data first
	rt, memStat, goroutines, err := t.fetchData()
	if err != nil {
		t.app.Stop()
		fmt.Fprintf(os.Stderr, "failed to get goroutines: %v\n", err)
		return
	}

	// Get processor info
	ps, err := t.memReader.Ps()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to get processors: %v\n", err)
		ps = nil
	}

	// Update title and memory stats
	t.renderTitle(rt, len(goroutines))

	if memStat != nil && t.memStatsView != nil {
		t.renderMemStats(memStat, goroutines, ps)
	}

	// Render goroutines table
	t.renderGoroutines(goroutines)
}
