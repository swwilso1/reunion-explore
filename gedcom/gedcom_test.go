package gedcom

import (
	"strings"
	"testing"
)

func TestGedcomDate(t *testing.T) {
	tests := map[string]string{
		"12 Sep 1953":      "12 SEP 1953",
		"about 1850":       "ABT 1850",
		"after 1 Jan 1900": "AFT 1 JAN 1900",
		"Jun 2000":         "JUN 2000",
		"1939":             "1939",
		"":                 "",
	}
	for in, want := range tests {
		if got := gedcomDate(in); got != want {
			t.Errorf("gedcomDate(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSplitChunks(t *testing.T) {
	s := strings.Repeat("abc def ", 60) + "é"
	chunks := splitChunks(s, 50)
	if strings.Join(chunks, "") != s {
		t.Fatal("chunks do not rejoin to the original")
	}
	for i, c := range chunks {
		if len(c) > 50 {
			t.Errorf("chunk %d has %d bytes", i, len(c))
		}
		if i > 0 && (strings.HasPrefix(c, " ") || strings.HasSuffix(chunks[i-1], " ")) {
			t.Errorf("chunk %d split at a space", i)
		}
	}
}

func TestText_ContAndEscaping(t *testing.T) {
	var b strings.Builder
	e := newExporter(&b, nil, Options{})
	e.text(1, "NOTE", "Email me @ home\nSecond\x07 line")
	e.out.Flush()
	want := "1 NOTE Email me @@ home\n2 CONT Second line\n"
	if b.String() != want {
		t.Errorf("text() wrote %q, want %q", b.String(), want)
	}
}
