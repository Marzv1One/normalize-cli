package normalize

import (
	"bytes"
	"errors"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

func Bytes(b []byte) (out []byte, changed bool, err error) {
	if !utf8.Valid(b) {
		return nil, false, errors.New("input is not valid UTF-8")
	}
	out = norm.NFC.Bytes(b)
	return out, !bytes.Equal(out, b), nil
}
