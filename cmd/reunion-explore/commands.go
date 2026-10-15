package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kedoco/reunion-explore/index"
	"github.com/kedoco/reunion-explore/model"
)

// printJSON marshals v as indented JSON to stdout.
func printJSON(v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing JSON: %w", err)
	}
	_, err = os.Stdout.Write(data)
	if err != nil {
		return err
	}
	fmt.Println()
	return nil
}

// --- json ---

func cmdJSON(ff *model.FamilyFile) error {
	return printJSON(ff)
}

// --- stats ---

func cmdStats(ff *model.FamilyFile, idx *Index, asJSON bool) error {
	male, female, unknown := 0, 0, 0
	named := 0
	for _, p := range ff.Persons {
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

	withPartners := 0
	withChildren := 0
	for _, f := range ff.Families {
		if f.Partner1 > 0 || f.Partner2 > 0 {
			withPartners++
		}
		if len(f.Children) > 0 {
			withChildren++
		}
	}

	if asJSON {
		return printJSON(map[string]interface{}{
			"persons":        len(ff.Persons),
			"persons_named":  named,
			"persons_male":   male,
			"persons_female": female,
			"persons_unknown_sex": unknown,
			"families":            len(ff.Families),
			"families_with_partners": withPartners,
			"families_with_children": withChildren,
			"places":       len(ff.Places),
			"event_types":  len(ff.EventDefinitions),
			"sources":      len(ff.Sources),
			"notes":        len(ff.Notes),
			"media":        len(ff.MediaRefs),
		})
	}

	fmt.Printf("Persons:          %d\n", len(ff.Persons))
	fmt.Printf("  Named:          %d\n", named)
	fmt.Printf("  Male:           %d\n", male)
	fmt.Printf("  Female:         %d\n", female)
	fmt.Printf("  Unknown sex:    %d\n", unknown)
	fmt.Printf("Families:         %d\n", len(ff.Families))
	fmt.Printf("  With partners:  %d\n", withPartners)
	fmt.Printf("  With children:  %d\n", withChildren)
	fmt.Printf("Places:           %d\n", len(ff.Places))
	fmt.Printf("Event types:      %d\n", len(ff.EventDefinitions))
	fmt.Printf("Sources:          %d\n", len(ff.Sources))
	fmt.Printf("Notes:            %d\n", len(ff.Notes))
	fmt.Printf("Media:            %d\n", len(ff.MediaRefs))
	return nil
}

// --- persons ---

func cmdPersons(ff *model.FamilyFile, idx *Index, surname string, asJSON bool) error {
	var filtered []model.Person
	for _, p := range ff.Persons {
		if surname != "" && !strings.EqualFold(p.Surname, surname) {
			continue
		}
		filtered = append(filtered, p)
	}

	if asJSON {
		return printJSON(filtered)
	}

	for i := range filtered {
		p := &filtered[i]
		fmt.Printf("#%-6d %s  %s\n", p.ID, p.Sex, FormatName(p))
	}
	return nil
}

// --- person ---

func cmdPerson(ff *model.FamilyFile, idx *Index, id uint32, asJSON bool) error {
	p, ok := idx.Persons[id]
	if !ok {
		return fmt.Errorf("person %d not found", id)
	}

	if asJSON {
		return printJSON(p)
	}

	fmt.Printf("Person #%d: %s\n", p.ID, FormatName(p))
	fmt.Printf("Sex: %s\n", p.Sex)

	if len(p.Events) > 0 {
		fmt.Println("\nEvents:")
		for _, evt := range p.Events {
			name := idx.SchemaName(evt.SchemaID)
			if name == "" {
				name = fmt.Sprintf("tag:0x%04X", evt.Tag)
			}
			line := fmt.Sprintf("  - %s", name)
			for _, ref := range evt.PlaceRefs {
				pname := idx.PlaceName(ref)
				if pname != "" {
					line += fmt.Sprintf("  @ %s", pname)
				}
			}
			fmt.Println(line)
		}
	}

	if len(p.Pictures) > 0 {
		fmt.Println("\nPictures:")
		for _, pic := range p.Pictures {
			fmt.Printf("  - %s: %s\n", pic.Caption, pic.Path)
		}
	}

	spouses := idx.Spouses(p.ID)
	if len(spouses) > 0 {
		fmt.Println("\nSpouses:")
		for _, sid := range spouses {
			fmt.Printf("  - #%d %s\n", sid, idx.PersonName(sid))
		}
	}

	children := idx.ChildrenOf(p.ID)
	if len(children) > 0 {
		fmt.Println("\nChildren:")
		for _, cid := range children {
			fmt.Printf("  - #%d %s\n", cid, idx.PersonName(cid))
		}
	}

	parents := idx.Parents(p.ID)
	if len(parents) > 0 {
		fmt.Println("\nParents:")
		for _, pid := range parents {
			fmt.Printf("  - #%d %s\n", pid, idx.PersonName(pid))
		}
	}

	siblings := idx.Siblings(p.ID)
	if len(siblings) > 0 {
		fmt.Println("\nSiblings:")
		for _, sid := range siblings {
			fmt.Printf("  - #%d %s\n", sid, idx.PersonName(sid))
		}
	}

	return nil
}

// --- couples ---

func cmdCouples(ff *model.FamilyFile, idx *Index, asJSON bool) error {
	var couples []model.Family
	for _, f := range ff.Families {
		if f.Partner1 == 0 && f.Partner2 == 0 {
			continue
		}
		couples = append(couples, f)
	}

	if asJSON {
		return printJSON(couples)
	}

	for _, f := range couples {
		name1 := idx.PersonName(f.Partner1)
		name2 := idx.PersonName(f.Partner2)
		if f.Partner2 == 0 {
			name2 = "(unknown)"
		}
		if f.Partner1 == 0 {
			name1 = "(unknown)"
		}
		fmt.Printf("#%-6d %s & %s", f.ID, name1, name2)
		if len(f.Children) > 0 {
			fmt.Printf("  [%d children]", len(f.Children))
		}
		fmt.Println()
	}
	return nil
}

// --- places ---

func cmdPlaces(ff *model.FamilyFile, asJSON bool) error {
	if asJSON {
		return printJSON(ff.Places)
	}
	for _, p := range ff.Places {
		fmt.Printf("#%-6d %s\n", p.ID, p.Name)
	}
	return nil
}

// --- events ---

func cmdEvents(ff *model.FamilyFile, asJSON bool) error {
	if asJSON {
		return printJSON(ff.EventDefinitions)
	}
	for _, e := range ff.EventDefinitions {
		gedcom := ""
		if e.GEDCOMCode != "" {
			gedcom = fmt.Sprintf(" (%s)", e.GEDCOMCode)
		}
		fmt.Printf("#%-6d %s%s\n", e.ID, e.DisplayName, gedcom)
	}
	return nil
}

// --- search ---

func cmdSearch(ff *model.FamilyFile, query string, asJSON bool) error {
	q := strings.ToLower(query)
	var matches []model.Person
	for _, p := range ff.Persons {
		name := strings.ToLower(FormatName(&p))
		if strings.Contains(name, q) {
			matches = append(matches, p)
		}
	}

	if asJSON {
		return printJSON(matches)
	}

	for i := range matches {
		p := &matches[i]
		fmt.Printf("#%-6d %s  %s\n", p.ID, p.Sex, FormatName(p))
	}
	return nil
}

// --- ancestors ---

// treeEntry is an alias for index.TreeEntry.
type treeEntry = index.TreeEntry

func cmdAncestors(idx *Index, id uint32, maxGen int, asJSON bool) error {
	p, ok := idx.Persons[id]
	if !ok {
		return fmt.Errorf("person %d not found", id)
	}

	if asJSON {
		var entries []treeEntry
		visited := make(map[uint32]bool)
		collectAncestors(idx, id, 0, maxGen, visited, &entries)
		return printJSON(entries)
	}

	fmt.Printf("Ancestors of #%d %s:\n", p.ID, FormatName(p))
	visited := make(map[uint32]bool)
	walkAncestors(idx, id, 0, maxGen, visited, "")
	return nil
}

func collectAncestors(idx *Index, id uint32, gen, maxGen int, visited map[uint32]bool, out *[]treeEntry) {
	if gen > maxGen || visited[id] {
		return
	}
	visited[id] = true

	for _, pid := range idx.Parents(id) {
		p, ok := idx.Persons[pid]
		if !ok {
			continue
		}
		*out = append(*out, treeEntry{Generation: gen + 1, Person: p})
		collectAncestors(idx, pid, gen+1, maxGen, visited, out)
	}
}

func walkAncestors(idx *Index, id uint32, gen, maxGen int, visited map[uint32]bool, prefix string) {
	if gen > maxGen || visited[id] {
		return
	}
	visited[id] = true

	for _, pid := range idx.Parents(id) {
		p, ok := idx.Persons[pid]
		if !ok {
			continue
		}
		fmt.Printf("%sGen %d: #%d %s (%s)\n", prefix, gen+1, p.ID, FormatName(p), p.Sex)
		walkAncestors(idx, pid, gen+1, maxGen, visited, prefix+"  ")
	}
}

// --- descendants ---

func cmdDescendants(idx *Index, id uint32, maxGen int, asJSON bool) error {
	p, ok := idx.Persons[id]
	if !ok {
		return fmt.Errorf("person %d not found", id)
	}

	if asJSON {
		var entries []treeEntry
		visited := make(map[uint32]bool)
		collectDescendants(idx, id, 0, maxGen, visited, &entries)
		return printJSON(entries)
	}

	fmt.Printf("Descendants of #%d %s:\n", p.ID, FormatName(p))
	visited := make(map[uint32]bool)
	walkDescendants(idx, id, 0, maxGen, visited, "")
	return nil
}

func collectDescendants(idx *Index, id uint32, gen, maxGen int, visited map[uint32]bool, out *[]treeEntry) {
	if gen > maxGen || visited[id] {
		return
	}
	visited[id] = true

	for _, cid := range idx.ChildrenOf(id) {
		c, ok := idx.Persons[cid]
		if !ok {
			continue
		}
		*out = append(*out, treeEntry{Generation: gen + 1, Person: c})
		collectDescendants(idx, cid, gen+1, maxGen, visited, out)
	}
}

func walkDescendants(idx *Index, id uint32, gen, maxGen int, visited map[uint32]bool, prefix string) {
	if gen > maxGen || visited[id] {
		return
	}
	visited[id] = true

	for _, cid := range idx.ChildrenOf(id) {
		c, ok := idx.Persons[cid]
		if !ok {
			continue
		}
		fmt.Printf("%sGen %d: #%d %s (%s)\n", prefix, gen+1, c.ID, FormatName(c), c.Sex)
		walkDescendants(idx, cid, gen+1, maxGen, visited, prefix+"  ")
	}
}

// --- treetops ---

func cmdTreetops(idx *Index, id uint32, asJSON bool) error {
	p, ok := idx.Persons[id]
	if !ok {
		return fmt.Errorf("person %d not found", id)
	}

	visited := make(map[uint32]bool)
	var treetopIDs []uint32
	findTreetops(idx, id, visited, &treetopIDs)

	if asJSON {
		var persons []*model.Person
		for _, tid := range treetopIDs {
			if tp, ok := idx.Persons[tid]; ok {
				persons = append(persons, tp)
			}
		}
		return printJSON(persons)
	}

	fmt.Printf("Treetops (terminal ancestors) of #%d %s:\n", p.ID, FormatName(p))
	for _, tid := range treetopIDs {
		fmt.Printf("  #%d %s\n", tid, idx.PersonName(tid))
	}
	fmt.Printf("\nTotal treetops: %d\n", len(treetopIDs))
	return nil
}

func findTreetops(idx *Index, id uint32, visited map[uint32]bool, treetops *[]uint32) {
	if visited[id] {
		return
	}
	visited[id] = true

	parents := idx.Parents(id)
	if len(parents) == 0 {
		*treetops = append(*treetops, id)
		return
	}
	for _, pid := range parents {
		findTreetops(idx, pid, visited, treetops)
	}
}

// --- summary ---

func cmdSummary(idx *Index, id uint32, asJSON bool) error {
	p, ok := idx.Persons[id]
	if !ok {
		return fmt.Errorf("person %d not found", id)
	}

	spouses := idx.Spouses(p.ID)
	siblings := idx.Siblings(p.ID)

	ancestorVisited := make(map[uint32]bool)
	countAncestors(idx, p.ID, ancestorVisited)
	delete(ancestorVisited, p.ID)
	ancestorCount := len(ancestorVisited)

	descendantVisited := make(map[uint32]bool)
	countDescendants(idx, p.ID, descendantVisited)
	delete(descendantVisited, p.ID)
	descendantCount := len(descendantVisited)

	treetopVisited := make(map[uint32]bool)
	var treetops []uint32
	findTreetops(idx, p.ID, treetopVisited, &treetops)

	surnameCounts := make(map[string]int)
	for aid := range ancestorVisited {
		a, ok := idx.Persons[aid]
		if ok && a.Surname != "" {
			surnameCounts[a.Surname]++
		}
	}

	if asJSON {
		return printJSON(map[string]interface{}{
			"person":      FormatName(p),
			"id":          p.ID,
			"spouses":     len(spouses),
			"siblings":    len(siblings),
			"ancestors":   ancestorCount,
			"descendants": descendantCount,
			"treetops":    len(treetops),
			"surnames":    surnameCounts,
		})
	}

	fmt.Printf("Summary for #%d %s\n", p.ID, FormatName(p))
	fmt.Printf("  Spouses:      %d\n", len(spouses))
	fmt.Printf("  Siblings:     %d\n", len(siblings))
	fmt.Printf("  Ancestors:    %d\n", ancestorCount)
	fmt.Printf("  Descendants:  %d\n", descendantCount)
	fmt.Printf("  Treetops:     %d\n", len(treetops))

	if len(surnameCounts) > 0 {
		fmt.Println("\n  Ancestor surnames:")
		type kv struct {
			k string
			v int
		}
		var sorted []kv
		for k, v := range surnameCounts {
			sorted = append(sorted, kv{k, v})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].v > sorted[j].v
		})
		for _, s := range sorted {
			fmt.Printf("    %-20s %d\n", s.k, s.v)
		}
	}
	return nil
}

func countAncestors(idx *Index, id uint32, visited map[uint32]bool) {
	if visited[id] {
		return
	}
	visited[id] = true
	for _, pid := range idx.Parents(id) {
		countAncestors(idx, pid, visited)
	}
}

func countDescendants(idx *Index, id uint32, visited map[uint32]bool) {
	if visited[id] {
		return
	}
	visited[id] = true
	for _, cid := range idx.ChildrenOf(id) {
		countDescendants(idx, cid, visited)
	}
}
