package cache

import (
	"fmt"
	"os"

	"github.com/kedoco/reunion-explore/internal/binutil"
	"github.com/kedoco/reunion-explore/model"
)

// ParseFmNames parses the fmnames.cache file (first/given names).
// Format: size(4) + magic(4) + count(4) = 12-byte header. The magic is
// "2wps" in Reunion 14 and "dwps" in Reunion 13.
// Then: offset table of count * uint32
// Each record at offset: size(1) + meta(5) + phonetic(2) + name_string,
// where size is the total record length including the size byte itself.
func ParseFmNames(path string) ([]model.FirstNameEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading fmnames.cache: %w", err)
	}

	if len(data) < 12 {
		return nil, fmt.Errorf("fmnames.cache too short: %d bytes", len(data))
	}

	count, _ := binutil.U32LE(data, 8)

	// Offset table starts at byte 12
	offsets, err := ReadOffsetTable(data, 12, int(count))
	if err != nil {
		return nil, fmt.Errorf("fmnames.cache offset table: %w", err)
	}

	entries := make([]model.FirstNameEntry, 0, count)
	for _, off := range offsets {
		o := int(off)
		if o >= len(data) {
			continue
		}
		recSize := int(data[o])
		if o+recSize > len(data) || recSize < 8 {
			continue
		}
		meta := make([]byte, 5)
		copy(meta, data[o+1:o+6])
		phonetic := string(data[o+6 : o+8])
		name := string(data[o+8 : o+recSize])

		entries = append(entries, model.FirstNameEntry{
			Name:     name,
			Meta:     meta,
			Phonetic: phonetic,
		})
	}

	return entries, nil
}
