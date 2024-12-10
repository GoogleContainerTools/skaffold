// Package kvfile provides utilities to parse line-delimited key/value files
// such as used for label-files and env-files.
//
// # File format
//
// key/value files use the following syntax:
//
//   - File must be valid UTF-8.
//   - BOM headers are removed.
//   - Leading whitespace is removed for each line.
//   - Lines starting with "#" are ignored.
//   - Empty lines are ignored.
//   - Key/Value pairs are provided as "KEY[=<VALUE>]".
//   - Maximum line-length is limited to [bufio.MaxScanTokenSize].
//
// # Interpolation, substitution, and escaping
//
// Both keys and values are used as-is; no interpolation, substitution or
// escaping is supported, and quotes are considered part of the key or value.
// Whitespace in values (including leading and trailing) is preserved. Given
// that the file format is line-delimited, neither key, nor value, can contain
// newlines.
//
// # Key/Value pairs
//
// Key/Value pairs take the following format:
//
//	KEY[=<VALUE>]
//
// KEY is required and may not contain whitespaces or NUL characters. Any
// other character (except for the "=" delimiter) are accepted, but  it is
// recommended to use a subset of the POSIX portable character set, as
// outlined in [Environment Variables].
//
// VALUE is optional, but may be empty. If no value is provided (i.e., no
// equal sign ("=") is present), the KEY is omitted in the result, but some
// functions accept a lookup-function to provide a default value for the
// given key.
//
// [Environment Variables]: https://pubs.opengroup.org/onlinepubs/7908799/xbd/envvar.html
package kvfile

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Parse parses a line-delimited key/value pairs separated by equal sign.
// It accepts a lookupFn to lookup default values for keys that do not define
// a value. An error is produced if parsing failed, the content contains invalid
// UTF-8 characters, or a key contains whitespaces.
func Parse(filename string, lookupFn func(key string) (value string, found bool)) ([]string, error) {
	fh, err := os.Open(filename)
	if err != nil {
		return []string{}, err
	}
	out, err := parseKeyValueFile(fh, lookupFn)
	_ = fh.Close()
	if err != nil {
		return []string{}, fmt.Errorf("invalid env file (%s): %v", filename, err)
	}
	return out, nil
}

// ParseFromReader parses a line-delimited key/value pairs separated by equal sign.
// It accepts a lookupFn to lookup default values for keys that do not define
// a value. An error is produced if parsing failed, the content contains invalid
// UTF-8 characters, or a key contains whitespaces.
func ParseFromReader(r io.Reader, lookupFn func(key string) (value string, found bool)) ([]string, error) {
	return parseKeyValueFile(r, lookupFn)
}

const whiteSpaces = " \t"

func parseKeyValueFile(r io.Reader, lookupFn func(string) (string, bool)) ([]string, error) {
	lines := []string{}
	scanner := bufio.NewScanner(r)
	utf8bom := []byte{0xEF, 0xBB, 0xBF}
	for currentLine := 1; scanner.Scan(); currentLine++ {
		scannedBytes := scanner.Bytes()
		if !utf8.Valid(scannedBytes) {
			return []string{}, fmt.Errorf("invalid utf8 bytes at line %d: %v", currentLine, scannedBytes)
		}
		// We trim UTF8 BOM
		if currentLine == 1 {
			scannedBytes = bytes.TrimPrefix(scannedBytes, utf8bom)
		}
		// trim the line from all leading whitespace first. trailing whitespace
		// is part of the value, and is kept unmodified.
		line := strings.TrimLeftFunc(string(scannedBytes), unicode.IsSpace)

		if len(line) == 0 || line[0] == '#' {
			// skip empty lines and comments (lines starting with '#')
			continue
		}

		key, _, hasValue := strings.Cut(line, "=")
		if len(key) == 0 {
			return []string{}, fmt.Errorf("no variable name on line '%s'", line)
		}

		// leading whitespace was already removed from the line, but
		// variables are not allowed to contain whitespace or have
		// trailing whitespace.
		if strings.ContainsAny(key, whiteSpaces) {
			return []string{}, fmt.Errorf("variable '%s' contains whitespaces", key)
		}

		if hasValue {
			// key/value pair is valid and has a value; add the line as-is.
			lines = append(lines, line)
			continue
		}

		if lookupFn != nil {
			// No value given; try to look up the value. The value may be
			// empty but if no value is found, the key is omitted.
			if value, found := lookupFn(line); found {
				lines = append(lines, key+"="+value)
			}
		}
	}
	return lines, scanner.Err()
}
