package proc

import (
	"fmt"
	"reflect"

	gbin "gospy/pkg/binary"
)

// return a slice of ptr addr
func parseSliceAt(p *Process, baseAddr uint64) ([]uint64, error) {
	data := make([]byte, 3*POINTER_SIZE)
	// read slice header
	if err := p.ReadData(data, baseAddr); err != nil {
		return nil, err
	}
	arrptr := toUint64(data[:POINTER_SIZE])
	slen := toUint64(data[POINTER_SIZE : POINTER_SIZE*2])
	scap := toUint64(data[POINTER_SIZE*2 : POINTER_SIZE*3])

	data = make([]byte, slen*POINTER_SIZE)
	if err := p.ReadData(data, arrptr); err != nil {
		return nil, err
	}
	result := make([]uint64, 0, scap)
	for i := uint64(0); i < slen; i++ {
		idx := i * POINTER_SIZE
		addr := toUint64(data[idx : idx+POINTER_SIZE])
		result = append(result, addr)
	}
	return result, nil
}

func getBinStrtFromField(p *Process, field reflect.StructField) (*gbin.Strt, error) {
	bname := field.Tag.Get("binStrt")
	if bname == "" {
		return nil, fmt.Errorf("pointer field %+v don't have `binStrt` tag", field)
	}
	bstrt, ok := p.bin.StrtMap[bname]
	if !ok {
		return nil, fmt.Errorf("can't find %s in p.bin", bname)
	}
	return bstrt, nil
}

// parse will fill fields in `obj` by reading memory start from `baseAddr`
// XX: reflect is disguesting, but ...
func parse(baseAddr uint64, obj GoStructer) error {
	p := obj.Process()
	binStrt := obj.BinStrt()
	data := make([]byte, binStrt.Size)
	if err := p.ReadData(data, baseAddr); err != nil {
		return err
	}
	members := binStrt.Members
	t := reflect.TypeOf(obj).Elem()
	v := reflect.ValueOf(obj).Elem()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Tag.Get("name")
		if name != "" {
			strtField := members[name]
			addr := uint64(strtField.StrtOffset)
			size := uint64(strtField.Size)
			// fill obj's fields
			switch field.Type.Kind() {
			case reflect.Struct:
				// eg: G.Sched
				bstrt, err := getBinStrtFromField(p, field)
				if err != nil {
					return err
				}
				strt := reflect.New(v.Field(i).Type())
				strt.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(p), reflect.ValueOf(bstrt)})
				if err := parse(baseAddr+addr, strt.Interface().(GoStructer)); err != nil {
					return err
				}
				v.Field(i).Set(strt.Elem())
			case reflect.Ptr:
				// eg: G.M
				_addr := toUint64(data[addr : addr+size])
				if _addr == 0 {
					continue
				}
				bstrt, err := getBinStrtFromField(p, field)
				if err != nil {
					return err
				}
				strt := reflect.New(v.Field(i).Type().Elem())
				// call Init dynamically
				strt.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(p), reflect.ValueOf(bstrt)})
				// recursive parse to fillin  instance
				if err := parse(_addr, strt.Interface().(GoStructer)); err != nil {
					return err
				}
				v.Field(i).Set(strt)
			case reflect.Uint64:
				f := toUint64(data[addr : addr+size])
				v.Field(i).SetUint(f)
			case reflect.Uint32:
				f := toUint32(data[addr : addr+size])
				v.Field(i).SetUint(uint64(f))
			case reflect.Uint16:
				f := toUint16(data[addr : addr+size])
				v.Field(i).SetUint(uint64(f))
			case reflect.Uint8:
				f := uint8(data[addr])
				v.Field(i).SetUint(uint64(f))
			case reflect.Int32:
				f := toInt32(data[addr : addr+size])
				v.Field(i).SetInt(int64(f))
			case reflect.Float64:
				f := toFloat64(data[addr : addr+size])
				v.Field(i).SetFloat(f)
			case reflect.Slice:
				// check on slice item type
				switch field.Type.Elem().Kind() {
				case reflect.Uint8:
					// eg: P.Runq
					v.Field(i).SetBytes(data[addr : addr+size])
					continue
				case reflect.Ptr:
					// eg: MHeap.MSpans
					if size != POINTER_SIZE*3 {
						// arrayptr + len + cap
						return fmt.Errorf("Invalid size %d for slice of pointer", size)
					}
					arrayptr := toUint64(data[addr : addr+POINTER_SIZE])
					slen := toUint64(data[addr+POINTER_SIZE : addr+POINTER_SIZE*2])
					scap := toUint64(data[addr+POINTER_SIZE*2 : addr+POINTER_SIZE*3])
					slice := reflect.MakeSlice(reflect.SliceOf(field.Type.Elem()), 0, int(scap))

					bstrt, err := getBinStrtFromField(p, field)
					if err != nil {
						return err
					}
					sliceData := make([]byte, slen*POINTER_SIZE)
					// bulk read array data on arrayptr
					if err := p.ReadData(sliceData, arrayptr); err != nil {
						return err
					}
					for j := uint64(0); j < slen; j++ {
						// rebuild slice items
						strt := reflect.New(field.Type.Elem().Elem())
						// call Init dynamically
						strt.MethodByName("Init").Call([]reflect.Value{reflect.ValueOf(p), reflect.ValueOf(bstrt)})
						idx := j * POINTER_SIZE
						if err := parse(toUint64(sliceData[idx:idx+POINTER_SIZE]), strt.Interface().(GoStructer)); err != nil {
							return err
						}
						slice = reflect.Append(slice, strt)
					}
					v.Field(i).Set(slice)
					continue
				default:
					return fmt.Errorf("Unsupport slice item %+v", field)
				}
			default:
				return fmt.Errorf("unsupport %+v, type: %s", field, field.Type.Kind())
			}
		}
	}

	return nil
}
