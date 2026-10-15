package familydata

import (
	"bytes"
	"encoding/binary"
	"strings"

	"github.com/kedoco/reunion-explore/model"
)

// Picture fields link a person to an image file. Tags start at 0x0258,
// one per picture slot.
const (
	TagPictureFirst uint16 = 0x0258
	TagPictureLast  uint16 = 0x02BB
)

func isPictureTag(tag uint16) bool {
	return tag >= TagPictureFirst && tag <= TagPictureLast
}

// pictureTextOffset is where the caption starts within a picture field.
const pictureTextOffset = 80

// ParsePicture decodes a picture field. Layout:
//
//	u16 size
//	u16 caption length << 5 (low 5 bits and high bits are flags)
//	u16 file data length << 4 (low 4 bits are flags)
//	... crop rectangle and other fixed data ...
//	@80: caption, then file data
//
// The file data is a plain path (Reunion 10 uses "~/..." paths, 13/14
// a bare filename), a macOS bookmark ("book"), or a classic Mac alias
// record (Reunion 10).
func ParsePicture(tag uint16, data []byte) (model.Picture, bool) {
	if len(data) < pictureTextOffset {
		return model.Picture{}, false
	}
	capLen := int(binary.LittleEndian.Uint16(data[2:4])>>5) & 0xFF
	fileLen := int(binary.LittleEndian.Uint16(data[4:6]) >> 4)
	capEnd := pictureTextOffset + capLen
	if capEnd+fileLen > len(data) {
		return model.Picture{}, false
	}

	pic := model.Picture{
		Slot:    int(tag-TagPictureFirst) + 1,
		Caption: string(data[pictureTextOffset:capEnd]),
	}

	file := data[capEnd : capEnd+fileLen]
	switch {
	case bytes.HasPrefix(file, []byte("book")):
		pic.Path = bookmarkPath(file)
	case isAliasRecord(file):
		pic.Path = aliasPath(file)
	default:
		pic.Path = string(bytes.TrimRight(file, "\x00"))
	}
	return pic, true
}

// bookmarkPath extracts the file path from macOS bookmark data
// (CFURLCreateBookmarkData format, little-endian).
func bookmarkPath(b []byte) string {
	const keyPath = 0x1004 // array of path component strings
	if len(b) < 16 {
		return ""
	}
	hdr := int(binary.LittleEndian.Uint32(b[12:16]))
	if hdr >= len(b) {
		return ""
	}
	data := b[hdr:]
	u32 := func(off int) (uint32, bool) {
		if off < 0 || off+4 > len(data) {
			return 0, false
		}
		return binary.LittleEndian.Uint32(data[off:]), true
	}
	// item returns the type and payload of the item at off.
	item := func(off int) (uint32, []byte) {
		ln, ok1 := u32(off)
		ty, ok2 := u32(off + 4)
		if !ok1 || !ok2 || off+8+int(ln) > len(data) {
			return 0, nil
		}
		return ty, data[off+8 : off+8+int(ln)]
	}

	toc, _ := u32(0)
	for seen := 0; toc != 0 && seen < 16; seen++ {
		count, ok := u32(int(toc) + 16)
		if !ok {
			return ""
		}
		for e := 0; e < int(count); e++ {
			entry := int(toc) + 20 + 12*e
			key, _ := u32(entry)
			off, _ := u32(entry + 4)
			if key != keyPath {
				continue
			}
			ty, arr := item(int(off))
			if ty != 0x0601 { // array
				return ""
			}
			var parts []string
			for k := 0; k+4 <= len(arr); k += 4 {
				if sty, s := item(int(binary.LittleEndian.Uint32(arr[k:]))); sty == 0x0101 { // UTF-8 string
					parts = append(parts, string(s))
				}
			}
			return "/" + strings.Join(parts, "/")
		}
		toc, _ = u32(int(toc) + 12)
	}
	return ""
}

// isAliasRecord reports whether b is a classic Mac OS alias record
// (version 2, big-endian), whose header repeats its own length.
func isAliasRecord(b []byte) bool {
	return len(b) >= 150 &&
		int(binary.BigEndian.Uint16(b[4:6])) == len(b) &&
		binary.BigEndian.Uint16(b[6:8]) == 2
}

// aliasPath extracts the POSIX path (tag 0x12) from an alias record's
// tagged extension data, falling back to the file name in the header.
func aliasPath(b []byte) string {
	const (
		tagPOSIXPath = 0x12
		tagEnd       = 0xFFFF
	)
	for pos := 150; pos+4 <= len(b); {
		tag := binary.BigEndian.Uint16(b[pos:])
		ln := int(binary.BigEndian.Uint16(b[pos+2:]))
		if tag == tagEnd || pos+4+ln > len(b) {
			break
		}
		if tag == tagPOSIXPath {
			return "/" + strings.TrimLeft(string(b[pos+4:pos+4+ln]), "/")
		}
		pos += 4 + ln + ln%2 // data is padded to an even length
	}
	// File name is a Str63 at offset 50.
	if n := int(b[50]); 51+n <= len(b) && n <= 63 {
		return string(b[51 : 51+n])
	}
	return ""
}
