package model

// Sex represents a person's sex.
type Sex int

const (
	SexUnknown Sex = 0
	SexMale    Sex = 1
	SexFemale  Sex = 2
)

func (s Sex) String() string {
	switch s {
	case SexMale:
		return "M"
	case SexFemale:
		return "F"
	default:
		return "U"
	}
}

// Person represents an individual in the family file.
type Person struct {
	ID              uint32           `json:"id"`
	SeqNum          uint16           `json:"seq_num"`
	GivenName       string           `json:"given_name,omitempty"`
	Surname         string           `json:"surname,omitempty"`
	PrefixTitle     string           `json:"prefix_title,omitempty"`
	SuffixTitle     string           `json:"suffix_title,omitempty"`
	UserID          string           `json:"user_id,omitempty"`
	Sex             Sex              `json:"sex"`
	Events          []PersonEvent    `json:"events,omitempty"`
	Pictures        []Picture        `json:"pictures,omitempty"`
	NoteRefs        []NoteRef        `json:"note_refs,omitempty"`
	SourceCitations []SourceCitation `json:"source_citations,omitempty"`
	RawFields       []RawField       `json:"raw_fields,omitempty"`
}

// Picture links a person to an image file.
type Picture struct {
	Slot    int    `json:"slot"` // 1-based picture slot
	Caption string `json:"caption,omitempty"`
	Path    string `json:"path,omitempty"` // file name or path as stored by Reunion
}

// PersonEvent represents an event associated with a person (birth, death, etc.).
type PersonEvent struct {
	Tag             uint16           `json:"tag"`
	TypeCode        uint16           `json:"type_code,omitempty"`
	SchemaID        uint16           `json:"schema_id,omitempty"`
	PlaceRefs       []int            `json:"place_refs,omitempty"`
	Date            string           `json:"date,omitempty"`
	Text            string           `json:"text,omitempty"`
	SourceCitations []SourceCitation `json:"source_citations,omitempty"`
	RawData         []byte           `json:"-"`
}

// RawField holds a TLV field that wasn't specifically parsed.
type RawField struct {
	Tag  uint16 `json:"tag"`
	Data []byte `json:"-"`
	Size uint16 `json:"size"`
}
