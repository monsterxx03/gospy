package term

import (
	"strconv"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"

	"gospy/pkg/proc"
)

type Term struct {
}

func (t *Term) Display(sumLines []string, gs []*proc.G) error {

	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	//p.Border = false
	p.Text = strings.Join(sumLines, "\n")
	p.SetRect(0, 0, 100, 5)
	ui.Render(p)

	table := widgets.NewTable()
	table.Rows = [][]string{
		[]string{"goroutine id", "thread id", "status", "wait reason", "func", "line"},
	}
	for _, g := range gs {
		tid := ""
		if g.ThreadID() != 0 {
			tid = strconv.FormatUint(g.ThreadID(), 10)
		}
		table.Rows = append(table.Rows, []string{strconv.FormatUint(g.ID, 10), tid, g.Status.String(), g.WaitReason.String(), g.StartLoc.Func, strconv.Itoa(g.GoLoc.Line)})
	}
	table.SetRect(0, 5, 150, 50)
	ui.Render(table)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return nil
		}
	}
}
