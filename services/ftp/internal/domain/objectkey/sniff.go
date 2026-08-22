package objectkey

import "errors"

var ErrTooLarge = errors.New("object too large")
var ErrBadMagic = errors.New("unsupported file type")

const MaxObjectBytes = 8 << 20
const MaxZipBytes = 32 << 20

func Sniff(data []byte) (contentType string, err error) {
	if len(data) < 12 {
		return "", ErrBadMagic
	}
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg", nil
	}
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png", nil
	}
	if string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp", nil
	}
	if string(data[0:4]) == "PK\x03\x04" {
		return "application/zip", nil
	}
	if len(data) > 0 && data[0] == '{' {
		return "application/json", nil
	}
	return "", ErrBadMagic
}
