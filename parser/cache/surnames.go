package cache

import (
	"fmt"
	"os"

	"github.com/kedoco/reunion-explore/internal/binutil"
	"github.com/kedoco/reunion-explore/model"
)

// ParseSurnames parses the surnames.cache file.
// Format: size(4) + "10ns"(4) + u16 + count(u16), then count records of
// len(u8) + flags(u8) + u16 + name. The name occupies the rest of the
// record minus (flags>>4)&3 padding bytes.
func ParseSurnames(path string) ([]model.SurnameEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading surnames.cache: %w", err)
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("surnames.cache too short: %d bytes", len(data))
	}

	magic := string(data[4:8])
	if magic != "10ns" {
		return nil, fmt.Errorf("surnames.cache: unexpected magic %q", magic)
	}

	count, _ := binutil.U16LE(data, 10)
	entries := make([]model.SurnameEntry, 0, count)
	for pos := 12; pos < len(data) && len(entries) < int(count); {
		recLen := int(data[pos])
		if recLen < 4 || pos+recLen > len(data) {
			break
		}
		pad := int(data[pos+1]>>4) & 3
		name := data[pos+4 : pos+recLen]
		if pad <= len(name) {
			name = name[:len(name)-pad]
		}
		entries = append(entries, model.SurnameEntry{Surname: string(name)})
		pos += recLen
	}

	return entries, nil
}
