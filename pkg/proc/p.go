package proc

type MCache struct {
	common
	TinyOffset  uint64 `name:"tinyoffset"`
	NTinyallocs uint64 `name:"local_tinyallocs"`

	Alloc    [numSpanClasses]*MSpan `name:"alloc" binStrt:"runtime.mspan"`
	FlushGen uint32                 `name:"flushGen"`
}

func (c *MCache) Parse(addr uint64) error {
	return parse(addr, c)
}

// P (processor) is runtime.p struct
type P struct {
	common
	ID          int32   `name:"id"`
	Status      pstatus `name:"status"`
	Schedtick   uint32  `name:"schedtick"`
	Syscalltick uint32  `name:"syscalltick"`
	M           *M      `name:"m" binStrt:"runtime.m"`
	MCache      *MCache `name:"mcache" binStrt:"runtime.mcache"`
	Runq        []byte  `name:"runq"`
	Runqsize    int
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
			g.Init(process, process.bin.GStruct)
			if err := g.Parse(gaddr); err != nil {
				return err
			}
			if !g.Dead() {
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
		_p.Init(p.Process(), p.BinStrt())
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
