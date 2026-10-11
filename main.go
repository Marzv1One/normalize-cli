package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/Marzv1One/normalize-cli/internal/normalize"
)

const (
	exitOK    = 0
	exitError = 1
	exitUsage = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("normalize-cli", flag.ContinueOnError)
	fs.SetOutput(stderr)
	inPlace := fs.Bool("w", false, "write the result back to the input file, in place")
	output := fs.String("o", "", "write the result to `path` instead of stdout")
	check := fs.Bool("check", false, "write nothing; exit 0 if the file is already NFC, 1 if it is not")
	quiet := fs.Bool("q", false, "suppress informational messages")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: normalize-cli [flags] <file>")
		fs.PrintDefaults()
	}
	var files []string
	rest := args
	for len(rest) > 0 {
		if err := fs.Parse(rest); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return exitOK
			}
			return exitUsage
		}
		rest = fs.Args()
		if len(rest) == 0 {
			break
		}
		files = append(files, rest[0])
		rest = rest[1:]
	}
	if len(files) != 1 {
		if len(files) == 0 {
			fmt.Fprintln(stderr, "normalize-cli: missing input file")
		} else {
			fmt.Fprintf(stderr, "normalize-cli: expected exactly one input file, got %d\n", len(files))
		}
		fs.Usage()
		return exitUsage
	}
	if *inPlace && *output != "" {
		fmt.Fprintln(stderr, "normalize-cli: -w and -o are mutually exclusive")
		return exitUsage
	}
	if *check && (*inPlace || *output != "") {
		fmt.Fprintln(stderr, "normalize-cli: -check cannot be combined with -w or -o")
		return exitUsage
	}

	fail := func(err error) int {
		fmt.Fprintf(stderr, "normalize-cli: %v\n", err)
		if *check {
			return exitUsage
		}
		return exitError
	}

	path := files[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return fail(err)
	}
	out, changed, err := normalize.Bytes(data)
	if err != nil {
		return fail(fmt.Errorf("%s: %w", path, err))
	}

	if *check {
		if changed {
			if !*quiet {
				fmt.Fprintf(stdout, "%s: not normalized\n", path)
			}
			return exitError
		}
		if !*quiet {
			fmt.Fprintf(stdout, "%s: already normalized\n", path)
		}
		return exitOK
	}

	if *inPlace {
		if !changed {
			if !*quiet {
				fmt.Fprintf(stdout, "%s: already normalized\n", path)
			}
			return exitOK
		}
		if err := writeInPlace(path, out); err != nil {
			return fail(err)
		}
		if !*quiet {
			fmt.Fprintf(stdout, "%s: normalized\n", path)
		}
		return exitOK
	}

	if *output != "" {
		if err := os.WriteFile(*output, out, 0o644); err != nil {
			return fail(err)
		}
		if !*quiet {
			fmt.Fprintf(stdout, "%s: wrote %s\n", path, *output)
		}
		return exitOK
	}

	if _, err := stdout.Write(out); err != nil {
		return fail(err)
	}
	return exitOK
}

func writeInPlace(path string, data []byte) (err error) {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".normalize-cli-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			os.Remove(tmpName)
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	if err = os.Chmod(tmpName, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
