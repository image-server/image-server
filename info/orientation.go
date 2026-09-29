package info

import (
	"bufio"
	"encoding/binary"
	"io"
)

// jpegOrientation returns the EXIF orientation (1-8) of a JPEG stream, or 1
// when there is none or it cannot be read. Only the markers before the image
// data are scanned.
func jpegOrientation(r io.Reader) int {
	br := bufio.NewReader(r)
	var soi [2]byte
	if _, err := io.ReadFull(br, soi[:]); err != nil || soi[0] != 0xFF || soi[1] != 0xD8 {
		return 1
	}
	for {
		var marker [2]byte
		if _, err := io.ReadFull(br, marker[:]); err != nil || marker[0] != 0xFF {
			return 1
		}
		for marker[1] == 0xFF { // fill bytes
			b, err := br.ReadByte()
			if err != nil {
				return 1
			}
			marker[1] = b
		}
		// SOS or EOI: no EXIF before the image data.
		if marker[1] == 0xDA || marker[1] == 0xD9 {
			return 1
		}
		var size [2]byte
		if _, err := io.ReadFull(br, size[:]); err != nil {
			return 1
		}
		n := int(binary.BigEndian.Uint16(size[:])) - 2
		if n < 0 {
			return 1
		}
		if marker[1] != 0xE1 {
			if _, err := br.Discard(n); err != nil {
				return 1
			}
			continue
		}
		segment := make([]byte, n)
		if _, err := io.ReadFull(br, segment); err != nil {
			return 1
		}
		if len(segment) >= 6 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}
	}
}

// tiffOrientation reads tag 0x0112 from IFD0 of a TIFF header. Anything but
// a well-formed SHORT with count 1 and value 1-8 is treated as no orientation,
// matching libvips, so info and processed outputs agree.
func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return 1
	}
	// Bounds by subtraction: no int overflow on 32-bit builds.
	ifd := order.Uint32(tiff[4:8])
	if ifd < 8 || uint64(ifd) > uint64(len(tiff)-2) {
		return 1
	}
	entries := tiff[ifd:]
	count := int(order.Uint16(entries[:2]))
	entries = entries[2:]
	for i := 0; i < count && len(entries) >= 12; i++ {
		entry := entries[:12]
		entries = entries[12:]
		if order.Uint16(entry[0:2]) != 0x0112 {
			continue
		}
		const typeShort = 3
		if order.Uint16(entry[2:4]) != typeShort || order.Uint32(entry[4:8]) != 1 {
			return 1
		}
		// SHORT value, stored in the first two bytes of the value field.
		v := int(order.Uint16(entry[8:10]))
		if v < 1 || v > 8 {
			return 1
		}
		return v
	}
	return 1
}

// swapsAxes is true for the EXIF orientations that turn the image 90 degrees,
// so the displayed width is the stored height.
func swapsAxes(orientation int) bool {
	return orientation >= 5 && orientation <= 8
}
