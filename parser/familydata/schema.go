package familydata

import (
	"encoding/binary"
	"fmt"

	"github.com/kedoco/reunion-explore/model"

	reunion "github.com/kedoco/reunion-explore"
)

// Tag constants for schema/event definition record fields.
const (
	TagDisplayName   uint16 = 0x0014
	TagGEDCOMCode    uint16 = 0x0019
	TagShortLabel    uint16 = 0x0028
	TagAbbreviation  uint16 = 0x0032
	TagAbbreviation2 uint16 = 0x0037
	TagAbbreviation3 uint16 = 0x003C
	TagSentenceForm  uint16 = 0x006E
	TagPreposition   uint16 = 0x0078
	TagSchemaKind    uint16 = 0x000A
)

// schemaKindChildStatus marks child-status definitions (Twin, Adopted, ...),
// whose 0x0032 field is an RGBA color rather than an abbreviation.
const schemaKindChildStatus = 14

// ParseSchema parses a 0x20CC schema record into an EventDefinition.
func ParseSchema(rec RawRecord, ec *reunion.ErrorCollector) (*model.EventDefinition, error) {
	def := &model.EventDefinition{
		ID:     rec.ID,
		SeqNum: rec.SeqNum,
	}

	if len(rec.Data) < 6 {
		return def, nil
	}

	fields := ParseTLVFields(rec.Data)

	var kind uint16
	for _, f := range fields {
		if f.Tag == TagSchemaKind && len(f.Data) >= 2 {
			kind = binary.LittleEndian.Uint16(f.Data)
		}
	}

	for _, f := range fields {
		str := cleanString(f.Data)

		if f.Tag == TagAbbreviation && kind == schemaKindChildStatus && len(f.Data) == 4 {
			def.Color = fmt.Sprintf("#%02X%02X%02X%02X", f.Data[0], f.Data[1], f.Data[2], f.Data[3])
			continue
		}

		switch f.Tag {
		case TagDisplayName:
			def.DisplayName = str
		case TagGEDCOMCode:
			def.GEDCOMCode = str
		case TagShortLabel:
			def.ShortLabel = str
		case TagAbbreviation:
			def.Abbreviation = str
		case TagAbbreviation2:
			def.Abbreviation2 = str
		case TagAbbreviation3:
			def.Abbreviation3 = str
		case TagSentenceForm:
			def.SentenceForm = str
		case TagPreposition:
			def.Preposition = str
		default:
			def.RawFields = append(def.RawFields, model.RawField{
				Tag:  f.Tag,
				Data: f.Data,
				Size: uint16(len(f.Data) + 4),
			})
		}
	}

	return def, nil
}
