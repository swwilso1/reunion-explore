package familydata

import (
	"github.com/kedoco/reunion-explore/model"

	reunion "github.com/kedoco/reunion-explore"
)

// Tag constants for person record fields.
const (
	TagGivenName        uint16 = 0x001E
	TagSurname1         uint16 = 0x000C
	TagSurname2         uint16 = 0x0023
	TagSexFlags         uint16 = 0x001B
	TagNameSourceCiting uint16 = 0x0020
	TagPrefixTitle      uint16 = 0x0028
	TagSuffixTitle      uint16 = 0x002D
	TagUserID           uint16 = 0x0037
)

// ParsePerson parses a 0x20C4 person record into a Person model.
func ParsePerson(rec RawRecord, ec *reunion.ErrorCollector) (*model.Person, error) {
	p := &model.Person{
		ID:     rec.ID,
		SeqNum: rec.SeqNum,
	}

	if len(rec.Data) < 6 {
		return p, nil
	}

	fields := ParseTLVFields(rec.Data)

	for _, f := range fields {
		switch f.Tag {
		case TagGivenName:
			p.GivenName = cleanString(f.Data)
		case TagSurname1, TagSurname2:
			if p.Surname == "" {
				p.Surname = cleanString(f.Data)
			}
		case TagSexFlags:
			if len(f.Data) >= 1 {
				switch f.Data[0] {
				case 1:
					p.Sex = model.SexMale
				case 2:
					p.Sex = model.SexFemale
				}
			}
		case TagPrefixTitle:
			p.PrefixTitle = cleanString(f.Data)
		case TagSuffixTitle:
			p.SuffixTitle = cleanString(f.Data)
		case TagUserID:
			p.UserID = cleanString(f.Data)
		case TagNameSourceCiting:
			if cites := ExtractSourceCitations(f.Data); len(cites) > 0 {
				p.SourceCitations = append(p.SourceCitations, cites...)
			}
		default:
			if isPictureTag(f.Tag) {
				if pic, ok := ParsePicture(f.Tag, f.Data); ok {
					p.Pictures = append(p.Pictures, pic)
					continue
				}
			}
			if isEventTag(f.Tag) {
				evt := model.PersonEvent{
					Tag:             f.Tag,
					PlaceRefs:       ExtractPlaceRefs(f.Data),
					RawData:         f.Data,
					SchemaID:        ParseEventField(f.Data),
					Date:            ExtractDate(f.Data),
					Text:            ExtractEventText(f.Data),
					SourceCitations: ExtractEventSourceCitations(f.Data),
				}
				p.Events = append(p.Events, evt)

				// For note-type events (tag < 0x03E8), extract the note reference
				if f.Tag < 0x03E8 {
					noteID := ExtractNoteRef(f.Data)
					if noteID > 0 {
						p.NoteRefs = append(p.NoteRefs, model.NoteRef{
							NoteID:   noteID,
							EventTag: f.Tag,
							SchemaID: evt.SchemaID,
						})
					}
				}
			} else {
				p.RawFields = append(p.RawFields, model.RawField{
					Tag:  f.Tag,
					Data: f.Data,
					Size: uint16(len(f.Data) + 4),
				})
			}
		}
	}

	return p, nil
}

func cleanString(data []byte) string {
	// Find printable string content
	for i := len(data) - 1; i >= 0; i-- {
		if data[i] != 0 {
			return string(data[:i+1])
		}
	}
	return ""
}

func isEventTag(tag uint16) bool {
	// Event tags use codes >= 0x100 in the person record
	return tag >= 0x0100
}
