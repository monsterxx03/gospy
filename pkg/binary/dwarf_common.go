package binary

import (
	"debug/dwarf"
	"fmt"
	"sync"
)

type dwarfer interface {
	DWARF() (*dwarf.Data, error)
}

type dwarfLoader struct {
	once        sync.Once
	data        *dwarf.Data
	err         error
	file        dwarfer
	offsetCache map[string]uint64 // key: "structName.fieldName"
}

func newDwarfLoader(file dwarfer) *dwarfLoader {
	return &dwarfLoader{file: file, offsetCache: make(map[string]uint64)}
}

func (d *dwarfLoader) load() (*dwarf.Data, error) {
	d.once.Do(func() {
		d.data, d.err = d.file.DWARF()
	})
	return d.data, d.err
}

func (d *dwarfLoader) HasDWARF() bool {
	_, err := d.load()
	return err == nil
}

func (d *dwarfLoader) GetStructOffset(typeName, fieldName string) (uint64, error) {
	// Check cache first
	cacheKey := typeName + "." + fieldName
	if offset, ok := d.offsetCache[cacheKey]; ok {
		return offset, nil
	}

	dwarfData, err := d.load()
	if err != nil {
		return 0, fmt.Errorf("DWARF unavailable: %w", err)
	}

	reader := dwarfData.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			return 0, err
		}
		if entry == nil {
			break
		}

		if entry.Tag == dwarf.TagStructType {
			if name, _ := entry.Val(dwarf.AttrName).(string); name == typeName {
				offset, err := d.findFieldOffset(reader, fieldName)
				if err != nil {
					return 0, err
				}
				d.offsetCache[cacheKey] = offset
				return offset, nil
			}
		}
	}
	return 0, fmt.Errorf("type %s not found", typeName)
}

func (d *dwarfLoader) findFieldOffset(reader *dwarf.Reader, fieldName string) (uint64, error) {
	for {
		entry, err := reader.Next()
		if err != nil {
			return 0, err
		}
		if entry == nil || entry.Tag == 0 {
			break // End of current struct
		}

		if entry.Tag == dwarf.TagMember {
			if name, _ := entry.Val(dwarf.AttrName).(string); name == fieldName {
				if offset, ok := entry.Val(dwarf.AttrDataMemberLoc).(int64); ok {
					return uint64(offset), nil
				}
				return 0, fmt.Errorf("offset not found for field %s", fieldName)
			}
		}
	}
	return 0, fmt.Errorf("field %s not found", fieldName)
}

func (d *dwarfLoader) GetStructSize(typeName string) (uint64, error) {
	// Check cache first
	if size, ok := d.offsetCache[typeName+".size"]; ok {
		return size, nil
	}

	dwarfData, err := d.load()
	if err != nil {
		return 0, fmt.Errorf("DWARF unavailable: %w", err)
	}

	reader := dwarfData.Reader()
	for {
		entry, err := reader.Next()
		if err != nil {
			return 0, err
		}
		if entry == nil {
			break
		}

		if entry.Tag == dwarf.TagStructType {
			name, ok := entry.Val(dwarf.AttrName).(string)
			if ok && name == typeName {
				if size, ok := entry.Val(dwarf.AttrByteSize).(int64); ok {
					// Cache the result
					d.offsetCache[typeName+".size"] = uint64(size)
					return uint64(size), nil
				}
			}
		}
	}

	return 0, fmt.Errorf("struct type %q not found", typeName)
}

func (d *dwarfLoader) GetNestedOffset(outerType, outerField, innerField string) (uint64, error) {
	outerOffset, err := d.GetStructOffset(outerType, outerField)
	if err != nil {
		return 0, err
	}

	innerOffset, err := d.GetStructOffset(outerType+"."+outerField, innerField)
	if err != nil {
		return 0, err
	}

	return outerOffset + innerOffset, nil
}
