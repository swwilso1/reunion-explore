package familydata

import (
	"bytes"
	"strings"

	reunion "github.com/kedoco/reunion-explore"
	"github.com/kedoco/reunion-explore/internal/binutil"
	"github.com/kedoco/reunion-explore/model"
)

// isSourceFieldTag reports whether tag holds a source template field.
// Fields are numbered from 0x03E8 in template order.
func isSourceFieldTag(tag uint16) bool {
	return tag >= 0x03E8 && tag < 0x0400
}

// extractSourceFieldValue returns the text of a source field. The field
// shares the event layout (value sub-TLV at offset 18), but unlike events
// has no date sub-TLV, so short values such as "0" are kept.
func extractSourceFieldValue(fieldData []byte) string {
	const pos = 18
	subLen, err := binutil.U16LE(fieldData, pos)
	if err != nil || subLen < 4 || pos+int(subLen) > len(fieldData) {
		return ""
	}
	if subTag, _ := binutil.U16LE(fieldData, pos+2); subTag != 0 {
		return ""
	}
	return strings.TrimSpace(string(bytes.ReplaceAll(fieldData[pos+4:pos+int(subLen)], []byte{0}, nil)))
}

// ParseSource parses a 0x20D0 source record from familydata.
func ParseSource(rec RawRecord, ec *reunion.ErrorCollector) (*model.Source, error) {
	s := &model.Source{
		ID:     rec.ID,
		SeqNum: rec.SeqNum,
	}

	if len(rec.Data) < 6 {
		return s, nil
	}

	fields := ParseTLVFields(rec.Data)

	// First pass: look for tag 0x0014 (display name) for the title
	for _, f := range fields {
		if isSourceFieldTag(f.Tag) {
			if v := extractSourceFieldValue(f.Data); v != "" {
				s.Fields = append(s.Fields, model.SourceField{
					SchemaID: ParseEventField(f.Data),
					Value:    v,
				})
			}
		}
		if f.Tag == TagDisplayName {
			str := cleanString(f.Data)
			if len(str) > 0 {
				s.Title = str
			}
		}
		s.RawFields = append(s.RawFields, model.RawField{
			Tag:  f.Tag,
			Data: f.Data,
			Size: uint16(len(f.Data) + 4),
		})
	}

	// Fallback: if no 0x0014 tag, use first non-empty string
	if s.Title == "" {
		for _, f := range fields {
			str := cleanString(f.Data)
			if len(str) > 2 {
				s.Title = str
				break
			}
		}
	}

	return s, nil
}
