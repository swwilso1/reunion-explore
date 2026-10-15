package reunion_test

import (
	"encoding/json"
	"strings"
	"testing"

	reunion "github.com/kedoco/reunion-explore"
	"github.com/kedoco/reunion-explore/model"
	_ "github.com/kedoco/reunion-explore/parser" // register v13/v14 parsers
)

func TestOpen(t *testing.T) {
	ff, err := reunion.Open("testdata/Sample Family 14.familyfile14", nil)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	// Signature
	if ff.Signature != "1579320" {
		t.Errorf("Signature = %q, want %q", ff.Signature, "1579320")
	}

	// Version
	if ff.Version != 14 {
		t.Errorf("Version = %d, want 14", ff.Version)
	}

	// Header
	if ff.Header == nil {
		t.Fatal("Header is nil")
	}
	if ff.Header.Magic != "3SDUAU~R" {
		t.Errorf("Header.Magic = %q, want %q", ff.Header.Magic, "3SDUAU~R")
	}

	// Record counts
	counts := []struct {
		name string
		got  int
		want int
	}{
		{"Persons", len(ff.Persons), 49},
		{"Families", len(ff.Families), 35},
		{"Places", len(ff.Places), 52},
		{"EventDefinitions", len(ff.EventDefinitions), 147},
		{"Sources", len(ff.Sources), 25},
		{"Notes", len(ff.Notes), 28},
		{"MediaRefs", len(ff.MediaRefs), 36},
		{"PlaceUsages", len(ff.PlaceUsages), 52},
	}
	for _, c := range counts {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}

	// All persons should have names
	for i, p := range ff.Persons {
		if p.GivenName == "" && p.Surname == "" {
			t.Errorf("Person[%d] (ID %d) has no name", i, p.ID)
		}
	}

	// All families should have at least one partner
	for i, f := range ff.Families {
		if f.Partner1 == 0 && f.Partner2 == 0 {
			t.Errorf("Family[%d] (ID %d) has no partners", i, f.ID)
		}
	}

	// First names cache
	if got := firstNames(ff); strings.Join(got, ",") != "Bouvier,Patrick" {
		t.Errorf("FirstNames = %q, want [Bouvier Patrick]", got)
	}

	// Spot-check: ID 4 = John Fitzgerald Kennedy, Male
	var jfk *model.Person
	for i := range ff.Persons {
		if ff.Persons[i].ID == 4 {
			jfk = &ff.Persons[i]
			break
		}
	}
	if jfk == nil {
		t.Fatal("Person with ID 4 not found")
	}
	if jfk.GivenName != "John Fitzgerald" {
		t.Errorf("JFK GivenName = %q, want %q", jfk.GivenName, "John Fitzgerald")
	}
	if jfk.Surname != "KENNEDY" {
		t.Errorf("JFK Surname = %q, want %q", jfk.Surname, "KENNEDY")
	}
	if jfk.Sex != model.SexMale {
		t.Errorf("JFK Sex = %d, want SexMale (%d)", jfk.Sex, model.SexMale)
	}

	// JFK person-level source citations (name citations from tag 0x0020)
	if len(jfk.SourceCitations) != 2 {
		t.Errorf("JFK SourceCitations count = %d, want 2", len(jfk.SourceCitations))
	} else {
		if jfk.SourceCitations[0].SourceID != 6 {
			t.Errorf("JFK SourceCitations[0].SourceID = %d, want 6", jfk.SourceCitations[0].SourceID)
		}
		if jfk.SourceCitations[1].SourceID != 1 {
			t.Errorf("JFK SourceCitations[1].SourceID = %d, want 1", jfk.SourceCitations[1].SourceID)
		}
	}

	// JFK birth event (tag 0x03E8) source citation
	var birthEvt *model.PersonEvent
	for i := range jfk.Events {
		if jfk.Events[i].Tag == 0x03E8 {
			birthEvt = &jfk.Events[i]
			break
		}
	}
	if birthEvt == nil {
		t.Error("JFK birth event (tag 0x03E8) not found")
	} else if len(birthEvt.SourceCitations) < 1 {
		t.Errorf("JFK birth event has %d source citations, want >= 1", len(birthEvt.SourceCitations))
	} else if birthEvt.SourceCitations[0].SourceID != 6 {
		t.Errorf("JFK birth citation sourceID = %d, want 6", birthEvt.SourceCitations[0].SourceID)
	}

	// Source titles should be clean (using tag 0x0014)
	for _, src := range ff.Sources {
		for _, r := range src.Title {
			if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
				t.Errorf("Source %d title contains control char U+%04X: %q", src.ID, r, src.Title)
				break
			}
		}
	}

	// JSON round-trip
	jsonData, err := ff.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error: %v", err)
	}
	if len(jsonData) == 0 {
		t.Fatal("ToJSON() produced empty output")
	}

	var raw json.RawMessage
	if err := json.Unmarshal(jsonData, &raw); err != nil {
		t.Fatalf("JSON output is not valid: %v", err)
	}

	compactData, err := ff.ToJSONCompact()
	if err != nil {
		t.Fatalf("ToJSONCompact() error: %v", err)
	}
	if len(compactData) >= len(jsonData) {
		t.Errorf("Compact JSON (%d bytes) should be smaller than indented (%d bytes)", len(compactData), len(jsonData))
	}
}

func TestOpenV13(t *testing.T) {
	ff, err := reunion.Open("testdata/Sample Family 13.familyfile13", nil)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	if ff.Version != 13 {
		t.Errorf("Version = %d, want 13", ff.Version)
	}
	if ff.Signature != "250208" {
		t.Errorf("Signature = %q, want %q", ff.Signature, "250208")
	}
	if len(ff.Warnings) != 0 {
		t.Errorf("Warnings = %v, want none", ff.Warnings)
	}

	// Same sample data as the v14 file; placeholder records must be excluded.
	counts := []struct {
		name string
		got  int
		want int
	}{
		{"Persons", len(ff.Persons), 49},
		{"Families", len(ff.Families), 35},
		{"Places", len(ff.Places), 52},
		{"EventDefinitions", len(ff.EventDefinitions), 147},
		{"Sources", len(ff.Sources), 25},
		{"Notes", len(ff.Notes), 28},
		{"MediaRefs", len(ff.MediaRefs), 35},
	}
	for _, c := range counts {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}

	for i, p := range ff.Persons {
		if p.GivenName == "" && p.Surname == "" {
			t.Errorf("Person[%d] (ID %d) has no name", i, p.ID)
		}
	}

	var jfk *model.Person
	for i := range ff.Persons {
		if ff.Persons[i].ID == 4 {
			jfk = &ff.Persons[i]
			break
		}
	}
	if jfk == nil {
		t.Fatal("Person with ID 4 not found")
	}
	if jfk.GivenName != "John Fitzgerald" || jfk.Surname != "KENNEDY" {
		t.Errorf("JFK name = %q %q, want %q %q", jfk.GivenName, jfk.Surname, "John Fitzgerald", "KENNEDY")
	}

	names := firstNames(ff)
	if len(names) != 59 {
		t.Errorf("FirstNames count = %d, want 59", len(names))
	}
	if len(names) > 0 && names[0] != "Alois" {
		t.Errorf("FirstNames[0] = %q, want %q", names[0], "Alois")
	}
	for _, n := range names {
		if strings.IndexFunc(n, func(r rune) bool { return r < 0x20 }) >= 0 {
			t.Errorf("first name contains control char: %q", n)
		}
	}

	// Plain-text notes (no markup) must not pick up the "talf" marker.
	const want = "After her husband died, she opened a “Notions Shop” to provide for her family."
	var found bool
	for _, n := range ff.Notes {
		if n.RawText == want {
			found = true
		}
		if strings.Contains(n.RawText, "talf") {
			t.Errorf("Note %d text contains marker: %q", n.ID, n.RawText)
		}
	}
	if !found {
		t.Errorf("plain-text note %q not found", want)
	}
}

func firstNames(ff *model.FamilyFile) []string {
	names := make([]string, len(ff.FirstNames))
	for i, fn := range ff.FirstNames {
		names[i] = fn.Name
	}
	return names
}

func TestOpen_NotABundle(t *testing.T) {
	_, err := reunion.Open("/tmp/not-a-bundle.txt", nil)
	if err == nil {
		t.Error("Open() should error for non-bundle path")
	}
}

func TestOpen_MissingBundle(t *testing.T) {
	_, err := reunion.Open("/tmp/nonexistent.familyfile14", nil)
	if err == nil {
		t.Error("Open() should error for missing bundle")
	}
}
