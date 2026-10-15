package familydata

import (
	"github.com/kedoco/reunion-explore/internal/binutil"
	"github.com/kedoco/reunion-explore/model"

	reunion "github.com/kedoco/reunion-explore"
)

// Tag constants for family record fields.
const (
	TagPartner1 uint16 = 0x0050
	TagPartner2 uint16 = 0x0051
	TagMarriage uint16 = 0x005F
)

// isChildTag returns true if the tag is a child reference. Children are
// numbered from 0xFA in birth order and continue past 0xFF (0x100, 0x101,
// ...) in families with more than six children, up to the note tags.
func isChildTag(tag uint16) bool {
	return tag >= 0x00FA && tag < 0x0190
}

// marriageSubFieldsOffset is where sub-TLVs start in a marriage field. Its
// header is u32 + u32 + u32 schema ID, shorter than a regular event's.
const marriageSubFieldsOffset = 12

// isFamilyEventTag returns true if the tag is a family event (>= 0x100).
func isFamilyEventTag(tag uint16) bool {
	return tag >= 0x0100
}

// ParseFamily parses a 0x20C8 family record into a Family model.
func ParseFamily(rec RawRecord, ec *reunion.ErrorCollector) (*model.Family, error) {
	f := &model.Family{
		ID:     rec.ID,
		SeqNum: rec.SeqNum,
	}

	if len(rec.Data) < 6 {
		return f, nil
	}

	fields := ParseTLVFields(rec.Data)

	for _, field := range fields {
		switch {
		case field.Tag == TagPartner1:
			if len(field.Data) >= 4 {
				id, _ := binutil.U32LE(field.Data, 0)
				f.Partner1 = id
			} else if len(field.Data) >= 2 {
				id, _ := binutil.U16LE(field.Data, 0)
				f.Partner1 = uint32(id)
			}
		case field.Tag == TagPartner2:
			if len(field.Data) >= 4 {
				id, _ := binutil.U32LE(field.Data, 0)
				f.Partner2 = id
			} else if len(field.Data) >= 2 {
				id, _ := binutil.U16LE(field.Data, 0)
				f.Partner2 = uint32(id)
			}
		case field.Tag == TagMarriage:
			if len(field.Data) >= marriageSubFieldsOffset {
				schemaID, _ := binutil.U32LE(field.Data, 8)
				f.Events = append(f.Events, model.FamilyEvent{
					Tag:             field.Tag,
					SchemaID:        uint16(schemaID),
					PlaceRefs:       ExtractPlaceRefs(field.Data),
					RawData:         field.Data,
					Date:            extractDateAt(field.Data, marriageSubFieldsOffset),
					Text:            extractEventTextAt(field.Data, marriageSubFieldsOffset),
					SourceCitations: extractEventSourceCitationsAt(field.Data, marriageSubFieldsOffset),
				})
			}
		case isPictureTag(field.Tag):
			if pic, ok := ParsePicture(field.Tag, field.Data); ok {
				f.Pictures = append(f.Pictures, pic)
			}
		case isChildTag(field.Tag):
			if len(field.Data) >= 4 {
				raw, _ := binutil.U32LE(field.Data, 0)
				// Child IDs are encoded as u32_LE >> 8
				id := raw >> 8
				if id > 0 {
					f.Children = append(f.Children, id)
				}
			}
		case isFamilyEventTag(field.Tag):
			evt := model.FamilyEvent{
				Tag:             field.Tag,
				PlaceRefs:       ExtractPlaceRefs(field.Data),
				RawData:         field.Data,
				SchemaID:        ParseEventField(field.Data),
				Date:            ExtractDate(field.Data),
				Text:            ExtractEventText(field.Data),
				SourceCitations: ExtractEventSourceCitations(field.Data),
			}
			f.Events = append(f.Events, evt)
		default:
			f.RawFields = append(f.RawFields, model.RawField{
				Tag:  field.Tag,
				Data: field.Data,
				Size: uint16(len(field.Data) + 4),
			})
		}
	}

	return f, nil
}
