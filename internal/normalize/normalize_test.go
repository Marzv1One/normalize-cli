package normalize

import (
	"bytes"
	"testing"
)

func rstr(r rune) string { return string(r) }

func TestBytes(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		changed bool
	}{
		{"decomposed e acute", "cafe" + rstr(0x0301), "caf" + rstr(0x00e9), true},
		{"decomposed a ring", "a" + rstr(0x030a), rstr(0x00e5), true},
		{"precomposed unchanged", "caf" + rstr(0x00e9), "caf" + rstr(0x00e9), false},
		{"hangul composition", rstr(0x1100) + rstr(0x1161), rstr(0xac00), true},
		{"canonical reorder then compose", "s" + rstr(0x0307) + rstr(0x0323), rstr(0x1e69), true},
		{"reorder without composition", "q" + rstr(0x0307) + rstr(0x0323), "q" + rstr(0x0323) + rstr(0x0307), true},
		{"ascii unchanged", "hello, world\n", "hello, world\n", false},
		{"empty", "", "", false},
		{"crlf preserved", "a" + rstr(0x0301) + "\r\nb\r\n", rstr(0x00e1) + "\r\nb\r\n", true},
		{"bom preserved", rstr(0xfeff) + "ok", rstr(0xfeff) + "ok", false},
		{"capital with diaeresis", "A" + rstr(0x0308) + "rger", rstr(0x00c4) + "rger", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, changed, err := Bytes([]byte(tt.in))
			if err != nil {
				t.Fatalf("Bytes() error = %v", err)
			}
			if string(out) != tt.want {
				t.Errorf("Bytes() = %q, want %q", out, tt.want)
			}
			if changed != tt.changed {
				t.Errorf("changed = %v, want %v", changed, tt.changed)
			}
		})
	}
}

func TestBytesInvalidUTF8(t *testing.T) {
	for _, in := range [][]byte{
		{0xff, 0xfe},
		{0x80},
		{'o', 'k', 0xff},
		{0xc3, 0x28},
	} {
		if _, _, err := Bytes(in); err == nil {
			t.Errorf("Bytes(% x) = nil error, want error", in)
		}
	}
}

func TestBytesIdempotent(t *testing.T) {
	in := []byte("cafe" + rstr(0x0301) + " " + rstr(0x1100) + rstr(0x1161) + " q" + rstr(0x0307) + rstr(0x0323))
	once, changed, err := Bytes(in)
	if err != nil {
		t.Fatalf("first pass: %v", err)
	}
	if !changed {
		t.Fatal("first pass: changed = false, want true")
	}
	twice, changed, err := Bytes(once)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if changed {
		t.Error("second pass: changed = true, want false (NFC must be idempotent)")
	}
	if !bytes.Equal(once, twice) {
		t.Errorf("second pass = %q, want %q", twice, once)
	}
}
