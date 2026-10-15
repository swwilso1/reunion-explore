package notes

import "testing"

// cocoaRTF mirrors the RTF Reunion 10 stores for notes.
const cocoaRTF = "{\\rtf1\\mac\\ansicpg10000\\cocoartf824\\cocoasubrtf420\n" +
	"{\\fonttbl\\f0\\fnil\\fcharset77 LucidaGrande;}\n" +
	"{\\colortbl;\\red255\\green255\\blue255;}\n" +
	"\\pard\\tx560\\tx1120\\ql\\qnatural\\pardirnatural\n\n" +
	"\\f0\\fs22 \\cf0 Also known as the \\'d4Crathorne Cemetery\\'d5\\\n" +
	"Second line \\{braced\\} caf\\u233 e}"

func TestRTFToText(t *testing.T) {
	want := "Also known as the ‘Crathorne Cemetery’\nSecond line {braced} café"
	if got := RTFToText(cocoaRTF); got != want {
		t.Errorf("RTFToText() = %q, want %q", got, want)
	}
}

func TestRTFToText_SkipsStarDestination(t *testing.T) {
	if got := RTFToText(`{\rtf1{\*\generator Foo;}Hello\par World}`); got != "Hello\nWorld" {
		t.Errorf("RTFToText() = %q, want %q", got, "Hello\nWorld")
	}
}

func TestRTFEnd(t *testing.T) {
	data := []byte(cocoaRTF + "p\x00\x09<\xa6N")
	if got := RTFEnd(data); got != len(cocoaRTF) {
		t.Errorf("RTFEnd() = %d, want %d", got, len(cocoaRTF))
	}
	if got := RTFEnd([]byte(`{\rtf1 unterminated`)); got != -1 {
		t.Errorf("RTFEnd(unterminated) = %d, want -1", got)
	}
}
