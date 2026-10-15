package familydata

import "testing"

func TestExtractSourceFieldValue_ShortValue(t *testing.T) {
	// Reunion 10 "Quality of Data" field holding the single character "0".
	data := []byte{
		0x17, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00,
		0x80, 0x00, // schema ID 128
		0x05, 0x00, 0x00, 0x00, '0', // value sub-TLV
	}
	if got := extractSourceFieldValue(data); got != "0" {
		t.Errorf("extractSourceFieldValue() = %q, want %q", got, "0")
	}
	if got := ParseEventField(data); got != 128 {
		t.Errorf("ParseEventField() = %d, want 128", got)
	}
}
