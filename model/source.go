package model

// Source represents a source record from the familydata.
type Source struct {
	ID        uint32        `json:"id"`
	SeqNum    uint16        `json:"seq_num"`
	Title     string        `json:"title,omitempty"`
	Fields    []SourceField `json:"fields,omitempty"`
	RawFields []RawField    `json:"raw_fields,omitempty"`
}

// SourceField is one field of a source (e.g. Title, Author, Date), as
// defined by the source's template.
type SourceField struct {
	SchemaID uint16 `json:"schema_id"`
	Label    string `json:"label,omitempty"` // field name from the schema definition
	Value    string `json:"value"`
}
