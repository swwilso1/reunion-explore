package notes

import (
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"
)

// IsRTF reports whether text is an RTF document.
func IsRTF(text string) bool {
	return strings.HasPrefix(text, `{\rtf`)
}

// RTFEnd returns the length of the RTF document at the start of data,
// ending at the brace that closes the outermost group, or -1 if the
// document is unterminated.
func RTFEnd(data []byte) int {
	depth := 0
	for i := 0; i < len(data); i++ {
		switch data[i] {
		case '\\':
			i++ // skip escaped character (\{, \}, \\)
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return -1
}

// skipDestinations are RTF groups whose content is not document text.
var skipDestinations = map[string]bool{
	"fonttbl": true, "colortbl": true, "stylesheet": true, "info": true,
	"pict": true, "header": true, "footer": true, "listtable": true,
	"listoverridetable": true, "expandedcolortbl": true,
}

// rtfSymbols maps control words to the text they produce.
var rtfSymbols = map[string]string{
	"par": "\n", "line": "\n", "tab": "\t",
	"emdash": "—", "endash": "–", "bullet": "•",
	"lquote": "‘", "rquote": "’",
	"ldblquote": "“", "rdblquote": "”",
}

// RTFToText converts an RTF document to plain text. It handles the subset
// produced by Cocoa's RTF writer: paragraphs, tabs, \'hh escapes in the
// document's code page, \uN escapes, and skipped header destinations.
func RTFToText(rtf string) string {
	var out strings.Builder
	var pending []byte // \'hh bytes awaiting code-page decoding
	decoder := charmap.Macintosh.NewDecoder()

	flush := func() {
		if len(pending) > 0 {
			s, err := decoder.Bytes(pending)
			if err == nil {
				out.Write(s)
			}
			pending = pending[:0]
		}
	}

	// Per-group state: whether the group is skipped, and \ucN.
	type groupState struct {
		skip bool
		uc   int
	}
	stack := []groupState{{uc: 1}}
	cur := func() *groupState { return &stack[len(stack)-1] }
	skipChars := 0 // characters to drop after a \uN escape

	emit := func(s string) {
		if cur().skip {
			return
		}
		if skipChars > 0 {
			skipChars--
			return
		}
		flush()
		out.WriteString(s)
	}

	for i := 0; i < len(rtf); i++ {
		c := rtf[i]
		switch c {
		case '{':
			flush()
			stack = append(stack, *cur())
			// A destination group is identified by its first control word.
			if strings.HasPrefix(rtf[i+1:], `\*`) {
				cur().skip = true
			} else if word, _, _ := readControlWord(rtf, i+1); skipDestinations[word] {
				cur().skip = true
			}
		case '}':
			flush()
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case '\r', '\n':
			// Raw line breaks are not significant in RTF.
		case '\\':
			if i+1 >= len(rtf) {
				break
			}
			next := rtf[i+1]
			switch {
			case next == '\'' && i+3 < len(rtf):
				if b, err := strconv.ParseUint(rtf[i+2:i+4], 16, 8); err == nil && !cur().skip {
					if skipChars > 0 {
						skipChars--
					} else {
						pending = append(pending, byte(b))
					}
				}
				i += 3
			case next == '\n' || next == '\r':
				emit("\n") // Cocoa writes paragraph breaks as "\" + newline
				i++
			case next == '\\' || next == '{' || next == '}':
				emit(string(next))
				i++
			case next == '~':
				emit(" ")
				i++
			case next == '_':
				emit("‑")
				i++
			case isLetter(next):
				word, param, end := readControlWord(rtf, i)
				i = end - 1
				switch {
				case word == "u":
					if n, err := strconv.Atoi(param); err == nil {
						if n < 0 {
							n += 65536
						}
						emit(string(rune(n)))
						skipChars = cur().uc
					}
				case word == "uc":
					if n, err := strconv.Atoi(param); err == nil {
						cur().uc = n
					}
				case rtfSymbols[word] != "":
					emit(rtfSymbols[word])
				}
			default:
				i++ // other control symbols (\-, \|, \:) produce no text
			}
		default:
			if c >= 0x80 {
				// 8-bit text is in the document's code page.
				if !cur().skip {
					pending = append(pending, c)
				}
			} else {
				emit(string(c))
			}
		}
	}
	flush()

	return strings.TrimSpace(out.String())
}

// readControlWord reads a control word starting at the backslash at pos.
// It returns the word, its numeric parameter (if any), and the index just
// past the word, its parameter and an optional delimiting space.
func readControlWord(s string, pos int) (word, param string, end int) {
	if pos >= len(s) || s[pos] != '\\' {
		return "", "", pos
	}
	i := pos + 1
	for i < len(s) && isLetter(s[i]) {
		i++
	}
	word = s[pos+1 : i]
	p := i
	if i < len(s) && s[i] == '-' {
		i++
	}
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	param = s[p:i]
	if i < len(s) && s[i] == ' ' {
		i++
	}
	return word, param, i
}

func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
