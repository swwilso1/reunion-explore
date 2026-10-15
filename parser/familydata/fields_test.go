package familydata

import (
	"encoding/binary"
	"testing"
)

// putU16LE writes a little-endian uint16 into buf at the given offset.
func putU16LE(buf []byte, offset int, v uint16) {
	binary.LittleEndian.PutUint16(buf[offset:], v)
}

// putU32LE writes a little-endian uint32 into buf at the given offset.
func putU32LE(buf []byte, offset int, v uint32) {
	binary.LittleEndian.PutUint32(buf[offset:], v)
}

// makeTLVField builds a single TLV field: [totalLen u16LE][tag u16LE][data...].
func makeTLVField(tag uint16, data []byte) []byte {
	totalLen := uint16(len(data) + 4)
	buf := make([]byte, totalLen)
	putU16LE(buf, 0, totalLen)
	putU16LE(buf, 2, tag)
	copy(buf[4:], data)
	return buf
}

// makePreamble creates the 6-byte preamble: 4-byte timestamp + 2-byte size.
func makePreamble() []byte {
	return make([]byte, 6)
}

func TestParseTLVFields(t *testing.T) {
	// Build record data: 6-byte preamble + two TLV fields
	field1Data := []byte("Alice")
	field2Data := []byte{0x01}

	var data []byte
	data = append(data, makePreamble()...)
	data = append(data, makeTLVField(0x001E, field1Data)...) // given name
	data = append(data, makeTLVField(0x001B, field2Data)...) // sex

	fields := ParseTLVFields(data)
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}

	if fields[0].Tag != 0x001E {
		t.Errorf("field[0].Tag = 0x%04X, want 0x001E", fields[0].Tag)
	}
	if string(fields[0].Data) != "Alice" {
		t.Errorf("field[0].Data = %q, want %q", fields[0].Data, "Alice")
	}

	if fields[1].Tag != 0x001B {
		t.Errorf("field[1].Tag = 0x%04X, want 0x001B", fields[1].Tag)
	}
	if len(fields[1].Data) != 1 || fields[1].Data[0] != 0x01 {
		t.Errorf("field[1].Data = %v, want [0x01]", fields[1].Data)
	}
}

func TestParseTLVFields_Empty(t *testing.T) {
	// Too short for preamble
	if fields := ParseTLVFields(nil); fields != nil {
		t.Errorf("expected nil for nil data, got %v", fields)
	}
	if fields := ParseTLVFields([]byte{0, 0, 0}); fields != nil {
		t.Errorf("expected nil for short data, got %v", fields)
	}
}

func TestParseTLVFields_TotalLenLessThan4Stops(t *testing.T) {
	// A field with totalLen < 4 should stop parsing (used as sentinel)
	var data []byte
	data = append(data, makePreamble()...)
	data = append(data, makeTLVField(0x0001, []byte("ok"))...)
	// Append a zero-length sentinel (totalLen=0)
	data = append(data, 0, 0, 0, 0)
	data = append(data, makeTLVField(0x0002, []byte("hidden"))...)

	fields := ParseTLVFields(data)
	if len(fields) != 1 {
		t.Fatalf("expected 1 field (sentinel should stop), got %d", len(fields))
	}
	if fields[0].Tag != 0x0001 {
		t.Errorf("field[0].Tag = 0x%04X, want 0x0001", fields[0].Tag)
	}
}

func TestExtractPlaceRefs(t *testing.T) {
	data := []byte("born in [[pt:42]] and died in [[pt:99]]")
	refs := ExtractPlaceRefs(data)
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0] != 42 {
		t.Errorf("refs[0] = %d, want 42", refs[0])
	}
	if refs[1] != 99 {
		t.Errorf("refs[1] = %d, want 99", refs[1])
	}

	// No matches
	none := ExtractPlaceRefs([]byte("no refs here"))
	if len(none) != 0 {
		t.Errorf("expected 0 refs, got %d", len(none))
	}
}

func TestExtractString(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"printable ascii", []byte{0x00, 0x01, 'H', 'e', 'l', 'l', 'o', 0x00}, "Hello"},
		{"all printable", []byte("Test 123"), "Test 123"},
		{"leading non-printable", []byte{0x00, 0x01, 0x02, 'A', 'B'}, "AB"},
		{"utf8 multi-byte", append([]byte{0x00}, []byte("café")...), "café"},
		{"empty", []byte{0x00, 0x01, 0x02}, ""},
		{"nil", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractString(tt.data)
			if got != tt.want {
				t.Errorf("ExtractString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractDate(t *testing.T) {
	// Helper to build a 26-byte event sub-data with date encoding.
	// Month is split: group = month/4 stored in totalQ low bits,
	// offset = month%4 stored in dayByte bits 7-6.
	buildDateData := func(precFlags byte, day, month, year int) []byte {
		buf := make([]byte, 26)
		// Sub-TLV at offset 18: length 8 means date sub-TLV
		putU16LE(buf, 18, 8)
		// Precision flags at offset 22
		buf[22] = precFlags
		// Day byte at offset 23: bits 7-6 = month offset, bits 5-0 = day
		group := month / 4
		offset := month % 4
		buf[23] = byte(offset<<6) | byte(day&0x3F)
		// Year+group encoding at offset 24 (u16 LE)
		totalQ := uint16((year+8000)*4 + group)
		putU16LE(buf, 24, totalQ)
		return buf
	}

	tests := []struct {
		name string
		data []byte
		want string
	}{
		{
			name: "normal date: 15 Mar 1990",
			data: buildDateData(0x00, 15, 3, 1990),
			want: "15 Mar 1990",
		},
		{
			name: "month only: Jun 2000",
			data: buildDateData(0x00, 0, 6, 2000),
			want: "Jun 2000",
		},
		{
			name: "about year only: about 1850",
			data: buildDateData(0xA0, 0, 1, 1850),
			want: "about 1850",
		},
		{
			name: "after date: after 1 Jan 1900",
			data: buildDateData(0x40, 1, 1, 1900),
			want: "after 1 Jan 1900",
		},
		{
			name: "Nov date: 22 Nov 1963",
			data: buildDateData(0x00, 22, 11, 1963),
			want: "22 Nov 1963",
		},
		{
			name: "Dec date: 25 Dec 1800",
			data: buildDateData(0x00, 25, 12, 1800),
			want: "25 Dec 1800",
		},
		{
			name: "too short",
			data: make([]byte, 10),
			want: "",
		},
		{
			name: "sub-TLV length not 8",
			data: func() []byte {
				buf := make([]byte, 26)
				putU16LE(buf, 18, 6) // not 8
				return buf
			}(),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractDate(tt.data)
			if got != tt.want {
				t.Errorf("ExtractDate() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractNoteRef(t *testing.T) {
	// Build event sub-data with a note reference sub-TLV at offset 18
	// Sub-TLV: [totalLen=8 u16LE][tag=0x0000 u16LE][noteID u32LE]
	buf := make([]byte, 26)
	putU16LE(buf, 18, 8)      // totalLen = 8
	putU16LE(buf, 20, 0x0000) // tag = 0x0000
	putU32LE(buf, 22, 42)     // note ID = 42

	got := ExtractNoteRef(buf)
	if got != 42 {
		t.Errorf("ExtractNoteRef() = %d, want 42", got)
	}

	// No note ref: wrong tag
	buf2 := make([]byte, 26)
	putU16LE(buf2, 18, 8)
	putU16LE(buf2, 20, 0x0001) // not 0x0000
	putU32LE(buf2, 22, 42)
	got2 := ExtractNoteRef(buf2)
	if got2 != 0 {
		t.Errorf("ExtractNoteRef(wrong tag) = %d, want 0", got2)
	}

	// Too short
	got3 := ExtractNoteRef(make([]byte, 10))
	if got3 != 0 {
		t.Errorf("ExtractNoteRef(short) = %d, want 0", got3)
	}
}

func TestExtractSourceCitations(t *testing.T) {
	// Build citation data:
	//   [innerLength u32LE][count u32LE]
	//   per entry: [entryLength u16LE][unknown u16LE][sourceID u32LE][detail...]

	// Entry 1: sourceID=6, detail="page 274"
	detail1 := []byte("page 274")
	entry1Len := uint16(8 + len(detail1))
	entry1 := make([]byte, entry1Len)
	putU16LE(entry1, 0, entry1Len)
	putU16LE(entry1, 2, 0x0000) // unknown
	putU32LE(entry1, 4, 6)
	copy(entry1[8:], detail1)

	// Entry 2: sourceID=1, detail="" (no detail)
	entry2 := make([]byte, 8)
	putU16LE(entry2, 0, 8)
	putU16LE(entry2, 2, 0x0000)
	putU32LE(entry2, 4, 1)

	innerLen := uint32(8 + len(entry1) + len(entry2))
	data := make([]byte, 8)
	putU32LE(data, 0, innerLen)
	putU32LE(data, 4, 2) // count=2
	data = append(data, entry1...)
	data = append(data, entry2...)

	cites := ExtractSourceCitations(data)
	if len(cites) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(cites))
	}
	if cites[0].SourceID != 6 {
		t.Errorf("cites[0].SourceID = %d, want 6", cites[0].SourceID)
	}
	if cites[0].Detail != "page 274" {
		t.Errorf("cites[0].Detail = %q, want %q", cites[0].Detail, "page 274")
	}
	if cites[1].SourceID != 1 {
		t.Errorf("cites[1].SourceID = %d, want 1", cites[1].SourceID)
	}
	if cites[1].Detail != "" {
		t.Errorf("cites[1].Detail = %q, want empty", cites[1].Detail)
	}

	// Too short
	if got := ExtractSourceCitations(nil); got != nil {
		t.Errorf("expected nil for nil data, got %v", got)
	}
	if got := ExtractSourceCitations(make([]byte, 4)); got != nil {
		t.Errorf("expected nil for short data, got %v", got)
	}

	// Zero count
	zeroData := make([]byte, 8)
	putU32LE(zeroData, 0, 8)
	putU32LE(zeroData, 4, 0)
	if got := ExtractSourceCitations(zeroData); got != nil {
		t.Errorf("expected nil for zero count, got %v", got)
	}
}

func TestParseTLVFields_DropsOverrunningField(t *testing.T) {
	data := []byte{
		0, 0, 0, 0, 0, 0, // preamble
		0x06, 0x00, 0x1b, 0x00, 0x01, 0x00, // valid field: len 6, tag 0x1B
		0x00, 0xfc, 0x09, 0x08, 0x3c, 0xa6, // junk claiming len 0xFC00
	}
	fields := ParseTLVFields(data)
	if len(fields) != 1 || fields[0].Tag != 0x1b {
		t.Errorf("ParseTLVFields() = %+v, want only tag 0x1B", fields)
	}
}
