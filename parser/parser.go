// Package parser provides the Reunion 13/14 parser implementation and orchestrator.
package parser

import (
	"fmt"
	"strings"

	reunion "github.com/kedoco/reunion-explore"
	"github.com/kedoco/reunion-explore/bundle"
	"github.com/kedoco/reunion-explore/model"
	"github.com/kedoco/reunion-explore/parser/cache"
	"github.com/kedoco/reunion-explore/parser/familydata"
	"github.com/kedoco/reunion-explore/parser/member"
	"github.com/kedoco/reunion-explore/parser/notes"
)

func init() {
	reunion.RegisterVersion(&Parser{version: reunion.Version13})
	reunion.RegisterVersion(&Parser{version: reunion.Version14})
}

// Parser implements VersionParser for Reunion 13 and 14, which share the
// same bundle layout and familydata record format.
type Parser struct {
	version reunion.Version
}

func (p *Parser) Version() reunion.Version { return p.version }

func (p *Parser) CanParse(bundlePath string) (bool, error) {
	b, err := bundle.OpenBundle(bundlePath)
	if err != nil {
		return false, err
	}
	return b.FamilyData != "" && b.Signature != "", nil
}

func (p *Parser) Parse(bundlePath string, opts reunion.ParseOptions) (*model.FamilyFile, error) {
	b, err := bundle.OpenBundle(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("opening bundle: %w", err)
	}

	ec := reunion.NewErrorCollector(opts.MaxErrors)
	ff := &model.FamilyFile{
		Version: int(p.version),
	}

	// Parse signature
	if b.Signature != "" {
		sig, err := ParseSignature(b.Signature)
		if err != nil {
			ec.Add("signature", -1, "failed to parse signature", err)
		} else {
			ff.Signature = sig
		}
	}

	// Parse familydata
	if b.FamilyData != "" {
		result, err := familydata.Parse(b.FamilyData, ec)
		if err != nil {
			return nil, fmt.Errorf("parsing familydata: %w", err)
		}
		ff.Header = result.Header
		ff.Persons = result.Persons
		ff.Families = result.Families
		ff.EventDefinitions = result.EventDefinitions
		ff.Sources = result.Sources
		ff.MediaRefs = result.MediaRefs
		ff.Places = result.Places
		// Inline notes from familydata
		ff.Notes = result.Notes
	}

	// Enrich familydata place names with full-length names from places.cache.
	// Familydata has correct IDs but truncated names; places.cache has full names.
	if path, ok := b.Caches["places.cache"]; ok {
		cachePlaces, err := cache.ParsePlaces(path)
		if err != nil {
			ec.Add("places.cache", -1, "failed to parse for name enrichment", err)
		} else {
			enrichPlaceNames(ff.Places, cachePlaces)
		}
	}

	if path, ok := b.Caches["placeUsage.cache"]; ok {
		usages, err := cache.ParsePlaceUsage(path)
		if err != nil {
			ec.Add("placeUsage.cache", -1, "failed to parse", err)
		} else {
			ff.PlaceUsages = usages
		}
	}

	if path, ok := b.Caches["fmnames.cache"]; ok {
		names, err := cache.ParseFmNames(path)
		if err != nil {
			ec.Add("fmnames.cache", -1, "failed to parse", err)
		} else {
			ff.FirstNames = names
		}
	}

	if path, ok := b.Caches["surnames.cache"]; ok {
		entries, err := cache.ParseSurnames(path)
		if err != nil {
			ec.Add("surnames.cache", -1, "failed to parse", err)
		} else {
			ff.Surnames = entries
		}
	}

	if path, ok := b.Caches["shNames.cache"]; ok {
		names, err := cache.ParseShNames(path)
		if err != nil {
			ec.Add("shNames.cache", -1, "failed to parse", err)
		} else {
			ff.SearchNames = names
		}
	}

	if path, ok := b.Caches["timestamps.cache"]; ok {
		entries, err := cache.ParseTimestamps(path)
		if err != nil {
			ec.Add("timestamps.cache", -1, "failed to parse", err)
		} else {
			ff.Timestamps = entries
		}
	}

	if path, ok := b.Caches["globalRecords.cache"]; ok {
		entry, err := cache.ParseGlobalRecords(path)
		if err != nil {
			ec.Add("globalRecords.cache", -1, "failed to parse", err)
		} else {
			ff.GlobalRecords = entry
		}
	}

	if path, ok := b.Caches["bookmarks.cache"]; ok {
		bk, err := cache.ParseBookmarks(path)
		if err != nil {
			ec.Add("bookmarks.cache", -1, "failed to parse", err)
		} else {
			ff.Bookmarks = bk
		}
	}

	if path, ok := b.Caches["colortags.cache"]; ok {
		tags, err := cache.ParseColorTags(path)
		if err != nil {
			ec.Add("colortags.cache", -1, "failed to parse", err)
		} else {
			ff.ColorTags = tags
		}
	}

	if path, ok := b.Caches["associations.cache"]; ok {
		assocs, err := cache.ParseAssociations(path)
		if err != nil {
			ec.Add("associations.cache", -1, "failed to parse", err)
		} else {
			ff.Associations = assocs
		}
	}

	if path, ok := b.Caches["find.cache"]; ok {
		text, err := cache.ParseFind(path)
		if err != nil {
			ec.Add("find.cache", -1, "failed to parse", err)
		} else {
			ff.FindText = text
		}
	}

	if path, ok := b.Caches["descriptions.cache"]; ok {
		desc, err := cache.ParseDescriptions(path)
		if err != nil {
			ec.Add("descriptions.cache", -1, "failed to parse", err)
		} else {
			ff.Description = desc
		}
	}

	// Parse note files from all members
	if len(b.NoteFiles) > 0 {
		fileNotes, err := notes.ParseAllNotes(b.NoteFiles)
		if err != nil {
			ec.Add("notes", -1, "failed to parse note files", err)
		} else {
			ff.Notes = append(ff.Notes, fileNotes...)
		}
	}

	// Parse members
	if len(b.Members) > 0 {
		members, err := member.ParseMembers(b.Members)
		if err != nil {
			ec.Add("members", -1, "failed to parse members", err)
		} else {
			ff.Members = members
		}
	}

	// Collect warnings
	for _, pe := range ec.Errors() {
		ff.Warnings = append(ff.Warnings, pe.Error())
	}

	return ff, nil
}

// enrichPlaceNames upgrades truncated familydata place names with full-length
// names from places.cache. It first tries matching by ID. For remaining
// unmatched places, it falls back to prefix matching, requiring the prefix
// boundary to fall at a natural break point (comma, space) in the full name
// to avoid false matches like "Paris" → "Paris, Texas".
func enrichPlaceNames(places []model.Place, cachePlaces []model.Place) {
	// Build ID-keyed lookup from cache
	byID := make(map[uint32]string, len(cachePlaces))
	cacheNames := make([]string, 0, len(cachePlaces))
	for _, cp := range cachePlaces {
		if cp.Name != "" {
			byID[cp.ID] = cp.Name
			cacheNames = append(cacheNames, cp.Name)
		}
	}

	for i := range places {
		name := places[i].Name
		if name == "" {
			continue
		}
		// Strategy 1: match by ID
		if cn, ok := byID[places[i].ID]; ok && len(cn) > len(name) && strings.HasPrefix(cn, name) {
			places[i].Name = cn
			continue
		}

		// Strategy 2: prefix match with truncation detection.
		// The binary format uses fixed-width fields, so truncation can cut
		// mid-word ("Califo") or right after a delimiter ("Glen Cove, New ").
		// We accept the match if:
		//   - The name has a trailing space (clearly cut mid-field), OR
		//   - The next char in the full name is a letter/digit (mid-word cut)
		// We reject if the name ends at a clean word boundary and the full
		// name continues with a delimiter (e.g., "Paris" → "Paris, Texas").
		bestMatch := ""
		for _, cn := range cacheNames {
			if cn == name {
				bestMatch = ""
				break
			}
			if len(cn) > len(name) && strings.HasPrefix(cn, name) {
				trailingSpace := name[len(name)-1] == ' '
				nextChar := cn[len(name)]
				midWordCut := nextChar != ',' && nextChar != ' '
				if trailingSpace || midWordCut {
					if bestMatch == "" || len(cn) < len(bestMatch) {
						bestMatch = cn
					}
				}
			}
		}
		if bestMatch != "" {
			places[i].Name = bestMatch
		}
	}

	// Strategy 3: fix truncated trailing components.
	// Build component frequency from cache to identify reliable components.
	// Components appearing only once that are prefixes of more-common ones
	// are likely truncation artifacts in the cache itself.
	compFreq := make(map[string]int)
	for _, cp := range cachePlaces {
		for _, part := range strings.Split(cp.Name, ", ") {
			part = strings.TrimSpace(part)
			if len(part) > 1 {
				compFreq[part]++
			}
		}
	}
	// Build clean component set: exclude single-occurrence components that
	// are strict prefixes of a more-frequent component.
	components := make(map[string]bool)
	for comp, freq := range compFreq {
		if freq == 1 {
			isPrefix := false
			for other, oFreq := range compFreq {
				if oFreq > 1 && len(other) > len(comp) && strings.HasPrefix(other, comp) {
					isPrefix = true
					break
				}
			}
			if isPrefix {
				continue // likely a truncation artifact
			}
		}
		components[comp] = true
	}

	for i := range places {
		name := places[i].Name
		commaIdx := strings.LastIndex(name, ", ")
		if commaIdx == -1 {
			continue
		}
		lastPart := name[commaIdx+2:]
		if lastPart == "" || components[lastPart] {
			continue // empty or already a known-good component
		}
		// Find the shortest known component this is a prefix of
		var match string
		for comp := range components {
			if len(comp) > len(lastPart) && strings.HasPrefix(comp, lastPart) {
				if match == "" || len(comp) < len(match) {
					match = comp
				}
			}
		}
		if match != "" {
			places[i].Name = name[:commaIdx+2] + match
		}
	}
}
