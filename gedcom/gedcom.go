// Package gedcom writes a parsed Reunion family file as GEDCOM 5.5.1.
package gedcom

import (
	"bufio"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/kedoco/reunion-explore/model"
)

// Options controls GEDCOM output.
type Options struct {
	// ProgramVersion is written as the HEAD.SOUR.VERS value.
	ProgramVersion string
}

// maxLineValue is the longest value written on one line before the rest
// continues on CONC lines; GEDCOM lines are limited to 255 characters.
const maxLineValue = 200

// Tag ranges in Reunion records.
const (
	tagNoteFirst  = 0x0100 // note-type fields (Misc. Notes, Medical, ...)
	tagEventFirst = 0x03E8 // events (Birth, Death, ...)
	tagFactFirst  = 0x0BB8 // facts with a value (Occupation, Religion, ...)
)

// GEDCOM 5.5.1 tags valid on individuals and families.
var (
	indiEvents = tagSet("BIRT CHR DEAT BURI CREM ADOP BAPM BARM BASM BLES CHRA CONF FCOM ORDN NATU EMIG IMMI CENS PROB WILL GRAD RETI EVEN BAPL CONL ENDL SLGC")
	indiAttrs  = tagSet("CAST DSCR EDUC IDNO NATI NCHI NMR OCCU PROP RELI RESI SSN TITL FACT")
	famEvents  = tagSet("ANUL CENS DIV DIVF ENGA MARB MARC MARR MARL MARS RESI EVEN SLGS")
)

func tagSet(s string) map[string]bool {
	m := make(map[string]bool)
	for _, t := range strings.Fields(s) {
		m[t] = true
	}
	return m
}

// Write writes ff to w as a GEDCOM 5.5.1 file in UTF-8.
func Write(w io.Writer, ff *model.FamilyFile, opts Options) error {
	e := newExporter(w, ff, opts)
	e.header()
	for i := range e.persons {
		e.person(e.persons[i])
	}
	for _, f := range e.families {
		e.family(f)
	}
	for _, s := range e.sources {
		e.source(s)
	}
	for _, n := range e.notes {
		e.note(n)
	}
	e.line(0, "@U1@", "SUBM", "")
	e.line(1, "", "NAME", "reunion-explore")
	e.line(0, "", "TRLR", "")
	return e.out.Flush()
}

type exporter struct {
	out  *bufio.Writer
	ff   *model.FamilyFile
	opts Options

	persons  []*model.Person
	families []*model.Family
	sources  []*model.Source
	notes    []*model.Note

	personIDs map[uint32]bool
	sex       map[uint32]model.Sex
	sourceIDs map[uint32]bool
	noteIDs   map[uint32]bool
	defs      map[uint32]*model.EventDefinition
	places    map[int]string
	famc      map[uint32][]uint32 // person -> families as child
	fams      map[uint32][]uint32 // person -> families as partner
	fileNotes map[uint32][]*model.Note
}

func newExporter(w io.Writer, ff *model.FamilyFile, opts Options) *exporter {
	e := &exporter{
		out:       bufio.NewWriter(w),
		ff:        ff,
		opts:      opts,
		personIDs: make(map[uint32]bool),
		sex:       make(map[uint32]model.Sex),
		sourceIDs: make(map[uint32]bool),
		noteIDs:   make(map[uint32]bool),
		defs:      make(map[uint32]*model.EventDefinition),
		places:    make(map[int]string),
		famc:      make(map[uint32][]uint32),
		fams:      make(map[uint32][]uint32),
		fileNotes: make(map[uint32][]*model.Note),
	}
	if ff == nil {
		return e
	}
	for i := range ff.Persons {
		e.persons = append(e.persons, &ff.Persons[i])
		e.personIDs[ff.Persons[i].ID] = true
		e.sex[ff.Persons[i].ID] = ff.Persons[i].Sex
	}
	sort.Slice(e.persons, func(i, j int) bool { return e.persons[i].ID < e.persons[j].ID })

	for i := range ff.Families {
		f := &ff.Families[i]
		if !e.personIDs[f.Partner1] && !e.personIDs[f.Partner2] && len(f.Children) == 0 {
			continue // empty family
		}
		e.families = append(e.families, f)
	}
	sort.Slice(e.families, func(i, j int) bool { return e.families[i].ID < e.families[j].ID })
	for _, f := range e.families {
		for _, p := range []uint32{f.Partner1, f.Partner2} {
			if e.personIDs[p] {
				e.fams[p] = append(e.fams[p], f.ID)
			}
		}
		for _, c := range f.Children {
			if e.personIDs[c] {
				e.famc[c] = append(e.famc[c], f.ID)
			}
		}
	}

	for i := range ff.Sources {
		e.sources = append(e.sources, &ff.Sources[i])
		e.sourceIDs[ff.Sources[i].ID] = true
	}
	sort.Slice(e.sources, func(i, j int) bool { return e.sources[i].ID < e.sources[j].ID })

	for i := range ff.Notes {
		n := &ff.Notes[i]
		switch {
		case n.PersonID > 0:
			// Notes from .note files attach directly to a person.
			e.fileNotes[uint32(n.PersonID)] = append(e.fileNotes[uint32(n.PersonID)], n)
		case noteText(n) != "" && !e.noteIDs[n.ID]:
			e.notes = append(e.notes, n)
			e.noteIDs[n.ID] = true
		}
	}
	sort.Slice(e.notes, func(i, j int) bool { return e.notes[i].ID < e.notes[j].ID })

	for i := range ff.EventDefinitions {
		e.defs[ff.EventDefinitions[i].ID] = &ff.EventDefinitions[i]
	}
	for _, p := range ff.Places {
		e.places[int(p.ID)] = p.Name
	}
	return e
}

func (e *exporter) header() {
	e.line(0, "", "HEAD", "")
	e.line(1, "", "SOUR", "REUNION_EXPLORE")
	e.line(2, "", "NAME", "reunion-explore")
	if e.opts.ProgramVersion != "" {
		e.line(2, "", "VERS", e.opts.ProgramVersion)
	}
	e.line(1, "", "SUBM", "@U1@")
	e.line(1, "", "GEDC", "")
	e.line(2, "", "VERS", "5.5.1")
	e.line(2, "", "FORM", "LINEAGE-LINKED")
	e.line(1, "", "CHAR", "UTF-8")
	if e.ff.Version > 0 {
		e.text(1, "NOTE", fmt.Sprintf("Converted from a Reunion %d family file", e.ff.Version))
	}
}

func (e *exporter) person(p *model.Person) {
	e.line(0, xref("I", p.ID), "INDI", "")

	if p.GivenName != "" || p.Surname != "" || len(p.SourceCitations) > 0 {
		e.line(1, "", "NAME", strings.TrimSpace(clean(p.GivenName)+" /"+clean(p.Surname)+"/"))
		e.optional(2, "NPFX", p.PrefixTitle)
		e.optional(2, "GIVN", p.GivenName)
		e.optional(2, "SURN", p.Surname)
		e.optional(2, "NSFX", p.SuffixTitle)
		for _, ev := range p.Events {
			if def := e.defs[uint32(ev.SchemaID)]; def != nil && def.GEDCOMCode == "SPFX" && ev.Text != "" {
				e.optional(2, "SPFX", ev.Text)
			}
		}
		e.citations(2, p.SourceCitations)
	}
	for _, ev := range p.Events {
		if def := e.defs[uint32(ev.SchemaID)]; def != nil && (def.GEDCOMCode == "ALIA" || def.GEDCOMCode == "AKA") && ev.Text != "" {
			e.line(1, "", "NAME", clean(ev.Text))
			e.line(2, "", "TYPE", "aka")
		}
	}

	switch p.Sex {
	case model.SexMale:
		e.line(1, "", "SEX", "M")
	case model.SexFemale:
		e.line(1, "", "SEX", "F")
	default:
		e.line(1, "", "SEX", "U")
	}

	for _, ev := range p.Events {
		if ev.Tag < tagEventFirst {
			continue // note-type fields are written as NOTE links below
		}
		e.event(ev.SchemaID, ev.Tag >= tagFactFirst, ev.Date, ev.PlaceRefs, ev.Text, ev.SourceCitations, indiEvents, indiAttrs)
	}

	e.pictures(p.Pictures)
	e.noteRefs(p.NoteRefs)
	for _, n := range e.fileNotes[p.ID] {
		e.text(1, "NOTE", noteText(n))
	}
	for _, fid := range e.famc[p.ID] {
		e.line(1, "", "FAMC", xref("F", fid))
	}
	for _, fid := range e.fams[p.ID] {
		e.line(1, "", "FAMS", xref("F", fid))
	}
	e.optional(1, "REFN", p.UserID)
	e.line(1, "", "RIN", fmt.Sprint(p.ID))
}

func (e *exporter) family(f *model.Family) {
	e.line(0, xref("F", f.ID), "FAM", "")

	husb, wife := f.Partner1, f.Partner2
	if e.sex[husb] == model.SexFemale || e.sex[wife] == model.SexMale {
		husb, wife = wife, husb
	}
	if e.personIDs[husb] {
		e.line(1, "", "HUSB", xref("I", husb))
	}
	if e.personIDs[wife] {
		e.line(1, "", "WIFE", xref("I", wife))
	}
	for _, c := range f.Children {
		if e.personIDs[c] {
			e.line(1, "", "CHIL", xref("I", c))
		}
	}

	for _, ev := range f.Events {
		if ev.Tag >= tagNoteFirst && ev.Tag < tagEventFirst {
			continue // note-type fields are written as NOTE links below
		}
		e.event(ev.SchemaID, false, ev.Date, ev.PlaceRefs, ev.Text, ev.SourceCitations, famEvents, nil)
	}

	e.pictures(f.Pictures)
	e.noteRefs(f.NoteRefs)
	e.line(1, "", "RIN", fmt.Sprint(f.ID))
}

// event writes one event or attribute. Events whose GEDCOM code is not
// valid for the record are written as EVEN (or FACT) with a TYPE.
func (e *exporter) event(schemaID uint16, isFact bool, date string, placeRefs []int, text string, cites []model.SourceCitation, events, attrs map[string]bool) {
	def := e.defs[uint32(schemaID)]
	code, name := "", ""
	if def != nil {
		code, name = def.GEDCOMCode, def.DisplayName
	}
	if code == "SPFX" || code == "ALIA" || code == "AKA" {
		return // written as part of NAME
	}

	value, memo := "", text
	switch {
	case attrs[code] && code != "RESI":
		value, memo = text, ""
	case events[code]:
	case isFact:
		code, value, memo = "FACT", text, ""
	default:
		code = "EVEN"
	}
	if name == "" {
		name = fmt.Sprintf("Event %d", schemaID)
	}
	if (attrs[code] || code == "FACT") && clean(value) == "" && date == "" && len(placeRefs) == 0 && len(cites) == 0 {
		return // an attribute with no value carries no information
	}

	e.text(1, code, value)
	if code == "EVEN" || code == "FACT" {
		e.line(2, "", "TYPE", clean(name))
	}
	e.optional(2, "DATE", gedcomDate(date))
	if len(placeRefs) > 0 {
		e.optional(2, "PLAC", e.places[placeRefs[0]])
	}
	if memo != "" {
		e.text(2, "NOTE", memo)
	}
	e.citations(2, cites)
}

func (e *exporter) citations(level int, cites []model.SourceCitation) {
	for _, c := range cites {
		if !e.sourceIDs[c.SourceID] {
			continue
		}
		e.line(level, "", "SOUR", xref("S", c.SourceID))
		e.optional(level+1, "PAGE", c.Detail)
	}
}

func (e *exporter) pictures(pics []model.Picture) {
	for _, pic := range pics {
		if pic.Path == "" {
			continue
		}
		e.line(1, "", "OBJE", "")
		e.line(2, "", "FILE", clean(pic.Path))
		if ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(pic.Path), ".")); ext != "" {
			if ext == "jpeg" {
				ext = "jpg"
			}
			e.line(3, "", "FORM", ext)
		}
		e.optional(2, "TITL", pic.Caption)
	}
}

func (e *exporter) noteRefs(refs []model.NoteRef) {
	for _, r := range refs {
		if e.noteIDs[r.NoteID] {
			e.line(1, "", "NOTE", xref("N", r.NoteID))
		}
	}
}

func (e *exporter) source(s *model.Source) {
	e.line(0, xref("S", s.ID), "SOUR", "")

	title := s.Title
	var author, publisher string
	var other []string
	for _, f := range s.Fields {
		switch f.Label {
		case "Title":
			title = f.Value
		case "Author":
			author = f.Value
		case "Publisher":
			publisher = f.Value
		default:
			label := f.Label
			if label == "" {
				label = fmt.Sprintf("Field %d", f.SchemaID)
			}
			other = append(other, label+": "+f.Value)
		}
	}
	// Without template fields, title is Reunion's composed citation string.
	e.optional(1, "TITL", title)
	e.optional(1, "AUTH", author)
	e.optional(1, "PUBL", publisher)
	if len(other) > 0 {
		e.text(1, "NOTE", strings.Join(other, "\n"))
	}
	e.line(1, "", "RIN", fmt.Sprint(s.ID))
}

func (e *exporter) note(n *model.Note) {
	e.textX(0, xref("N", n.ID), "NOTE", noteText(n))
}

func noteText(n *model.Note) string {
	if n.DisplayText != "" {
		return n.DisplayText
	}
	return n.RawText
}

// optional writes a line only when value is non-empty.
func (e *exporter) optional(level int, tag, value string) {
	if v := clean(value); v != "" {
		e.text(level, tag, v)
	}
}

// text writes a value that may span lines (CONT) or exceed the line
// length limit (CONC).
func (e *exporter) text(level int, tag, value string) {
	e.textX(level, "", tag, value)
}

func (e *exporter) textX(level int, xrefID, tag, value string) {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	for i, l := range lines {
		l = strings.ReplaceAll(clean(l), "@", "@@")
		chunks := splitChunks(l, maxLineValue)
		for j, c := range chunks {
			switch {
			case i == 0 && j == 0:
				e.line(level, xrefID, tag, c)
			case j == 0:
				e.line(level+1, "", "CONT", c)
			default:
				e.line(level+1, "", "CONC", c)
			}
		}
	}
}

// line writes one GEDCOM line. value must already be cleaned and escaped
// unless it is a pointer.
func (e *exporter) line(level int, xrefID, tag, value string) {
	fmt.Fprint(e.out, level)
	if xrefID != "" {
		fmt.Fprint(e.out, " ", xrefID)
	}
	fmt.Fprint(e.out, " ", tag)
	if value != "" {
		fmt.Fprint(e.out, " ", value)
	}
	fmt.Fprint(e.out, "\n")
}

// splitChunks splits s into pieces of at most n bytes on rune boundaries,
// avoiding splits next to a space (CONC joins pieces without a separator,
// and many readers trim trailing or leading spaces).
func splitChunks(s string, n int) []string {
	var out []string
	for len(s) > n {
		cut := n
		for cut > n/2 && (!utf8.RuneStart(s[cut]) || s[cut] == ' ' || s[cut-1] == ' ') {
			cut--
		}
		for !utf8.RuneStart(s[cut]) {
			cut--
		}
		out = append(out, s[:cut])
		s = s[cut:]
	}
	return append(out, s)
}

// clean removes control characters, which GEDCOM values cannot contain,
// and turns tabs into spaces.
func clean(s string) string {
	return strings.TrimSpace(strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r == utf8.RuneError || unicode.IsControl(r):
			return -1
		}
		return r
	}, s))
}

func xref(prefix string, id uint32) string {
	return fmt.Sprintf("@%s%d@", prefix, id)
}

var dateWords = strings.NewReplacer(
	"about ", "ABT ", "after ", "AFT ", "before ", "BEF ",
	"Jan", "JAN", "Feb", "FEB", "Mar", "MAR", "Apr", "APR", "May", "MAY", "Jun", "JUN",
	"Jul", "JUL", "Aug", "AUG", "Sep", "SEP", "Oct", "OCT", "Nov", "NOV", "Dec", "DEC",
)

// gedcomDate converts a parsed date ("about 12 Sep 1953") to GEDCOM form
// ("ABT 12 SEP 1953").
func gedcomDate(d string) string {
	return dateWords.Replace(d)
}
