package term

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

type Term struct {
}

func (t *Term) Display() error {

	if err := ui.Init(); err != nil {
		return err
	}
	defer ui.Close()

	p := widgets.NewParagraph()
	//p.Border = false
	p.Text = "stats ..."
	p.SetRect(0, 0, 35, 5)
	ui.Render(p)

	table := widgets.NewTable()
	table.Rows = [][]string{
		[]string{"goroutine id", "thread id", "status", "wait reason"},
	}
	table.SetRect(0, 5, 100, 50)
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
