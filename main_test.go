package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rstr(r rune) string { return string(r) }

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestRunStdout(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.txt")
	in := []byte("cafe" + rstr(0x0301) + "\n")
	writeFile(t, p, in)

	var stdout, stderr bytes.Buffer
	code := run([]string{p}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got, want := stdout.String(), "caf"+rstr(0x00e9)+"\n"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
	if got := readFile(t, p); !bytes.Equal(got, in) {
		t.Errorf("input modified: %q", got)
	}
}

func TestRunWriteInPlace(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.txt")
	writeFile(t, p, []byte("cafe"+rstr(0x0301)+"\n"))
	if err := os.Chmod(p, 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	wantPerm := info.Mode().Perm()

	var stdout, stderr bytes.Buffer
	code := run([]string{"-w", p}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got, want := string(readFile(t, p)), "caf"+rstr(0x00e9)+"\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	info, err = os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != wantPerm {
		t.Errorf("perm = %o, want %o", got, wantPerm)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("leftover files in dir: %v", names)
	}
	if !strings.Contains(stdout.String(), "normalized") {
		t.Errorf("stdout = %q, want a status message", stdout.String())
	}
}

func TestRunWriteInPlaceAlreadyNormalized(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.txt")
	writeFile(t, p, []byte("caf"+rstr(0x00e9)+"\n"))
	before, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"-w", p}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	after, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) {
		t.Error("file was rewritten although already normalized")
	}
	if got, want := string(readFile(t, p)), "caf"+rstr(0x00e9)+"\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if !strings.Contains(stdout.String(), "already normalized") {
		t.Errorf("stdout = %q", stdout.String())
	}
}

func TestRunOutputFile(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "in.txt")
	out := filepath.Join(dir, "out.txt")
	writeFile(t, in, []byte("cafe"+rstr(0x0301)))

	var stdout, stderr bytes.Buffer
	code := run([]string{"-o", out, in}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if got, want := string(readFile(t, out)), "caf"+rstr(0x00e9); got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
	if got, want := string(readFile(t, in)), "cafe"+rstr(0x0301); got != want {
		t.Errorf("input modified: %q, want %q", got, want)
	}
}

func TestRunCheck(t *testing.T) {
	dir := t.TempDir()
	needs := filepath.Join(dir, "needs.txt")
	done := filepath.Join(dir, "done.txt")
	writeFile(t, needs, []byte("cafe"+rstr(0x0301)))
	writeFile(t, done, []byte("caf"+rstr(0x00e9)))

	var stdout, stderr bytes.Buffer

	code := run([]string{"-check", needs}, &stdout, &stderr)
	if code != exitError {
		t.Errorf("not-normalized: exit = %d, want %d", code, exitError)
	}
	if !strings.Contains(stdout.String(), "not normalized") {
		t.Errorf("stdout = %q", stdout.String())
	}
	if got, want := string(readFile(t, needs)), "cafe"+rstr(0x0301); got != want {
		t.Errorf("file modified: %q, want %q", got, want)
	}

	stdout.Reset()
	code = run([]string{"-check", done}, &stdout, &stderr)
	if code != exitOK {
		t.Errorf("normalized: exit = %d, want %d", code, exitOK)
	}

	code = run([]string{"-check", filepath.Join(dir, "missing.txt")}, &stdout, &stderr)
	if code != exitUsage {
		t.Errorf("missing file: exit = %d, want %d", code, exitUsage)
	}
}

func TestRunInvalidUTF8(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bad.bin")
	bad := []byte{0xff, 0xfe, 0x00}
	writeFile(t, p, bad)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-w", p}, &stdout, &stderr)
	if code != exitError {
		t.Fatalf("exit = %d, want %d", code, exitError)
	}
	if !strings.Contains(stderr.String(), "UTF-8") {
		t.Errorf("stderr = %q", stderr.String())
	}
	if got := readFile(t, p); !bytes.Equal(got, bad) {
		t.Errorf("file modified: % x", got)
	}
}

func TestRunFlagsAfterFile(t *testing.T) {
	t.Run("output flag", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "in.txt")
		out := filepath.Join(dir, "out.txt")
		writeFile(t, p, []byte("cafe"+rstr(0x0301)))

		var stdout, stderr bytes.Buffer
		code := run([]string{p, "-o", out}, &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
		}
		if got, want := string(readFile(t, out)), "caf"+rstr(0x00e9); got != want {
			t.Errorf("output = %q, want %q", got, want)
		}
		if got, want := string(readFile(t, p)), "cafe"+rstr(0x0301); got != want {
			t.Errorf("input modified: %q, want %q", got, want)
		}
	})

	t.Run("write flag", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "in.txt")
		writeFile(t, p, []byte("cafe"+rstr(0x0301)))

		var stdout, stderr bytes.Buffer
		code := run([]string{p, "-w"}, &stdout, &stderr)
		if code != exitOK {
			t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
		}
		if got, want := string(readFile(t, p)), "caf"+rstr(0x00e9); got != want {
			t.Errorf("content = %q, want %q", got, want)
		}
	})

	t.Run("check flag", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "in.txt")
		writeFile(t, p, []byte("cafe"+rstr(0x0301)))

		var stdout, stderr bytes.Buffer
		code := run([]string{p, "-check"}, &stdout, &stderr)
		if code != exitError {
			t.Errorf("exit = %d, want %d", code, exitError)
		}
	})
}

func TestRunUsageErrors(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.txt")
	writeFile(t, p, []byte("x"))

	cases := [][]string{
		{},
		{p, p},
		{p, "-w", p},
		{"-o", p},
		{"-o", filepath.Join(dir, "o.txt")},
		{"-w", "-o", filepath.Join(dir, "o.txt"), p},
		{"-check", "-w", p},
		{"-check", "-o", filepath.Join(dir, "o.txt"), p},
		{"-bogus", p},
	}
	for _, args := range cases {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != exitUsage {
			t.Errorf("run(%v) exit = %d, want %d", args, code, exitUsage)
		}
	}
}

func TestRunMissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{filepath.Join(t.TempDir(), "nope.txt")}, &stdout, &stderr)
	if code != exitError {
		t.Errorf("exit = %d, want %d", code, exitError)
	}
}

func TestRunQuiet(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "in.txt")
	writeFile(t, p, []byte("caf"+rstr(0x00e9)))

	var stdout, stderr bytes.Buffer
	code := run([]string{"-q", "-w", p}, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit = %d", code)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want empty with -q", stdout.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-h"}, &stdout, &stderr); code != exitOK {
		t.Errorf("exit = %d, want %d", code, exitOK)
	}
}
