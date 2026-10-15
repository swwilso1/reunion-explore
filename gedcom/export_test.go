package gedcom_test

import (
	"regexp"
	"strings"
	"testing"

	reunion "github.com/kedoco/reunion-explore"
	"github.com/kedoco/reunion-explore/gedcom"
	_ "github.com/kedoco/reunion-explore/parser"
)

var linePattern = regexp.MustCompile(`^(\d+)(?: (@[^@ ]+@))? ([A-Z_][A-Z0-9_]*)(?: (.*))?$`)

func TestWrite_SampleFamily14(t *testing.T) {
	ff, err := reunion.Open("../testdata/Sample Family 14.familyfile14", nil)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	var b strings.Builder
	if err := gedcom.Write(&b, ff, gedcom.Options{ProgramVersion: "test"}); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	out := b.String()
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")

	// Structure: syntax, level steps, line length, resolvable pointers.
	defined := map[string]bool{}
	var pointers []string
	records := map[string]int{}
	prev := -1
	for i, l := range lines {
		m := linePattern.FindStringSubmatch(l)
		if m == nil {
			t.Fatalf("line %d: bad syntax %q", i+1, l)
		}
		level := int(m[1][0] - '0')
		if level > prev+1 {
			t.Errorf("line %d: level jumps from %d to %d", i+1, prev, level)
		}
		prev = level
		if len(l) > 255 {
			t.Errorf("line %d: %d bytes", i+1, len(l))
		}
		if m[2] != "" {
			defined[m[2]] = true
		}
		if level == 0 {
			records[m[3]]++
		}
		if regexp.MustCompile(`^@[^@]+@$`).MatchString(m[4]) {
			pointers = append(pointers, m[4])
		}
	}
	for _, p := range pointers {
		if !defined[p] {
			t.Errorf("pointer %s has no record", p)
		}
	}
	if lines[0] != "0 HEAD" || lines[len(lines)-1] != "0 TRLR" {
		t.Error("file must start with HEAD and end with TRLR")
	}
	for tag, want := range map[string]int{"INDI": 49, "FAM": 35, "SOUR": 25, "NOTE": 28} {
		if records[tag] != want {
			t.Errorf("%s records = %d, want %d", tag, records[tag], want)
		}
	}

	// Content spot checks.
	for _, want := range []string{
		"0 @F8@ FAM\n1 HUSB @I4@\n1 WIFE @I17@\n",             // JFK + Jacqueline
		"1 MARR\n2 DATE 12 SEP 1953\n2 PLAC Newport, RI\n",    // their marriage
		"1 NAME John Fitzgerald /KENNEDY/\n",                  // name with surname slashes
		"1 BIRT\n2 DATE 6 SEP 1888\n2 PLAC East Boston, MA\n", // Joseph P. Kennedy
		"2 DATE ABT 1823\n",                                   // approximate date
		"1 RELI Roman Catholic\n",                             // fact with value
		"1 AUTH Doris Kearns Goodwin\n",                       // source field
		"2 FILE /Users/kevin/Pictures/Reunion Pictures/Sample Pictures/kennedy, joseph p banker.jpg\n3 FORM jpg\n", // bookmark picture
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if n := strings.Count(blockOf(out, "0 @F1@ FAM"), "1 CHIL "); n != 9 {
		t.Errorf("family 1 has %d CHIL lines, want 9", n)
	}
}

func TestWrite_NoPictures(t *testing.T) {
	ff, err := reunion.Open("../testdata/Sample Family 14.familyfile14", nil)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	var b strings.Builder
	if err := gedcom.Write(&b, ff, gedcom.Options{NoPictures: true}); err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if strings.Contains(b.String(), "OBJE") {
		t.Error("output contains OBJE with NoPictures set")
	}
	if !strings.Contains(b.String(), "1 MARR\n2 DATE 12 SEP 1953\n") {
		t.Error("output missing other data with NoPictures set")
	}
}

// blockOf returns the record starting with header, up to the next record.
func blockOf(out, header string) string {
	i := strings.Index(out, header+"\n")
	if i < 0 {
		return ""
	}
	rest := out[i+len(header)+1:]
	if j := strings.Index(rest, "\n0 "); j >= 0 {
		return rest[:j]
	}
	return rest
}
