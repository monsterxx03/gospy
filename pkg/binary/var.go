package binary

import (
	"fmt"
)

type Var interface {
	String() string
	GetAddr() uint64
}

type CommonType struct {
	Name string
	Addr uint64
	Size int64
	Value interface{}
}

func (c *CommonType) GetAddr() uint64 {
	return c.Addr
}

type UintVar struct {
	CommonType
}

func (u *UintVar) String() string {
	return fmt.Sprintf("type: uint%d, value: %v", u.Size*8, u.Value)
}

type IntVar struct {
	CommonType
}

func (i *IntVar) String() string {
	return fmt.Sprintf("type: int%d, value: %v", i.Size*8, i.Value)
}

type BoolVar struct {
	CommonType
}

func (b *BoolVar) String() string {
	return fmt.Sprintf("type: bool, value: %v", b.Value)
}

type StringVar struct {
	CommonType
}

func (s *StringVar) String() string {
	return fmt.Sprintf("type: string, value: %v", s.Value)
}

type PtrVar struct {
	CommonType
	Type Var
}

func (b *PtrVar) String() string {
	return "*" + b.Type.String()
}

