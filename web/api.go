package web

import (
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"

	"github.com/kedoco/reunion-explore/index"
	"github.com/kedoco/reunion-explore/model"
)

// --- Response Types ---

// PersonRef is a lightweight person reference for lists and links.
type PersonRef struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
	Sex  string `json:"sex"`
}

// TreeEntryRef is a lightweight ancestor/descendant entry.
type TreeEntryRef struct {
	Generation int    `json:"generation"`
	ID         uint32 `json:"id"`
	Name       string `json:"name"`
	Sex        string `json:"sex"`
}

// SourceCitationDisplay is a resolved source citation for display.
type SourceCitationDisplay struct {
	SourceID    uint32 `json:"source_id"`
	SourceTitle string `json:"source_title,omitempty"`
	Detail      string `json:"detail,omitempty"`
}

// PersonDetail is a full person record with resolved relationships.
type PersonDetail struct {
	ID              uint32                  `json:"id"`
	GivenName       string                  `json:"given_name,omitempty"`
	Surname         string                  `json:"surname,omitempty"`
	PrefixTitle     string                  `json:"prefix_title,omitempty"`
	SuffixTitle     string                  `json:"suffix_title,omitempty"`
	UserID          string                  `json:"user_id,omitempty"`
	Name            string                  `json:"name"`
	Sex             int                     `json:"sex"`
	SexLabel        string                  `json:"sex_label"`
	SourceCitations []SourceCitationDisplay  `json:"source_citations,omitempty"`
	ResolvedEvents  []ResolvedEvent         `json:"resolved_events,omitempty"`
	Notes           []NoteDisplay           `json:"notes,omitempty"`
	Pictures        []model.Picture         `json:"pictures,omitempty"`
	Spouses         []PersonRef             `json:"spouses,omitempty"`
	Children        []PersonRef             `json:"children,omitempty"`
	Parents         []PersonRef             `json:"parents,omitempty"`
	Siblings        []PersonRef             `json:"siblings,omitempty"`
}

// ResolvedEvent is a person event with resolved schema and place names.
type ResolvedEvent struct {
	SchemaID        uint16                  `json:"schema_id"`
	SchemaName      string                  `json:"schema_name,omitempty"`
	Tag             uint16                  `json:"tag"`
	Date            string                  `json:"date,omitempty"`
	Text            string                  `json:"text,omitempty"`
	Places          []PlaceRef              `json:"places,omitempty"`
	SourceCitations []SourceCitationDisplay  `json:"source_citations,omitempty"`
	IsNote          bool                    `json:"is_note,omitempty"`
	IsFact          bool                    `json:"is_fact,omitempty"`
}

// PlaceRef is a lightweight place reference.
type PlaceRef struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

// NoteDisplay is a lightweight note with display text for person detail views.
type NoteDisplay struct {
	ID    uint32 `json:"id"`
	Label string `json:"label,omitempty"`
	Text  string `json:"text"`
	HTML  string `json:"html,omitempty"`
}

// FamilyRef is a lightweight family reference for lists.
type FamilyRef struct {
	ID            uint32 `json:"id"`
	Partner1Name  string `json:"partner1_name,omitempty"`
	Partner2Name  string `json:"partner2_name,omitempty"`
	ChildrenCount int    `json:"children_count"`
}

// FamilyDetail is a full family record with resolved names.
type FamilyDetail struct {
	ID             uint32      `json:"id"`
	Partner1       uint32      `json:"partner1,omitempty"`
	Partner2       uint32      `json:"partner2,omitempty"`
	Partner1Detail *PersonRef  `json:"partner1_detail,omitempty"`
	Partner2Detail *PersonRef  `json:"partner2_detail,omitempty"`
	ChildrenDetail []PersonRef `json:"children_detail,omitempty"`
}

// StatsResponse contains summary counts.
type StatsResponse struct {
	Persons             int `json:"persons"`
	PersonsNamed        int `json:"persons_named"`
	PersonsMale         int `json:"persons_male"`
	PersonsFemale       int `json:"persons_female"`
	PersonsUnknownSex   int `json:"persons_unknown_sex"`
	Families            int `json:"families"`
	FamiliesWithPartners int `json:"families_with_partners"`
	FamiliesWithChildren int `json:"families_with_children"`
	Places              int `json:"places"`
	EventTypes          int `json:"event_types"`
	Sources             int `json:"sources"`
	Notes               int `json:"notes"`
	Media               int `json:"media"`
}

// SummaryResponse provides per-person statistics.
type SummaryResponse struct {
	Person      string         `json:"person"`
	ID          uint32         `json:"id"`
	Spouses     int            `json:"spouses"`
	Siblings    int            `json:"siblings"`
	Ancestors   int            `json:"ancestors"`
	Descendants int            `json:"descendants"`
	Treetops    int            `json:"treetops"`
	Surnames    map[string]int `json:"surnames,omitempty"`
}

// PaginatedResponse wraps a paginated list.
type PaginatedResponse struct {
	Items any `json:"items"`
	Total int `json:"total"`
	Page  int `json:"page"`
}

// --- Helpers ---

func (s *Server) personRef(id uint32) PersonRef {
	p, ok := s.load().idx.Persons[id]
	if !ok {
		return PersonRef{ID: id, Name: "?"}
	}
	return PersonRef{ID: id, Name: index.FormatName(p), Sex: p.Sex.String()}
}

func (s *Server) personRefs(ids []uint32) []PersonRef {
	refs := make([]PersonRef, 0, len(ids))
	for _, id := range ids {
		refs = append(refs, s.personRef(id))
	}
	return refs
}

func (s *Server) resolveSourceCitations(cites []model.SourceCitation) []SourceCitationDisplay {
	if len(cites) == 0 {
		return nil
	}
	out := make([]SourceCitationDisplay, 0, len(cites))
	for _, c := range cites {
		title := ""
		if src, ok := s.load().idx.Sources[c.SourceID]; ok {
			title = src.Title
		}
		out = append(out, SourceCitationDisplay{
			SourceID:    c.SourceID,
			SourceTitle: title,
			Detail:      c.Detail,
		})
	}
	return out
}

func (s *Server) resolveEvents(events []model.PersonEvent) []ResolvedEvent {
	resolved := make([]ResolvedEvent, 0, len(events))
	for _, evt := range events {
		re := ResolvedEvent{
			SchemaID:        evt.SchemaID,
			SchemaName:      s.load().idx.SchemaName(evt.SchemaID),
			Tag:             evt.Tag,
			Date:            evt.Date,
			Text:            evt.Text,
			SourceCitations: s.resolveSourceCitations(evt.SourceCitations),
			IsNote:          evt.Tag < 0x03E8, // tags below 1000 are note references
			IsFact:          evt.Tag >= 0x0BB8,
		}
		for _, placeRef := range evt.PlaceRefs {
			pid := uint32(placeRef)
			name := s.load().idx.PlaceName(placeRef)
			re.Places = append(re.Places, PlaceRef{ID: pid, Name: name})
		}
		resolved = append(resolved, re)
	}
	return resolved
}

// renderMarkupHTML converts markup nodes to an HTML string.
func (s *Server) renderMarkupHTML(nodes []model.MarkupNode) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case model.MarkupText:
			b.WriteString(html.EscapeString(n.Text))
		case model.MarkupBold:
			b.WriteString("<strong>")
			b.WriteString(s.renderMarkupHTML(n.Children))
			b.WriteString("</strong>")
		case model.MarkupItalic:
			b.WriteString("<em>")
			b.WriteString(s.renderMarkupHTML(n.Children))
			b.WriteString("</em>")
		case model.MarkupUnderline:
			b.WriteString("<u>")
			b.WriteString(s.renderMarkupHTML(n.Children))
			b.WriteString("</u>")
		case model.MarkupURL:
			b.WriteString(`<a href="`)
			b.WriteString(html.EscapeString(n.Value))
			b.WriteString(`" target="_blank" rel="noopener">`)
			b.WriteString(s.renderMarkupHTML(n.Children))
			b.WriteString("</a>")
		case model.MarkupSourceCitation:
			title := "Source " + n.Value
			if src, ok := s.load().idx.Sources[parseUint32(n.Value)]; ok && src.Title != "" {
				title = src.Title
			}
			b.WriteString(`<sup class="source-cite-group" title="`)
			b.WriteString(html.EscapeString(title))
			b.WriteString(`"><a href="#source/`)
			b.WriteString(html.EscapeString(n.Value))
			b.WriteString(`">`)
			b.WriteString(html.EscapeString(n.Value))
			b.WriteString("</a></sup>")
		default:
			// MarkupFontFlag, MarkupColor, etc. — render children only
			b.WriteString(s.renderMarkupHTML(n.Children))
		}
	}
	return b.String()
}

func parseUint32(s string) uint32 {
	var v uint32
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + uint32(c-'0')
		}
	}
	return v
}

// --- Handlers ---

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	male, female, unknown, named := 0, 0, 0, 0
	for _, p := range s.load().ff.Persons {
		switch p.Sex {
		case model.SexMale:
			male++
		case model.SexFemale:
			female++
		default:
			unknown++
		}
		if p.GivenName != "" || p.Surname != "" {
			named++
		}
	}
	withPartners, withChildren := 0, 0
	for _, f := range s.load().ff.Families {
		if f.Partner1 > 0 || f.Partner2 > 0 {
			withPartners++
		}
		if len(f.Children) > 0 {
			withChildren++
		}
	}
	writeJSON(w, http.StatusOK, StatsResponse{
		Persons:              len(s.load().ff.Persons),
		PersonsNamed:         named,
		PersonsMale:          male,
		PersonsFemale:        female,
		PersonsUnknownSex:    unknown,
		Families:             len(s.load().ff.Families),
		FamiliesWithPartners: withPartners,
		FamiliesWithChildren: withChildren,
		Places:               len(s.load().ff.Places),
		EventTypes:           len(s.load().ff.EventDefinitions),
		Sources:              len(s.load().ff.Sources),
		Notes:                len(s.load().ff.Notes),
		Media:                len(s.load().ff.MediaRefs),
	})
}

func (s *Server) handlePersons(w http.ResponseWriter, r *http.Request) {
	surname := r.URL.Query().Get("surname")
	query := r.URL.Query().Get("q")
	page := parseIntQuery(r, "page", 1)
	perPage := parseIntQuery(r, "per_page", 100)

	var refs []PersonRef
	for i := range s.load().ff.Persons {
		p := &s.load().ff.Persons[i]
		if surname != "" && !strings.EqualFold(p.Surname, surname) {
			continue
		}
		if query != "" {
			name := strings.ToLower(index.FormatName(p))
			if !strings.Contains(name, strings.ToLower(query)) {
				continue
			}
		}
		refs = append(refs, PersonRef{ID: p.ID, Name: index.FormatName(p), Sex: p.Sex.String()})
	}

	total := len(refs)
	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Items: refs[start:end],
		Total: total,
		Page:  page,
	})
}

func (s *Server) handlePerson(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, ok := s.load().idx.Persons[id]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("person %d not found", id))
		return
	}

	// Collect notes from person's NoteRefs (inline notes linked via event data)
	var notes []NoteDisplay
	for _, ref := range p.NoteRefs {
		n, ok := s.load().idx.Notes[ref.NoteID]
		if !ok || n.DisplayText == "" {
			continue
		}
		nd := NoteDisplay{
			ID:    n.ID,
			Label: s.load().idx.SchemaName(ref.SchemaID),
			Text:  n.DisplayText,
		}
		if len(n.Markup) > 0 {
			nd.HTML = s.renderMarkupHTML(n.Markup)
		}
		notes = append(notes, nd)
	}
	// Also include file-based notes linked by PersonID, skipping duplicates
	seen := make(map[string]bool, len(notes))
	for _, nd := range notes {
		seen[nd.Text] = true
	}
	for i := range s.load().ff.Notes {
		n := &s.load().ff.Notes[i]
		if n.PersonID == int(p.ID) && n.DisplayText != "" && !seen[n.DisplayText] {
			nd := NoteDisplay{
				ID:    n.ID,
				Label: s.load().idx.SchemaName(uint16(n.SourceID)),
				Text:  n.DisplayText,
			}
			if len(n.Markup) > 0 {
				nd.HTML = s.renderMarkupHTML(n.Markup)
			}
			notes = append(notes, nd)
		}
	}

	detail := PersonDetail{
		ID:              p.ID,
		GivenName:       p.GivenName,
		Surname:         p.Surname,
		PrefixTitle:     p.PrefixTitle,
		SuffixTitle:     p.SuffixTitle,
		UserID:          p.UserID,
		Name:            index.FormatName(p),
		Sex:             int(p.Sex),
		SexLabel:        p.Sex.String(),
		SourceCitations: s.resolveSourceCitations(p.SourceCitations),
		ResolvedEvents:  s.resolveEvents(p.Events),
		Notes:           notes,
		Pictures:        p.Pictures,
		Spouses:         s.personRefs(s.load().idx.Spouses(p.ID)),
		Children:        s.personRefs(s.load().idx.ChildrenOf(p.ID)),
		Parents:         s.personRefs(s.load().idx.Parents(p.ID)),
		Siblings:        s.personRefs(s.load().idx.Siblings(p.ID)),
	}

	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handlePersonFamilies(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := s.load().idx.Persons[id]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("person %d not found", id))
		return
	}

	famIDs := s.load().idx.FamiliesForPerson(id)
	details := make([]FamilyDetail, 0, len(famIDs))
	for _, fid := range famIDs {
		f, ok := s.load().idx.Families[fid]
		if !ok {
			continue
		}
		details = append(details, s.buildFamilyDetail(f))
	}
	writeJSON(w, http.StatusOK, details)
}

func (s *Server) handlePersonAncestors(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := s.load().idx.Persons[id]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("person %d not found", id))
		return
	}
	gen := parseIntQuery(r, "generations", 10)
	entries := s.load().idx.Ancestors(id, gen)
	refs := make([]TreeEntryRef, 0, len(entries))
	for _, e := range entries {
		refs = append(refs, TreeEntryRef{
			Generation: e.Generation,
			ID:         e.Person.ID,
			Name:       index.FormatName(e.Person),
			Sex:        e.Person.Sex.String(),
		})
	}
	writeJSON(w, http.StatusOK, refs)
}

func (s *Server) handlePersonDescendants(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := s.load().idx.Persons[id]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("person %d not found", id))
		return
	}
	gen := parseIntQuery(r, "generations", 10)
	entries := s.load().idx.Descendants(id, gen)
	refs := make([]TreeEntryRef, 0, len(entries))
	for _, e := range entries {
		refs = append(refs, TreeEntryRef{
			Generation: e.Generation,
			ID:         e.Person.ID,
			Name:       index.FormatName(e.Person),
			Sex:        e.Person.Sex.String(),
		})
	}
	writeJSON(w, http.StatusOK, refs)
}

func (s *Server) handlePersonTreetops(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := s.load().idx.Persons[id]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("person %d not found", id))
		return
	}
	persons := s.load().idx.Treetops(id)
	refs := make([]PersonRef, 0, len(persons))
	for _, p := range persons {
		refs = append(refs, PersonRef{ID: p.ID, Name: index.FormatName(p), Sex: p.Sex.String()})
	}
	writeJSON(w, http.StatusOK, refs)
}

func (s *Server) handlePersonSummary(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, ok := s.load().idx.Persons[id]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("person %d not found", id))
		return
	}

	spouses := s.load().idx.Spouses(p.ID)
	siblings := s.load().idx.Siblings(p.ID)
	ancestors := s.load().idx.Ancestors(p.ID, 100)
	descendants := s.load().idx.Descendants(p.ID, 100)
	treetops := s.load().idx.Treetops(p.ID)

	surnameCounts := make(map[string]int)
	for _, a := range ancestors {
		if a.Person.Surname != "" {
			surnameCounts[a.Person.Surname]++
		}
	}

	writeJSON(w, http.StatusOK, SummaryResponse{
		Person:      index.FormatName(p),
		ID:          p.ID,
		Spouses:     len(spouses),
		Siblings:    len(siblings),
		Ancestors:   len(ancestors),
		Descendants: len(descendants),
		Treetops:    len(treetops),
		Surnames:    surnameCounts,
	})
}

func (s *Server) handleFamilies(w http.ResponseWriter, r *http.Request) {
	page := parseIntQuery(r, "page", 1)
	perPage := parseIntQuery(r, "per_page", 100)

	refs := make([]FamilyRef, 0, len(s.load().ff.Families))
	for _, f := range s.load().ff.Families {
		ref := FamilyRef{
			ID:            f.ID,
			ChildrenCount: len(f.Children),
		}
		if f.Partner1 > 0 {
			ref.Partner1Name = s.load().idx.PersonName(f.Partner1)
		}
		if f.Partner2 > 0 {
			ref.Partner2Name = s.load().idx.PersonName(f.Partner2)
		}
		refs = append(refs, ref)
	}

	total := len(refs)
	start := (page - 1) * perPage
	if start > total {
		start = total
	}
	end := start + perPage
	if end > total {
		end = total
	}

	writeJSON(w, http.StatusOK, PaginatedResponse{
		Items: refs[start:end],
		Total: total,
		Page:  page,
	})
}

func (s *Server) handleFamily(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	f, ok := s.load().idx.Families[id]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("family %d not found", id))
		return
	}
	writeJSON(w, http.StatusOK, s.buildFamilyDetail(f))
}

func (s *Server) buildFamilyDetail(f *model.Family) FamilyDetail {
	d := FamilyDetail{
		ID:       f.ID,
		Partner1: f.Partner1,
		Partner2: f.Partner2,
	}
	if f.Partner1 > 0 {
		ref := s.personRef(f.Partner1)
		d.Partner1Detail = &ref
	}
	if f.Partner2 > 0 {
		ref := s.personRef(f.Partner2)
		d.Partner2Detail = &ref
	}
	d.ChildrenDetail = make([]PersonRef, 0, len(f.Children))
	for _, cid := range f.Children {
		d.ChildrenDetail = append(d.ChildrenDetail, s.personRef(cid))
	}
	return d
}

func (s *Server) handlePlaces(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.load().ff.Places)
}

func (s *Server) handlePlace(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	p, ok := s.load().idx.Places[id]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("place %d not found", id))
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handlePlacePersons(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := s.load().idx.Places[id]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("place %d not found", id))
		return
	}
	personIDs := s.load().idx.PersonsByPlace(id)
	writeJSON(w, http.StatusOK, s.personRefs(personIDs))
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.load().ff.EventDefinitions)
}

func (s *Server) handleEvent(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	e, ok := s.load().idx.Schemas[id]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("event type %d not found", id))
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) handleEventPersons(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := s.load().idx.Schemas[id]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("event type %d not found", id))
		return
	}
	personIDs := s.load().idx.PersonsBySchema(id)
	writeJSON(w, http.StatusOK, s.personRefs(personIDs))
}

func (s *Server) handleSources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.load().ff.Sources)
}

func (s *Server) handleSource(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	src, ok := s.load().idx.Sources[id]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("source %d not found", id))
		return
	}
	writeJSON(w, http.StatusOK, src)
}

func (s *Server) handleSourcePersons(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if _, ok := s.load().idx.Sources[id]; !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("source %d not found", id))
		return
	}
	// Find all persons that cite this source (person-level or event-level)
	var refs []PersonRef
	seen := make(map[uint32]bool)
	for i := range s.load().ff.Persons {
		p := &s.load().ff.Persons[i]
		found := false
		for _, sc := range p.SourceCitations {
			if sc.SourceID == id {
				found = true
				break
			}
		}
		if !found {
			for _, evt := range p.Events {
				for _, sc := range evt.SourceCitations {
					if sc.SourceID == id {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
		}
		if found && !seen[p.ID] {
			seen[p.ID] = true
			refs = append(refs, PersonRef{ID: p.ID, Name: index.FormatName(p), Sex: p.Sex.String()})
		}
	}
	writeJSON(w, http.StatusOK, refs)
}

func (s *Server) handleNotes(w http.ResponseWriter, r *http.Request) {
	personIDStr := r.URL.Query().Get("person_id")
	if personIDStr != "" {
		pid := parseIntQuery(r, "person_id", 0)
		var filtered []model.Note
		for _, n := range s.load().ff.Notes {
			if n.PersonID == pid {
				filtered = append(filtered, n)
			}
		}
		writeJSON(w, http.StatusOK, filtered)
		return
	}
	writeJSON(w, http.StatusOK, s.load().ff.Notes)
}

func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	n, ok := s.load().idx.Notes[id]
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("note %d not found", id))
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeJSON(w, http.StatusOK, []PersonRef{})
		return
	}
	persons := s.load().idx.Search(query)
	// Sort by name for consistent results
	sort.Slice(persons, func(i, j int) bool {
		return index.FormatName(persons[i]) < index.FormatName(persons[j])
	})
	refs := make([]PersonRef, 0, len(persons))
	for _, p := range persons {
		refs = append(refs, PersonRef{ID: p.ID, Name: index.FormatName(p), Sex: p.Sex.String()})
	}
	writeJSON(w, http.StatusOK, refs)
}
