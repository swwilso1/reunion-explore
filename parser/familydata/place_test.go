package familydata

import "testing"

func TestParsePlace_NameNotJoinedToNextField(t *testing.T) {
	// Reunion 10 place record: timestamp + size, then the name field
	// (tag 0x1E) followed by a field whose length byte is 0x30 ('0').
	data := []byte{
		0xf5, 0xc6, 0x00, 0x52, 0x44, 0x00, // timestamp + repeated size
		0x12, 0x00, 0x1e, 0x00, // name field: len 18, tag 0x1E
		'L', 'o', 'n', 'e', ' ', 'T', 'r', 'e', 'e', ' ', '-', ' ', 'C', 'O',
		0x30, 0x00, 0x3c, 0x00, 0x2c, 0x00, // next field: len 0x30, tag 0x3C
	}
	p, err := ParsePlace(RawRecord{ID: 8, Data: data}, nil)
	if err != nil {
		t.Fatalf("ParsePlace() error: %v", err)
	}
	if p.Name != "Lone Tree - CO" {
		t.Errorf("Name = %q, want %q", p.Name, "Lone Tree - CO")
	}
}
