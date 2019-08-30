package procmaps

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type Range struct {
	Start    uint64
	End      uint64
	Perm     string
	Offset   uint64
	Dev      string
	Inode    uint64
	Filename string
}

func (r *Range) Size() uint64 {
	return r.End - r.Start
}

func (r *Range) IsRead() bool {
	return r.Perm[0] == 'r'
}

func (r *Range) IsWrite() bool {
	return r.Perm[1] == 'w'
}

func (r *Range) IsExe() bool {
	return r.Perm[2] == 'x'
}

func (r *Range) IsPrivate() bool {
	return r.Perm[3] == 'p'
}

func (r *Range) IsShare() bool {
	return r.Perm[3] == 's'
}

func ReadProcMaps(pid int) ([]Range, error) {
	return parseProcMaps(fmt.Sprintf("/proc/%d/maps", pid))
}

func parseProcMaps(path string) ([]Range, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	result := make([]Range, 0)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		splits := strings.Fields(line)
		if len(splits) != 5 && len(splits) != 6 {
			return nil, fmt.Errorf("invalid map range: %s", line)
		}
		rangeSplit := strings.Split(splits[0], "-")
		start, err := strconv.ParseUint(rangeSplit[0], 16, 64)
		if err != nil {
			return nil, err
		}
		end, err := strconv.ParseUint(rangeSplit[1], 16, 64)
		if err != nil {
			return nil, err
		}
		perm := splits[1]
		offset, err := strconv.ParseUint(splits[2], 16, 64)
		if err != nil {
			return nil, err
		}
		dev := splits[3]
		inode, err := strconv.Atoi(splits[4])
		if err != nil {
			return nil, err
		}
		filename := ""
		if len(splits) == 6 {
			filename = splits[5]
		}
		result = append(result, Range{Start: uint64(start),
			End: uint64(end), Perm: perm,
			Offset: uint64(offset), Dev: dev,
			Inode: uint64(inode), Filename: filename,
		})
	}
	return result, nil
}
