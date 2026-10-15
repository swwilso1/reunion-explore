package cache

import (
	"fmt"
	"os"

	"github.com/kedoco/reunion-explore/internal/binutil"
	"github.com/kedoco/reunion-explore/model"
)

// ParseShNames parses the shNames.cache file (searchable full names).
// Format: size(4) + count(u32) + "10hSan"(6) + padding to offset 0x20,
// then count records of length(u16, including itself) + name.
func ParseShNames(path string) ([]model.SearchName, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading shNames.cache: %w", err)
	}

	const recordsStart = 0x20
	if len(data) < recordsStart {
		return nil, fmt.Errorf("shNames.cache too short: %d bytes", len(data))
	}

	count, _ := binutil.U32LE(data, 4)
	names := make([]model.SearchName, 0, min(int(count), len(data)/2))
	for pos := recordsStart; pos+2 <= len(data) && len(names) < int(count); {
		recLen, _ := binutil.U16LE(data, pos)
		if recLen < 2 || pos+int(recLen) > len(data) {
			break
		}
		if name := string(data[pos+2 : pos+int(recLen)]); name != "" {
			names = append(names, model.SearchName{Name: name})
		}
		pos += int(recLen)
	}

	return names, nil
}
