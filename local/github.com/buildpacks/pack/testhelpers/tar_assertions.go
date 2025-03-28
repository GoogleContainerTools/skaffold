package testhelpers

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"

	"github.com/buildpacks/pack/pkg/archive"
)

var gzipMagicHeader = []byte{'\x1f', '\x8b'}

type TarEntryAssertion func(t *testing.T, header *tar.Header, data []byte)

type TarEntriesAssertion func(t *testing.T, header1 *tar.Header, data1 []byte, header2 *tar.Header, data2 []byte)

func AssertOnTarEntry(t *testing.T, tarPath, entryPath string, assertFns ...TarEntryAssertion) {
	t.Helper()

	tarFile, err := os.Open(filepath.Clean(tarPath))
	AssertNil(t, err)
	defer tarFile.Close()

	header, data, err := readTarFileEntry(tarFile, entryPath)
	AssertNil(t, err)

	for _, fn := range assertFns {
		fn(t, header, data)
	}
}

func AssertOnNestedTar(nestedEntryPath string, assertions ...TarEntryAssertion) TarEntryAssertion {
	return func(t *testing.T, _ *tar.Header, data []byte) {
		t.Helper()

		header, data, err := readTarFileEntry(bytes.NewReader(data), nestedEntryPath)
		AssertNil(t, err)

		for _, assertion := range assertions {
			assertion(t, header, data)
		}
	}
}

func AssertOnTarEntries(t *testing.T, tarPath string, entryPath1, entryPath2 string, assertFns ...TarEntriesAssertion) {
	t.Helper()

	tarFile, err := os.Open(filepath.Clean(tarPath))
	AssertNil(t, err)
	defer tarFile.Close()

	header1, data1, err := readTarFileEntry(tarFile, entryPath1)
	AssertNil(t, err)

	_, err = tarFile.Seek(0, io.SeekStart)
	AssertNil(t, err)

	header2, data2, err := readTarFileEntry(tarFile, entryPath2)
	AssertNil(t, err)

	for _, fn := range assertFns {
		fn(t, header1, data1, header2, data2)
	}
}

func readTarFileEntry(reader io.Reader, entryPath string) (*tar.Header, []byte, error) {
	var (
		gzipReader *gzip.Reader
		err        error
	)

	headerBytes, isGzipped, err := isGzipped(reader)
	if err != nil {
		return nil, nil, errors.Wrap(err, "checking if reader")
	}
	reader = io.MultiReader(bytes.NewReader(headerBytes), reader)

	if isGzipped {
		gzipReader, err = gzip.NewReader(reader)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create gzip reader")
		}
		reader = gzipReader
		defer gzipReader.Close()
	}

	return archive.ReadTarEntry(reader, entryPath)
}

func isGzipped(reader io.Reader) (headerBytes []byte, isGzipped bool, err error) {
	magicHeader := make([]byte, 2)
	n, err := reader.Read(magicHeader)
	if n == 0 && err == io.EOF {
		return magicHeader, false, nil
	}
	if err != nil {
		return magicHeader, false, err
	}
	// This assertion is based on https://stackoverflow.com/a/28332019. It checks whether the two header bytes of
	// the file match the expected headers for a gzip file; the first one is 0x1f and the second is 0x8b
	return magicHeader, bytes.Equal(magicHeader, gzipMagicHeader), nil
}

func ContentContains(expected string) TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, contents []byte) {
		t.Helper()
		AssertContains(t, string(contents), expected)
	}
}

func ContentEquals(expected string) TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, contents []byte) {
		t.Helper()
		AssertEq(t, string(contents), expected)
	}
}

func SymlinksTo(expectedTarget string) TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, _ []byte) {
		t.Helper()
		if header.Typeflag != tar.TypeSymlink {
			t.Fatalf("path '%s' is not a symlink, type flag is '%c'", header.Name, header.Typeflag)
		}

		if header.Linkname != expectedTarget {
			t.Fatalf("symlink '%s' does not point to '%s', instead it points to '%s'", header.Name, expectedTarget, header.Linkname)
		}
	}
}

func AreEquivalentHardLinks() TarEntriesAssertion {
	return func(t *testing.T, header1 *tar.Header, _ []byte, header2 *tar.Header, _ []byte) {
		t.Helper()
		if header1.Typeflag != tar.TypeLink && header2.Typeflag != tar.TypeLink {
			t.Fatalf("path '%s' and '%s' are not hardlinks, type flags are '%c' and '%c'", header1.Name, header2.Name, header1.Typeflag, header2.Typeflag)
		}

		if header1.Linkname != header2.Name && header2.Linkname != header1.Name {
			t.Fatalf("'%s' and '%s' are not the same file", header1.Name, header2.Name)
		}
	}
}

func HasOwnerAndGroup(expectedUID int, expectedGID int) TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, _ []byte) {
		t.Helper()
		if header.Uid != expectedUID {
			t.Fatalf("expected '%s' to have uid '%d', but got '%d'", header.Name, expectedUID, header.Uid)
		}
		if header.Gid != expectedGID {
			t.Fatalf("expected '%s' to have gid '%d', but got '%d'", header.Name, expectedGID, header.Gid)
		}
	}
}

func IsJSON() TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, data []byte) {
		if !json.Valid(data) {
			t.Fatal("not valid JSON")
		}
	}
}

func IsGzipped() TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, data []byte) {
		_, isGzipped, err := isGzipped(bytes.NewReader(data))
		AssertNil(t, err)
		if !isGzipped {
			t.Fatal("is not gzipped")
		}
	}
}

func HasFileMode(expectedMode int64) TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, _ []byte) {
		t.Helper()
		if header.Mode != expectedMode {
			t.Fatalf("expected '%s' to have mode '%o', but got '%o'", header.Name, expectedMode, header.Mode)
		}
	}
}

func HasModTime(expectedTime time.Time) TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, _ []byte) {
		t.Helper()
		if header.ModTime.UnixNano() != expectedTime.UnixNano() {
			t.Fatalf("expected '%s' to have mod time '%s', but got '%s'", header.Name, expectedTime, header.ModTime)
		}
	}
}

func DoesNotHaveModTime(expectedTime time.Time) TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, _ []byte) {
		t.Helper()
		if header.ModTime.UnixNano() == expectedTime.UnixNano() {
			t.Fatalf("expected '%s' to not have mod time '%s'", header.Name, expectedTime)
		}
	}
}

func IsDirectory() TarEntryAssertion {
	return func(t *testing.T, header *tar.Header, _ []byte) {
		t.Helper()
		if header.Typeflag != tar.TypeDir {
			t.Fatalf("expected '%s' to be a directory but was '%d'", header.Name, header.Typeflag)
		}
	}
}
