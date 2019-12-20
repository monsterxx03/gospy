package proc

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	gbin "gospy/pkg/binary"
)

type stack struct {
	common
	Lo uint64 `name:"lo"`
	Hi uint64 `name:"hi"`
}

func (s *stack) Parse(addr uint64) error {
	return parse(addr, s)
}

// Sudog is runtime.sudog, when a g entering waiting state, it's attached on sudog
type Sudog struct {
	common
	IsSelect    bool   `name:"isSelect"`
	Ticket      uint32 `name:"ticket"`
	AcquireTime int64  `name:"acquiretime"`
	ReleaseTime int64  `name:"releasetime"`
	C           *HChan `name:"c" binStrt:"runtime.hchan"`
}

func (s *Sudog) Parse(addr uint64) error {
	return parse(addr, s)
}

// HChan is runtime.hchan, result of make(chan xx)
type HChan struct {
	common
	QCount    uint   `name:"qcount"`
	DataqSize uint   `name:"dataqsiz"`
	ElemSize  uint16 `name:"elemsize"`
	ElemType  *Type  `name:"elemtype" binStrt:"runtime._type"`
	Closed    uint32 `name:"closed"`
	Sendx     uint   `name:"sendx"`
	Recvx     uint   `name:"recvx"`
}

func (h *HChan) Parse(addr uint64) error {
	return parse(addr, h)
}

func (h *HChan) String() string {
	return fmt.Sprintf("make(chan %s(%d), %d)", h.ElemType, h.ElemSize, h.DataqSize)
}

type GSummary struct {
	ID         string
	Status     string
	WaitReason string
	Loc        string
}

// G is runtime.g struct parsed from process memory and binary dwarf
type G struct {
	common
	Stack        stack          `name:"stack" binStrt:"runtime.stack" json:"-"`
	ID           uint64         `name:"goid" json:"id"`
	Status       gstatus        `name:"atomicstatus" json:"status"`
	WaitReason   gwaitReason    `name:"waitreason" json:"waitreason"`   // if Status ==Gwaiting
	Sched        Gobuf          `name:"sched" binStrt:"runtime.gobuf" json:"-"`
	Startpc      uint64         `name:"startpc" json:"-"`
	Gopc         uint64         `name:"gopc" json:"-"`
	M            *M             `name:"m" binStrt:"runtime.m"` // hold worker thread info
	WaitingSudog *Sudog         `name:"waiting" binStrt:"runtime.sudog" json:"-"`
	CurLoc       *gbin.Location `json:"-"` // runtime location
	UserLoc      *gbin.Location `json:"-"` // location of user code, a subset of CurLoc
	GoLoc        *gbin.Location `json:"-"` // location of `go` statement that spawed this goroutine
	StartLoc     *gbin.Location  `json:"-"`// location of goroutine start function
}

func (g *G) MarshalJSON() ([]byte, error) {
	type Alias G
	wr, err := g.GetWaitReason()
	if err != nil {
		return nil, err
	}
	return json.Marshal(&struct{
		Status string `json:"status"`
		WaitReason string `json:"waitreason"`
		*Alias
	}{
		Status: g.Status.String(),
		WaitReason: wr,
		Alias: (*Alias)(g),
	})
}

func (g *G) Summary(pcType string) (*GSummary, error) {
	var id, status, chanStr, reason string
	var err error
	chanStr, err = g.GetWaitingChan()
	if err != nil {
		return nil, err
	}
	status = g.Status.String()

	if g.Waiting() {
		reason, err = g.GetWaitReason()
		if err != nil {
			return nil, err
		}
		chanStr, err = g.GetWaitingChan()
		if err != nil {
			return nil, err
		}
		if chanStr != "" {
			reason = chanStr
		}
	}
	if g.M != nil {
		id = fmt.Sprintf("%d(M%d)", g.ID, g.M.ID)
	} else {
		id = strconv.Itoa(int(g.ID))
	}
	s := &GSummary{
		ID:         id,
		Status:     status,
		WaitReason: reason,
		Loc:        g.GetLocation(pcType).String(),
	}
	return s, nil
}

func (g *G) GetWaitReason() (string, error) {
	version, err := g.Process().GoVersion()
	if err != nil {
		return "", err
	}
	return getWaitReasonMap(version)[g.WaitReason], nil
}

func (g *G) GetWaitingChan() (string, error) {
	if g.WaitingSudog != nil {
		if g.WaitingSudog.IsSelect {
			return "select <- " + g.WaitingSudog.C.String(), nil
		}
		reason, err := g.GetWaitReason()
		if err != nil {
			return "", err
		}
		if strings.Contains(reason, "chan receive") {
			return "<- " + g.WaitingSudog.C.String(), nil
		}
		if strings.Contains(reason, "chan send") {
			return "-> " + g.WaitingSudog.C.String(), nil
		}
		return g.WaitingSudog.C.String(), nil
	}
	return "", nil
}

func (g *G) StackSize() uint64 {
	return g.Stack.Hi - g.Stack.Lo
}

func (g *G) Parse(addr uint64) error {
	if err := parse(addr, g); err != nil {
		return err
	}
	if g.Status == gdead {
		return nil
	}
	p := g.Process()
	g.CurLoc = p.getLocation(g.Sched.PC)
	g.GoLoc = p.getLocation(g.Gopc)
	g.StartLoc = p.getLocation(g.Startpc)
	return nil
}

func (g *G) ParsePtrSlice(addr uint64) ([]*G, error) {
	res, err := parseSliceAt(g.Process(), addr)
	if err != nil {
		return nil, err
	}
	result := make([]*G, 0, len(res))
	for _, addr := range res {
		_g := new(G)
		_g.Init(g.Process(), g.BinStrt(), addr)
		if err := _g.Parse(addr); err != nil {
			return nil, err
		}
		if _g.Dead() {
			continue
		}
		result = append(result, _g)
	}
	return result, nil
}

func (g *G) GetLocation(pcType string) *gbin.Location {
	switch pcType {
	case "current":
		return g.CurLoc
	case "caller":
		return g.GoLoc
	case "start":
		return g.StartLoc
	default:
		return nil
	}
}

func (g *G) Idle() bool {
	return g.Status == gidle
}

func (g *G) Running() bool {
	return g.Status == grunning
}

func (g *G) Syscall() bool {
	return g.Status == gsyscall
}

func (g *G) Waiting() bool {
	// waiting means this goroutine is blocked in runtime.
	return g.Status == gwaiting
}

func (g *G) Dead() bool {
	// dead means this goroutine is not executing user code.
	// Maybe exited, or just being initialized.
	return g.Status == gdead
}

func (g *G) ThreadID() uint64 {
	if g.M == nil {
		return 0
	}
	return g.M.ID
}
