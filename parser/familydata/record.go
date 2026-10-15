package familydata

import (
	"github.com/kedoco/reunion-explore/internal/binutil"
)

// RecordType identifies the type of a familydata record.
type RecordType uint16

const (
	RecordTypePerson RecordType = 0x20C4
	RecordTypeFamily RecordType = 0x20C8
	RecordTypeSchema RecordType = 0x20CC
	RecordTypeSource RecordType = 0x20D0
	RecordTypeMedia  RecordType = 0x20D4
	RecordTypePlace  RecordType = 0x20D8
	RecordTypeNote   RecordType = 0x2104
	RecordTypeDoc    RecordType = 0x2108
	RecordTypeReport RecordType = 0x210C
)

// recordTypeFlag is set on most record type codes in Reunion 10 files
// (e.g. 0xA0C4 for a person); the remaining bits match later versions.
const recordTypeFlag = 0x8000

// Marker is the 4-byte record marker found in familydata.
var Marker = []byte{0x05, 0x03, 0x02, 0x01}

// RawRecord represents a single record found in the familydata file.
type RawRecord struct {
	Offset  int        // byte offset of the record start (8 bytes before marker)
	Type    RecordType // 2-byte type code
	SeqNum  uint16     // 2-byte sequence number
	DataLen uint32     // data length from the 4 bytes after marker
	ID      uint32     // record ID from bytes 16-19
	Data    []byte     // full record data starting from offset
}

// ScanRecords scans the familydata for all records marked by the 05030201 pattern.
func ScanRecords(data []byte) []RawRecord {
	var records []RawRecord
	pos := 0

	for pos < len(data)-4 {
		// Search for the marker
		found := -1
		for i := pos; i <= len(data)-4; i++ {
			if data[i] == 0x05 && data[i+1] == 0x03 && data[i+2] == 0x02 && data[i+3] == 0x01 {
				found = i
				break
			}
		}
		if found == -1 {
			break
		}

		// Record structure:
		// offset-8: 4 bytes padding/zeros
		// offset-4: 2 bytes sequence number (LE)
		// offset-2: 2 bytes type code (LE)
		// offset+0: 4 bytes marker (05030201)
		// offset+4: 4 bytes data size (LE)
		// offset+8: 4 bytes record ID (LE)
		// offset+12: 4 bytes timestamp

		markerPos := found
		recStart := markerPos - 8
		if recStart < 0 {
			recStart = 0
		}

		rec := RawRecord{
			Offset: recStart,
		}

		// Read type code (2 bytes before marker)
		if markerPos >= 2 {
			tc, _ := binutil.U16LE(data, markerPos-2)
			rec.Type = RecordType(tc &^ recordTypeFlag)
		}

		// Read sequence number (4 bytes before marker)
		if markerPos >= 4 {
			seq, _ := binutil.U16LE(data, markerPos-4)
			rec.SeqNum = seq
		}

		// Read data length
		if markerPos+8 <= len(data) {
			dl, _ := binutil.U32LE(data, markerPos+4)
			rec.DataLen = dl
		}

		// Read record ID
		if markerPos+12 <= len(data) {
			id, _ := binutil.U32LE(data, markerPos+8)
			rec.ID = id
		}

		// Extract the full record data
		dataStart := markerPos + 12
		dataEnd := dataStart + int(rec.DataLen)
		if dataEnd > len(data) {
			dataEnd = len(data)
		}
		if dataStart < len(data) {
			rec.Data = data[dataStart:dataEnd]
		}

		records = append(records, rec)
		pos = markerPos + 4
	}

	// Extend each record's data to include any bytes between the declared
	// DataLen boundary and the next record's marker. Some records (notably
	// families) have child reference data that overflows past the declared
	// DataLen. The overflow bytes may sit in the inter-record gap or in the
	// 4-byte "padding" area at the start of the next record's header.
	// The TLV parser safely stops on zero padding (totalLen=0 < 4 → break)
	// or small header values (totalLen < 4 → break).
	for i := range records {
		dataStart := records[i].Offset + 20 // markerPos+12 = Offset+8+12 = Offset+20
		// Use the next record's marker position (Offset+8) as the upper bound,
		// since the 4 bytes between Offset and Offset+4 can contain overflow
		// data rather than true padding.
		var boundary int
		if i+1 < len(records) {
			boundary = records[i+1].Offset + 8 // next marker position
		} else {
			boundary = len(data)
		}
		if boundary > len(data) {
			boundary = len(data)
		}
		declaredEnd := dataStart + int(records[i].DataLen)
		if boundary > declaredEnd && dataStart < len(data) {
			records[i].Data = data[dataStart:boundary]
		}
	}

	return records
}
