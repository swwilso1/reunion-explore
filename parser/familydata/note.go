package familydata

import (
	"bytes"

	reunion "github.com/kedoco/reunion-explore"
	"github.com/kedoco/reunion-explore/model"
	"github.com/kedoco/reunion-explore/parser/notes"
)

// ParseNote parses a 0x2104 inline note record from familydata.
func ParseNote(rec RawRecord, ec *reunion.ErrorCollector) (*model.Note, error) {
	n := &model.Note{
		ID:     rec.ID,
		SeqNum: rec.SeqNum,
	}

	if len(rec.Data) > 8 {
		data := rec.Data[8:]
		text := extractNoteText(data)
		n.RawText = text
		if text != "" {
			n.Markup = notes.ParseMarkup(text)
			n.DisplayText = model.PlainText(n.Markup)
		}
	}

	return n, nil
}

// noteTextMarker precedes the note text: "talf" + u32 + u32, then the text.
var noteTextMarker = []byte("talf")

// openTagBytes is the UTF-8 encoding of « (U+00AB).
var openTagBytes = []byte{0xC2, 0xAB}

func extractNoteText(data []byte) string {
	// Skip past the "talf" marker and its 8-byte trailer when present.
	// Reunion 13 stores a nonzero value after the marker, which would
	// otherwise be mistaken for the start of the text.
	if idx := bytes.Index(data[:min(len(data), 32)], noteTextMarker); idx >= 0 && idx+12 <= len(data) {
		data = data[idx+12:]
	}

	// Inline notes have a binary preamble followed by text content.
	// The text typically starts with «ff=1» markup.
	// Look for the first occurrence of the « character (0xC2 0xAB).
	idx := bytes.Index(data, openTagBytes)
	if idx >= 0 {
		return trimNoteTrailer(data[idx:])
	}

	// Fallback: find the first run of printable text (at least 5 chars)
	for i := 0; i < len(data); i++ {
		if data[i] >= 0x20 && data[i] < 0x7F {
			printable := 0
			for j := i; j < len(data) && j < i+10; j++ {
				if data[j] >= 0x20 && data[j] < 0x7F {
					printable++
				}
			}
			if printable >= 5 {
				return trimNoteTrailer(data[i:])
			}
		}
	}

	return ""
}

// trimNoteTrailer removes the binary trailer that follows note text.
// Note records are null-padded after the text and end with a small
// binary footer (typically NN 00 04 21). We find the last closing
// markup tag «/...» or the last substantial text, then trim from
// the first null byte after it.
func trimNoteTrailer(data []byte) string {
	// Find the last «/ sequence (closing markup tag)
	lastClose := bytes.LastIndex(data, []byte{0xC2, 0xAB, 0x2F})
	if lastClose >= 0 {
		// Find the » that closes this tag
		end := bytes.Index(data[lastClose:], []byte{0xC2, 0xBB})
		if end >= 0 {
			return string(data[:lastClose+end+2])
		}
		// Bare «/ at end (no closing ») — just include it
		return string(data[:lastClose+3])
	}

	// No closing markup tag — plain text note.
	// Text is followed by null padding + binary footer (e.g. NN 00 04 21).
	// Null bytes never appear in valid text, so trim at the first one.
	if idx := bytes.IndexByte(data, 0); idx >= 0 {
		return string(data[:idx])
	}
	return string(data)
}
