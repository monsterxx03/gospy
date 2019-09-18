package term

import (
	"sort"
	"strconv"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"gospy/pkg/proc"
)

const (
	SUMMARY_HEIGHT = 4
	TOP_HEIGHT     = 50
)

var TOP_HEADER = []string{"GCount", "Func"}

type fnStat struct {
	fn    string
	count int
}

type Term struct {
	summary         *widgets.Paragraph
	top             *widgets.Table
	refreshInterval time.Duration
	proc            *proc.Process
	nonblocking     bool
	pcType          string
}

func NewTerm(p *proc.Process, interval int, nonblocking bool, pcType string) *Term {
	sum := widgets.NewParagraph()
	sum.PaddingTop = -1
	sum.PaddingLeft = -1
	sum.Border = false

	table := widgets.NewTable()
	table.FillRow = true
	table.PaddingLeft = -1
	table.PaddingTop = -1
	table.Border = false
	table.ColumnWidths = []int{10, 200}
	table.TextAlignment = ui.AlignLeft
	table.ColumnSeparator = false
	table.RowSeparator = false
	table.Rows = [][]string{TOP_HEADER}
	table.RowStyles[0] = ui.NewStyle(ui.ColorBlack, ui.ColorWhite)
	return &Term{summary: sum, top: table, refreshInterval: time.Duration(interval), proc: p, nonblocking: nonblocking, pcType: pcType}
}

func (t *Term) RefreshSummary() error {
	sum, err := t.proc.Summary(!t.nonblocking)
	if err != nil {
		return err
	}
	t.summary.Text = sum.String()
	ui.Render(t.summary)
	return nil
}

func (t *Term) RefreshTop() error {
	gs, err := t.proc.GetGoroutines(!t.nonblocking)
	if err != nil {
		return err
	}
	fnStats := t.aggregateGoroutines(gs)
	type kv struct {
		k string
		v int
	}
	result := make([]kv, 0, len(fnStats))
	for k, v := range fnStats {
		result = append(result, kv{k, v})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].v > result[j].v
	})

	t.top.Rows = [][]string{TOP_HEADER}
	for _, item := range result {
		t.top.Rows = append(t.top.Rows, []string{strconv.Itoa(item.v), item.k})
	}
	ui.Render(t.top)
	return nil
}

// func (t *Term) Collect(doneCh chan int, errCh chan error) {
// 	for {
// 		select {
// 		case <-doneCh:
// 			return
// 		default:
// 			gs, err := t.proc.GetGoroutines(!t.nonblocking)
// 			if err != nil {
// 				errCh <- err
// 				return
// 			}
// 			t.fnStats = aggregateGoroutines(gs)
// 			pause := time.Duration(1e9 / t.sampleRate)
// 			time.Sleep(pause * time.Nanosecond)
// 		}
// 	}
// }

func (t *Term) Refresh() error {
	if err := t.RefreshSummary(); err != nil {
		return err
	}
	if err := t.RefreshTop(); err != nil {
		return err
	}
	return nil
}

func (t *Term) Display() error {
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()
	// go t.Collect(doneCh, errCh)

	tWidth, _ := ui.TerminalDimensions()

	t.summary.SetRect(0, 0, tWidth, SUMMARY_HEIGHT)
	t.top.SetRect(0, SUMMARY_HEIGHT, tWidth, TOP_HEIGHT)
	if err := t.Refresh(); err != nil {
		return err
	}

	ticker := time.NewTicker(t.refreshInterval * time.Second)
	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return nil
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				t.summary.SetRect(0, 0, payload.Width, SUMMARY_HEIGHT)
				t.top.SetRect(0, SUMMARY_HEIGHT, payload.Width, TOP_HEIGHT)
				ui.Clear()
				ui.Render(t.summary, t.top)
			}
		case <-ticker.C:
			if err := t.Refresh(); err != nil {
				return err
			}
		}
	}
}

func (t *Term) aggregateGoroutines(gs []*proc.G) map[string]int {
	result := make(map[string]int)
	for _, g := range gs {
		fn := g.GetLocation(t.pcType).String()
		if _, ok := result[fn]; !ok {
			result[fn] = 1
		} else {
			result[fn]++
		}
	}
	return result
}
