package term

import (
	"sort"
	"strconv"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"gospy/pkg/proc"
)

type sampleStats struct {
	count uint64
	total uint64
}

const (
	SUMMARY_HEIGHT = 5
	TOP_HEIGHT     = 50
)

var TOP_HEADER = []string{"COUNT", "FUNC"}

type fnStat struct {
	fn    string
	count int
}

type Term struct {
	summary     *widgets.Paragraph
	top         *widgets.Table
	proc        *proc.Process
	sampleRate  int
	nonblocking bool

	stats   *sampleStats
	fnStats map[string]*fnStat
}

func NewTerm(p *proc.Process, rate int, nonblocking bool) *Term {
	sum := widgets.NewParagraph()
	sum.PaddingTop = -1
	sum.PaddingLeft = -1
	sum.Border = false

	table := widgets.NewTable()
	table.Border = false
	table.RowSeparator = false
	table.Rows = [][]string{TOP_HEADER}
	return &Term{summary: sum, top: table, proc: p, sampleRate: rate, nonblocking: nonblocking, stats: new(sampleStats), fnStats: make(map[string]*fnStat)}
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
	result := make([]*fnStat, 0, len(t.fnStats))
	for _, val := range t.fnStats {
		result = append(result, val)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].count > result[j].count
	})
	t.top.Rows = [][]string{TOP_HEADER}
	for _, row := range result {
		t.top.Rows = append(t.top.Rows, []string{strconv.Itoa(row.count), row.fn})
	}
	ui.Render(t.top)
	return nil
}

func (t *Term) Collect(doneCh chan int, errCh chan error) {
	for {
		select {
		case <-doneCh:
			return
		default:
			gs, err := t.proc.GetGoroutines(!t.nonblocking)
			if err != nil {
				errCh <- err
				return
			}
			for k, v := range aggregateGoroutines(gs) {
				if _, ok := t.fnStats[k]; !ok {
					t.fnStats[k] = v
				} else {
					t.fnStats[k].count += v.count
				}
			}
			pause := time.Duration(1e9 / t.sampleRate)
			time.Sleep(pause * time.Nanosecond)
		}
	}
}

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
	errCh := make(chan error)
	doneCh := make(chan int)
	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()
	go t.Collect(doneCh, errCh)

	tWidth, _ := ui.TerminalDimensions()

	t.summary.SetRect(0, 0, tWidth, SUMMARY_HEIGHT)
	t.top.SetRect(0, SUMMARY_HEIGHT, tWidth, TOP_HEIGHT)
	t.Refresh()

	ticker := time.NewTicker(1 * time.Second)
	uiEvents := ui.PollEvents()
	for {
		select {
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				doneCh <- 1
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
		case err := <-errCh:
			return err
		}
	}
}

func aggregateGoroutines(gs []*proc.G) map[string]*fnStat {
	result := make(map[string]*fnStat)
	for _, g := range gs {
		fn := g.StartLoc.String()
		if _, ok := result[fn]; !ok {
			result[fn] = &fnStat{fn: fn, count: 1}
		} else {
			result[fn].count++
		}
	}
	return result
}
