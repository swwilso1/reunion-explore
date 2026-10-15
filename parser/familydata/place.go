package familydata

import (
	reunion "github.com/kedoco/reunion-explore"
	"github.com/kedoco/reunion-explore/model"
)

// TagPlaceName is the TLV tag holding a place record's name.
const TagPlaceName uint16 = 0x001E

// ParsePlace parses a 0x20D8 place record from familydata.
func ParsePlace(rec RawRecord, ec *reunion.ErrorCollector) (*model.Place, error) {
	p := &model.Place{
		ID: rec.ID,
	}

	// Read the name from its TLV field so the following field's length
	// bytes are never mistaken for part of the name.
	for _, f := range ParseTLVFields(rec.Data) {
		if f.Tag == TagPlaceName {
			p.Name = cleanString(f.Data)
			break
		}
	}

	if p.Name == "" && len(rec.Data) > 8 {
		// Fallback: scan for the first printable string
		p.Name = ExtractString(rec.Data[8:])
	}

	return p, nil
}
