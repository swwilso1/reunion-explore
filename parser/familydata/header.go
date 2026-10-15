package familydata

import (
	"bytes"
	"encoding/binary"

	"github.com/kedoco/reunion-explore/internal/binutil"
	"github.com/kedoco/reunion-explore/model"
)

// recordTypeDevice is the header record holding the device strings.
const recordTypeDevice RecordType = 0x2010

// ParseHeader extracts the familydata file header.
// The header starts with the 8-byte magic "3SDUAU~R", followed by metadata,
// then newline-separated strings: device ID, model name, serial, app path.
func ParseHeader(data []byte) (*model.Header, error) {
	if len(data) < 8 {
		return nil, binutil.ErrShortRead
	}

	h := &model.Header{
		Magic: string(data[0:8]),
	}

	// The device strings live in the first record, which is a 0x2010
	// record in Reunion 13/14. Reunion 10 starts directly with other
	// records, so there are none to read.
	if idx := bytes.Index(data, Marker); idx >= 2 {
		if RecordType(binary.LittleEndian.Uint16(data[idx-2:])&^recordTypeFlag) != recordTypeDevice {
			return h, nil
		}
	}

	// After the fixed header area, look for newline-terminated strings
	// starting around offset 0x58 (the CXXQGL2G... device ID area)
	// Scan from byte 80 onwards for \n-terminated strings
	pos := 80
	if pos >= len(data) {
		return h, nil
	}

	// Find the start of the device ID string (first printable run after pos 0x50)
	for pos < len(data) && pos < 256 {
		if data[pos] >= 0x20 && data[pos] < 0x7F {
			break
		}
		pos++
	}

	// Read device ID
	devID, n := binutil.ReadNewlineTermString(data, pos)
	if n > 0 {
		h.DeviceID = devID
		pos += n
	}

	// Read model name
	modelName, n := binutil.ReadNewlineTermString(data, pos)
	if n > 0 {
		h.Model = modelName
		pos += n
	}

	// Read serial number
	serial, n := binutil.ReadNewlineTermString(data, pos)
	if n > 0 {
		h.Serial = serial
		pos += n
	}

	// Read app path
	appPath, n := binutil.ReadNullTermString(data, pos)
	if n > 0 {
		h.AppPath = appPath
	}

	return h, nil
}
