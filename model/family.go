package model

// Family represents a family unit linking partners and children.
type Family struct {
	ID        uint32        `json:"id"`
	SeqNum    uint16        `json:"seq_num"`
	Partner1  uint32        `json:"partner1,omitempty"`
	Partner2  uint32        `json:"partner2,omitempty"`
	Children  []uint32      `json:"children,omitempty"`
	Events    []FamilyEvent `json:"events,omitempty"`
	Pictures  []Picture     `json:"pictures,omitempty"`
	RawFields []RawField    `json:"raw_fields,omitempty"`
}

// FamilyEvent represents an event associated with a family (marriage, etc.).
type FamilyEvent struct {
	Tag             uint16           `json:"tag"`
	TypeCode        uint16           `json:"type_code,omitempty"`
	SchemaID        uint16           `json:"schema_id,omitempty"`
	PlaceRefs       []int            `json:"place_refs,omitempty"`
	Date            string           `json:"date,omitempty"`
	Text            string           `json:"text,omitempty"`
	SourceCitations []SourceCitation `json:"source_citations,omitempty"`
	RawData         []byte           `json:"-"`
}
