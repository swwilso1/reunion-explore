package familydata

import (
	"encoding/binary"
	"testing"
)

// pictureField builds picture field data with the given caption and file data.
func pictureField(caption string, file []byte) []byte {
	data := make([]byte, pictureTextOffset)
	binary.LittleEndian.PutUint16(data[0:], uint16(pictureTextOffset+len(caption)+len(file)))
	binary.LittleEndian.PutUint16(data[2:], uint16(len(caption))<<5|0x08)
	binary.LittleEndian.PutUint16(data[4:], uint16(len(file))<<4)
	data = append(data, caption...)
	return append(data, file...)
}

func TestParsePicture_Path(t *testing.T) {
	// Reunion 10 stores a "~/" path directly after the caption.
	data := pictureField("s41221cb109829_7_1", []byte("~/Pictures/s41221cb109829_7_1.jpg"))
	pic, ok := ParsePicture(0x0258, data)
	if !ok {
		t.Fatal("ParsePicture() failed")
	}
	if pic.Slot != 1 || pic.Caption != "s41221cb109829_7_1" || pic.Path != "~/Pictures/s41221cb109829_7_1.jpg" {
		t.Errorf("ParsePicture() = %+v", pic)
	}
}

func TestParsePicture_Alias(t *testing.T) {
	// Classic Mac alias record (big-endian) with a POSIX path extension.
	posix := "Users/someone/Pictures/photo.jpg"
	alias := make([]byte, 150)
	binary.BigEndian.PutUint16(alias[6:], 2) // version
	alias[50] = byte(len("photo.jpg"))
	copy(alias[51:], "photo.jpg")
	ext := binary.BigEndian.AppendUint16(nil, 0x12)
	ext = binary.BigEndian.AppendUint16(ext, uint16(len(posix)))
	ext = append(ext, posix...)
	if len(posix)%2 == 1 {
		ext = append(ext, 0)
	}
	ext = append(ext, 0xFF, 0xFF, 0, 0)
	alias = append(alias, ext...)
	binary.BigEndian.PutUint16(alias[4:], uint16(len(alias)))

	pic, ok := ParsePicture(0x0259, pictureField("Photo", alias))
	if !ok {
		t.Fatal("ParsePicture() failed")
	}
	if pic.Slot != 2 || pic.Caption != "Photo" || pic.Path != "/"+posix {
		t.Errorf("ParsePicture() = %+v", pic)
	}
}
