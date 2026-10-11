# normalize-cli

Normalize a text file to Unicode Normalization Form C (NFC).

NFC composes decomposed character sequences (for example `e` + U+0301 becomes
the single precomposed `é`). It is the form recommended by the W3C and the one
most systems expect, which fixes "same-looking text, different bytes" issues
across platforms.

## Install

```
go install github.com/Marzv1One/normalize-cli@latest
```

## Usage

```
normalize-cli [flags] <file>
```

Flags may appear before or after the file argument. By default the normalized
content is written to stdout, leaving the input untouched.

| Flag     | Description                                                        |
|----------|--------------------------------------------------------------------|
| `-w`     | Write the result back to the input file, in place (atomic, preserves permissions). |
| `-o path`| Write the result to `path` instead of stdout.                      |
| `-check` | Write nothing. Exit 0 if the file is already NFC, 1 if it is not.  |
| `-q`     | Suppress informational messages.                                   |

`-w` and `-o` are mutually exclusive; `-check` cannot be combined with either.

## Exit codes

| Mode     | 0                  | 1                                   | 2            |
|----------|--------------------|-------------------------------------|--------------|
| normal   | success            | runtime error (I/O, invalid UTF-8)  | usage error  |
| `-check` | already normalized | not normalized                      | error        |

## Examples

```
# preview the normalized content
normalize-cli weird.txt

# fix a file in place
normalize-cli -w weird.txt

# CI gate: fail if any file still needs normalization
normalize-cli -check docs/*.txt
```

## Behavior notes

- Input must be valid UTF-8. Binary files are rejected with an error instead of
  being silently corrupted with U+FFFD replacement characters.
- With `-w`, an already-normalized file is left untouched (mtime preserved).
- Line endings and a leading UTF-8 BOM are preserved.
- In-place writes are atomic: the result is written to a temporary file in the
  same directory and renamed over the original.
