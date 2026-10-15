package familydata

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kedoco/reunion-explore/internal/binutil"
	"github.com/kedoco/reunion-explore/model"
)

// placeRefPattern matches [[pt:NNN]] place references in event data.
var placeRefPattern = regexp.MustCompile(`\[\[pt:(\d+)\]\]`)

// TLVField represents a tag-length-value field extracted from record data.
type TLVField struct {
	Tag    uint16
	Offset int // offset within record data
	Data   []byte
}

// ParseTLVFields parses TLV fields from raw record data.
// Record data starts with a 6-byte preamble (4-byte timestamp + 2-byte repeated size),
// followed by fields in the format: total_length(u16 LE) + tag(u16 LE) + data(total_length - 4).
func ParseTLVFields(data []byte) []TLVField {
	if len(data) < 6 {
		return nil
	}

	// Skip preamble: 4-byte timestamp + 2-byte repeated size
	return parseTLVFieldsFrom(data[6:])
}

// parseTLVFieldsFrom parses TLV fields starting at an arbitrary position.
func parseTLVFieldsFrom(data []byte) []TLVField {
	var fields []TLVField
	pos := 0

	for pos+4 <= len(data) {
		totalLen, err := binutil.U16LE(data, pos)
		if err != nil {
			break
		}
		tag, err := binutil.U16LE(data, pos+2)
		if err != nil {
			break
		}

		// total_length includes the 4-byte header itself
		if totalLen < 4 {
			break
		}

		// A field running past the end of the data is not a real field;
		// Reunion 10 leaves uninitialized bytes after some records.
		dataLen := int(totalLen) - 4
		fieldEnd := pos + int(totalLen)
		if fieldEnd > len(data) {
			break
		}

		fieldData := data[pos+4 : pos+4+dataLen]

		fields = append(fields, TLVField{
			Tag:    tag,
			Offset: pos,
			Data:   fieldData,
		})

		pos = fieldEnd
	}

	return fields
}

// ParseEventField extracts the schema definition ID from event sub-data.
// Event fields (tags >= 0x100) contain nested data. After a 4-byte sub-header,
// the schema definition ID is at offset 12 within the sub-data as u16 LE.
func ParseEventField(fieldData []byte) (schemaID uint16) {
	// Sub-data starts after the 4-byte sub-header
	// Schema ID is at offset 12 within the sub-data (i.e., byte 16 from start of fieldData)
	// But per the plan: "offset 12 in sub-data (after the 4-byte sub-header)"
	// So: 4 (sub-header) + 12 = offset 16 from start of fieldData
	if len(fieldData) >= 18 {
		id, err := binutil.U16LE(fieldData, 16)
		if err == nil {
			schemaID = id
		}
	}
	return
}

// ExtractPlaceRefs finds all [[pt:NNN]] references in data and returns the place IDs.
func ExtractPlaceRefs(data []byte) []int {
	matches := placeRefPattern.FindAllSubmatch(data, -1)
	var refs []int
	for _, m := range matches {
		if len(m) >= 2 {
			n, err := strconv.Atoi(string(m[1]))
			if err == nil {
				refs = append(refs, n)
			}
		}
	}
	return refs
}

// ExtractDate decodes a date from event sub-data.
// Event data has an 18-byte fixed header, followed by sub-TLV fields.
// The first sub-TLV (at offset 18) contains the date when its total length == 8.
//
// Year and month group are encoded in bytes[24:25] as u16 LE:
//
//	totalQ = (year + 8000) * 4 + group
//	group: 0 → months 1-3, 1 → months 4-7, 2 → months 8-11, 3 → month 12
//
// Day and month offset are in byte[23]:
//
//	bits 7-6: month offset within the group (0-3)
//	bits 5-0: day of month (0 = unknown)
//	month = group*4 + offset  (1-12)
//
// Precision flags are in byte[22]:
//
//	0x00 = exact date
//	0xA0 = approximate ("about"), year-only
//	0x40 = "after"
//	0xE0 = "after", year-only
func ExtractDate(fieldData []byte) string {
	return extractDateAt(fieldData, eventSubFieldsOffset)
}

// eventSubFieldsOffset is where sub-TLVs start in a person or family event.
// Marriage fields use marriageSubFieldsOffset instead.
const eventSubFieldsOffset = 18

// extractDateAt decodes a date sub-TLV starting at off.
func extractDateAt(fieldData []byte, off int) string {
	if len(fieldData) < off+8 {
		return ""
	}
	// Date sub-TLV has exactly length 8 (4-byte header + 4-byte date)
	subLen, err := binutil.U16LE(fieldData, off)
	if err != nil || subLen != 8 {
		return ""
	}

	precFlags := fieldData[off+4]
	dayByte := fieldData[off+5]
	monthYearLo := fieldData[off+6]
	monthYearHi := fieldData[off+7]

	totalQ := int(monthYearHi)<<8 | int(monthYearLo)
	year := totalQ/4 - 8000

	// Month is split across two locations:
	//   - bits 1-0 of totalQ give the high 2 bits (group 0-3)
	//   - bits 7-6 of dayByte give the low 2 bits (offset 0-3 within group)
	// Combined: month = group*4 + offset, yielding 1-12.
	group := totalQ % 4
	month := group*4 + int(dayByte>>6)

	day := int(dayByte & 0x3F)

	if year < 1 || year > 9999 || month > 12 {
		return ""
	}
	// Month 0 with no day is a plain year (e.g. "1939").
	if month == 0 && day != 0 {
		return ""
	}

	monthNames := [13]string{"", "Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}

	// Precision flags encode qualifier and precision:
	//   bit 6 (0x40): "after" qualifier
	//   bits 7+5 (0xA0): year-only precision (month is a meaningless default)
	//   0x40 = "after" with month precision
	//   0xA0 = "about" with year-only precision
	//   0xE0 = "after" with year-only precision
	yearOnly := precFlags&0xA0 == 0xA0

	prefix := ""
	switch {
	case precFlags&0x40 != 0:
		prefix = "after "
	case yearOnly:
		prefix = "about "
	}

	if yearOnly || month == 0 {
		return fmt.Sprintf("%s%d", prefix, year)
	}
	if day > 0 && day <= 31 {
		return fmt.Sprintf("%s%d %s %d", prefix, day, monthNames[month], year)
	}
	return fmt.Sprintf("%s%s %d", prefix, monthNames[month], year)
}

// ExtractNoteRef extracts a note record ID from event sub-data.
// Note-type events (tag < 0x03E8) may contain a sub-TLV at offset 18
// with tag 0x0000 whose 4-byte data is the referenced note's record ID.
// Returns 0 if no note reference is found.
func ExtractNoteRef(fieldData []byte) uint32 {
	pos := 18
	for pos+4 <= len(fieldData) {
		subLen, err := binutil.U16LE(fieldData, pos)
		if err != nil || subLen < 4 {
			break
		}
		subTag, err := binutil.U16LE(fieldData, pos+2)
		if err != nil {
			break
		}
		if pos+int(subLen) > len(fieldData) {
			break
		}
		if subTag == 0x0000 && subLen == 8 {
			id, err := binutil.U32LE(fieldData, pos+4)
			if err == nil && id > 0 {
				return id
			}
		}
		pos += int(subLen)
	}
	return 0
}

// ExtractSourceCitations decodes the binary citation format:
//
//	Offset  Size   Description
//	0       4      innerLength (u32LE)
//	4       4      citationCount (u32LE)
//	Per entry:
//	  0     2      entryLength (u16LE, includes this field)
//	  2     2      unknown/hash (skip)
//	  4     4      sourceRecordID (u32LE)
//	  8     N      detail text (entryLength - 8 bytes), stripped of nulls
func ExtractSourceCitations(data []byte) []model.SourceCitation {
	if len(data) < 8 {
		return nil
	}
	_, err := binutil.U32LE(data, 0) // innerLength
	if err != nil {
		return nil
	}
	count, err := binutil.U32LE(data, 4)
	if err != nil || count == 0 {
		return nil
	}

	var citations []model.SourceCitation
	pos := 8
	for i := uint32(0); i < count && pos+8 <= len(data); i++ {
		entryLen, err := binutil.U16LE(data, pos)
		if err != nil || entryLen < 8 {
			break
		}
		if pos+int(entryLen) > len(data) {
			break
		}
		sourceID, err := binutil.U32LE(data, pos+4)
		if err != nil {
			break
		}
		detail := ""
		if entryLen > 8 {
			detailBytes := data[pos+8 : pos+int(entryLen)]
			detailBytes = bytes.ReplaceAll(detailBytes, []byte{0}, nil)
			detail = string(detailBytes)
		}
		citations = append(citations, model.SourceCitation{
			SourceID: sourceID,
			Detail:   detail,
		})
		pos += int(entryLen)
	}
	return citations
}

// ExtractEventSourceCitations walks the sub-TLVs at offset 18 in event data
// and passes the last sub-TLV's data to ExtractSourceCitations.
func ExtractEventSourceCitations(fieldData []byte) []model.SourceCitation {
	return extractEventSourceCitationsAt(fieldData, eventSubFieldsOffset)
}

func extractEventSourceCitationsAt(fieldData []byte, pos int) []model.SourceCitation {
	lastData := []byte(nil)
	for pos+4 <= len(fieldData) {
		subLen, err := binutil.U16LE(fieldData, pos)
		if err != nil || subLen < 4 {
			break
		}
		if pos+int(subLen) > len(fieldData) {
			break
		}
		lastData = fieldData[pos+4 : pos+int(subLen)]
		pos += int(subLen)
	}
	if lastData == nil {
		return nil
	}
	return ExtractSourceCitations(lastData)
}

// ExtractEventText walks sub-TLVs at offset 18 in event field data and returns
// the first text sub-TLV (tag 0x0000, length > 8, contains printable text).
// It strips [[pt:NNN]] place-ref markers from the result.
func ExtractEventText(fieldData []byte) string {
	return extractEventTextAt(fieldData, eventSubFieldsOffset)
}

func extractEventTextAt(fieldData []byte, pos int) string {
	for pos+4 <= len(fieldData) {
		subLen, err := binutil.U16LE(fieldData, pos)
		if err != nil || subLen < 4 {
			break
		}
		subTag, err := binutil.U16LE(fieldData, pos+2)
		if err != nil {
			break
		}
		if pos+int(subLen) > len(fieldData) {
			break
		}
		// Skip date sub-TLVs (length == 8) and non-zero tags
		if subTag == 0x0000 && subLen > 8 {
			raw := fieldData[pos+4 : pos+int(subLen)]
			// Memo text starts with a printable byte; citation sub-TLVs
			// start with binary header bytes (innerLength u32LE, count u32LE).
			// Skip sub-TLVs whose first non-null byte is a control character.
			firstPrintable := false
			for _, b := range raw {
				if b == 0 {
					continue
				}
				firstPrintable = b >= 0x20
				break
			}
			if !firstPrintable {
				pos += int(subLen)
				continue
			}
			// Strip null bytes
			raw = bytes.ReplaceAll(raw, []byte{0}, nil)
			// Strip [[pt:NNN]] place references
			text := placeRefPattern.ReplaceAllString(string(raw), "")
			text = strings.TrimSpace(text)
			if text != "" {
				return text
			}
		}
		pos += int(subLen)
	}
	return ""
}

// ExtractString finds the first run of printable characters (ASCII and valid
// UTF-8) in data. It uses unicode/utf8.DecodeRune for correct multi-byte
// handling, including rejection of overlong encodings and surrogates.
func ExtractString(data []byte) string {
	start := -1
	i := 0
	for i < len(data) {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size <= 1 {
			// Invalid UTF-8 byte — string boundary
			if start != -1 {
				return string(data[start:i])
			}
			i++
			continue
		}
		if r >= 0x20 && r != 0x7F && unicode.IsPrint(r) {
			if start == -1 {
				start = i
			}
			i += size
		} else {
			// Control char or non-printable — string boundary
			if start != -1 {
				return string(data[start:i])
			}
			i += size
		}
	}
	if start != -1 {
		return string(data[start:])
	}
	return ""
}
