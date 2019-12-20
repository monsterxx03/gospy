package proc

import (
	"encoding/json"
	"fmt"
	"sort"
)

type MCache struct {
	common
	TinyOffset  uint64 `name:"tinyoffset"`
	NTinyallocs uint64 `name:"local_tinyallocs"`

	Alloc [numSpanClasses]*MSpan `name:"alloc" binStrt:"runtime.mspan"`

	LargeFree  uint64 `name:"local_largefree"`  // bytes freed for large objects (>maxsmallsize)
	NLargeFree uint64 `name:"local_nlargefree"` // number of frees for large objects (>maxsmallsize)
	FlushGen   uint32 `name:"flushGen"`         // added on 1.12
}

type smallsize struct {
	sc         spanClass
	npages     uint64
	allocCount uint16
}

func (c *MCache) SmallSizeObjectSummary() []smallsize {
	summary := make(map[uint64]smallsize)
	// group by spanclass size
	for _, m := range c.Alloc {
		if !m.Active() {
			continue
		}
		sum, ok := summary[m.SpanClass.Size()]
		if !ok {
			summary[m.SpanClass.Size()] = smallsize{sc: m.SpanClass, npages: m.Npages, allocCount: m.AllocCount}
		} else {
			sum.npages += m.Npages
			sum.allocCount += m.AllocCount
		}
	}
	// sort size key
	sizes := make([]uint64, 0, len(summary))
	for size := range summary {
		sizes = append(sizes, size)
	}
	sort.Slice(sizes, func(i, j int) bool {
		return sizes[i] < sizes[j]
	})
	// sosrt by size
	scs := make([]smallsize, 0, len(sizes))
	for _, size := range sizes {
		scs = append(scs, summary[size])
	}
	return scs
}

func (c *MCache) Parse(addr uint64) error {
	return parse(addr, c)
}

// P (processor) is runtime.p struct
type P struct {
	common
	ID          int32   `name:"id" json:"id"`
	Status      pstatus `name:"status" json:"status"`
	Schedtick   uint32  `name:"schedtick" json:"schedtick"`
	Syscalltick uint32  `name:"syscalltick" json:"syscalltick"`
	M           *M      `name:"m" binStrt:"runtime.m" json:"m"`
	MCache      *MCache `name:"mcache" binStrt:"runtime.mcache" json:"-"`
	Runq        []byte  `name:"runq" json:"-"`
	LocalGQ     []*G   `json:"localgq"`
	Runqsize    int `json:"runqsize"`
}

func (p *P) MarshalJSON() ([]byte, error) {
	type Alias P
	return json.Marshal(&struct {
		Name string `json:"name"`
		Status string `json:"status"`
		*Alias
	}{
		Name: fmt.Sprintf("P%d", p.ID),
		Status: p.Status.String(),
		Alias: (*Alias)(p),
	})
}

func (p *P) Parse(addr uint64) error {
	if err := parse(addr, p); err != nil {
		return err
	}
	runqsize := 0
	process := p.Process()
	for i := 0; i < len(p.Runq); i += POINTER_SIZE {
		gaddr := toUint64(p.Runq[i : i+POINTER_SIZE])
		if gaddr != 0 {
			// should cache g by gaddr during one snapshot
			g := new(G)
			g.Init(process, process.bin.GStruct, gaddr)
			if err := g.Parse(gaddr); err != nil {
				return err
			}
			if !g.Dead() {
				p.LocalGQ = append(p.LocalGQ, g)
				runqsize++
			}
		}
	}
	p.Runqsize = runqsize
	return nil
}

func (p *P) ParsePtrSlice(addr uint64) ([]*P, error) {
	res, err := parseSliceAt(p.Process(), addr)
	if err != nil {
		return nil, err
	}
	result := make([]*P, 0, len(res))
	for _, addr := range res {
		_p := new(P)
		_p.Init(p.Process(), p.BinStrt(), addr)
		if err := _p.Parse(addr); err != nil {
			return nil, err
		}
		result = append(result, _p)
	}
	return result, nil
}

func (p *P) Idle() bool {
	return p.Status == pidle
}

func (p *P) Running() bool {
	return p.Status == prunning
}

func (p *P) Syscall() bool {
	return p.Status == psyscall
}

func (p *P) Gcstop() bool {
	return p.Status == pgcstop
}

func (p *P) Dead() bool {
	return p.Status == pdead
}
