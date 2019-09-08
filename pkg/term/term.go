package term

import (
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
	p.Border = false
	tWidth, _ := ui.TerminalDimensions()
	p.SetRect(0, 0, tWidth, 5)
	ui.Render(p)

	table := widgets.NewTable()
	table.Border = false
	table.RowSeparator = false
	table.Rows = [][]string{
		[]string{"func", "count"},
	}
	for _, g := range gs {
		table.Rows = append(table.Rows, []string{g.GoLoc.String()})
	}
	table.SetRect(0, 3, tWidth, 50)
	ui.Render(table)

	uiEvents := ui.PollEvents()
	for {
		e := <-uiEvents
		switch e.ID {
		case "q", "<C-c>":
			return nil
		case "<Resize>":
			payload := e.Payload.(ui.Resize)
			p.SetRect(0, 0, payload.Width, 5)
			table.SetRect(0, 5, payload.Width, 50)
			ui.Clear()
			ui.Render(p, table)
		}
	}
}
