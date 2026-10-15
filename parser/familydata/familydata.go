// Package familydata parses the familyfile.familydata binary file.
package familydata

import (
	"fmt"
	"os"

	reunion "github.com/kedoco/reunion-explore"
	"github.com/kedoco/reunion-explore/model"
)

// placeholderID marks template records that are not real entries.
const placeholderID = 0xFFFFFFFF

// Result holds all data extracted from the familydata file.
type Result struct {
	Header           *model.Header
	Persons          []model.Person
	Families         []model.Family
	Places           []model.Place
	EventDefinitions []model.EventDefinition
	Sources          []model.Source
	Notes            []model.Note
	MediaRefs        []model.MediaRef
}

// Parse reads and parses the familydata binary file.
func Parse(path string, ec *reunion.ErrorCollector) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading familydata: %w", err)
	}

	if len(data) < 16 {
		return nil, fmt.Errorf("familydata too short: %d bytes", len(data))
	}

	result := &Result{}

	// Parse header
	header, err := ParseHeader(data)
	if err != nil {
		ec.Add("familydata", 0, "header parse error", err)
	} else {
		result.Header = header
	}

	// Scan all records
	records := ScanRecords(data)

	// Process each record type
	for _, rec := range records {
		// Reunion 13 stores a placeholder record with ID 0xFFFFFFFF for
		// each record type; it holds no user data.
		if rec.ID == placeholderID {
			continue
		}
		switch rec.Type {
		case RecordTypePerson:
			person, err := ParsePerson(rec, ec)
			if err != nil {
				ec.Add("familydata", rec.Offset, "person parse error", err)
				continue
			}
			result.Persons = append(result.Persons, *person)

		case RecordTypeFamily:
			family, err := ParseFamily(rec, ec)
			if err != nil {
				ec.Add("familydata", rec.Offset, "family parse error", err)
				continue
			}
			result.Families = append(result.Families, *family)

		case RecordTypeSchema:
			def, err := ParseSchema(rec, ec)
			if err != nil {
				ec.Add("familydata", rec.Offset, "schema parse error", err)
				continue
			}
			result.EventDefinitions = append(result.EventDefinitions, *def)

		case RecordTypePlace:
			place, err := ParsePlace(rec, ec)
			if err != nil {
				ec.Add("familydata", rec.Offset, "place parse error", err)
				continue
			}
			result.Places = append(result.Places, *place)

		case RecordTypeNote:
			note, err := ParseNote(rec, ec)
			if err != nil {
				ec.Add("familydata", rec.Offset, "note parse error", err)
				continue
			}
			result.Notes = append(result.Notes, *note)

		case RecordTypeSource:
			source, err := ParseSource(rec, ec)
			if err != nil {
				ec.Add("familydata", rec.Offset, "source parse error", err)
				continue
			}
			result.Sources = append(result.Sources, *source)

		case RecordTypeMedia:
			media, err := ParseMedia(rec, ec)
			if err != nil {
				ec.Add("familydata", rec.Offset, "media parse error", err)
				continue
			}
			result.MediaRefs = append(result.MediaRefs, *media)
		}
	}

	labelSourceFields(result)

	return result, nil
}

// labelSourceFields names each source field after its schema definition.
func labelSourceFields(r *Result) {
	names := make(map[uint32]string, len(r.EventDefinitions))
	for _, d := range r.EventDefinitions {
		names[d.ID] = d.DisplayName
	}
	for i := range r.Sources {
		for j := range r.Sources[i].Fields {
			fld := &r.Sources[i].Fields[j]
			fld.Label = names[uint32(fld.SchemaID)]
		}
	}
}
