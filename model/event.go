package model

// EventDefinition describes a schema/event type parsed from 0x20CC records.
type EventDefinition struct {
	ID            uint32 `json:"id"`
	SeqNum        uint16 `json:"seq_num"`
	DisplayName   string `json:"display_name,omitempty"`
	GEDCOMCode    string `json:"gedcom_code,omitempty"`
	ShortLabel    string `json:"short_label,omitempty"`
	Abbreviation  string `json:"abbreviation,omitempty"`
	Abbreviation2 string `json:"abbreviation2,omitempty"`
	Abbreviation3 string `json:"abbreviation3,omitempty"`
	SentenceForm  string `json:"sentence_form,omitempty"`
	Preposition   string `json:"preposition,omitempty"`
	Color         string `json:"color,omitempty"` // #RRGGBBAA, for child-status definitions
	RawFields     []RawField `json:"raw_fields,omitempty"`
}
