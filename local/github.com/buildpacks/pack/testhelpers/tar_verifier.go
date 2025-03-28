package testhelpers

import (
	"archive/tar"
	"io"
	"testing"
	"time"
)

type TarVerifier struct {
	t   *testing.T
	tr  *tar.Reader
	uid int
	gid int
}

func NewTarVerifier(t *testing.T, tr *tar.Reader, uid, gid int) *TarVerifier {
	return &TarVerifier{
		t:   t,
		tr:  tr,
		uid: uid,
		gid: gid,
	}
}

func (v *TarVerifier) NextDirectory(name string, mode int64) {
	v.t.Helper()
	header, err := v.tr.Next()
	if err != nil {
		v.t.Fatalf("Failed to get next file: %s", err)
	}

	if header.Name != name {
		v.t.Fatalf(`expected dir with name %s, got %s`, name, header.Name)
	}
	if header.Typeflag != tar.TypeDir {
		v.t.Fatalf(`expected %s to be a Directory`, header.Name)
	}
	if header.Uid != v.uid {
		v.t.Fatalf(`expected %s to have Uid %d but, got: %d`, header.Name, v.uid, header.Uid)
	}
	if header.Gid != v.gid {
		v.t.Fatalf(`expected %s to have Gid %d but, got: %d`, header.Name, v.gid, header.Gid)
	}
	if header.Mode != mode {
		v.t.Fatalf(`expected %s to have mode %o but, got: %o`, header.Name, mode, header.Mode)
	}
	if !header.ModTime.Equal(time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)) {
		v.t.Fatalf(`expected %s to have been normalized, got: %s`, header.Name, header.ModTime.String())
	}
}

func (v *TarVerifier) NoMoreFilesExist() {
	v.t.Helper()
	header, err := v.tr.Next()
	if err == nil {
		v.t.Fatalf(`expected no more files but found: %s`, header.Name)
	} else if err != io.EOF {
		v.t.Error(err.Error())
	}
}

func (v *TarVerifier) NextFile(name, expectedFileContents string, expectedFileMode int64) {
	v.t.Helper()
	header, err := v.tr.Next()
	if err != nil {
		v.t.Fatalf("Failed to get next file: %s", err)
	}

	if header.Name != name {
		v.t.Fatalf(`expected dir with name %s, got %s`, name, header.Name)
	}
	if header.Typeflag != tar.TypeReg {
		v.t.Fatalf(`expected %s to be a file`, header.Name)
	}
	if header.Uid != v.uid {
		v.t.Fatalf(`expected %s to have Uid %d but, got: %d`, header.Name, v.uid, header.Uid)
	}
	if header.Gid != v.gid {
		v.t.Fatalf(`expected %s to have Gid %d but, got: %d`, header.Name, v.gid, header.Gid)
	}

	fileContents := make([]byte, header.Size)
	v.tr.Read(fileContents)
	if string(fileContents) != expectedFileContents {
		v.t.Fatalf(`expected to some-file.txt to have %s got %s`, expectedFileContents, string(fileContents))
	}

	if !header.ModTime.Equal(time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)) {
		v.t.Fatalf(`expected %s to have been normalized, got: %s`, header.Name, header.ModTime.String())
	}

	if header.Mode != expectedFileMode {
		v.t.Fatalf("files should have mode %o, got: %o", expectedFileMode, header.Mode)
	}
}

func (v *TarVerifier) NextSymLink(name, link string) {
	v.t.Helper()
	header, err := v.tr.Next()
	if err != nil {
		v.t.Fatalf("Failed to get next file: %s", err)
	}

	if header.Name != name {
		v.t.Fatalf(`expected dir with name %s, got %s`, name, header.Name)
	}
	if header.Typeflag != tar.TypeSymlink {
		v.t.Fatalf(`expected %s to be a link got %s`, header.Name, string(header.Typeflag))
	}
	if header.Uid != v.uid {
		v.t.Fatalf(`expected %s to have Uid %d but, got: %d`, header.Name, v.uid, header.Uid)
	}
	if header.Gid != v.gid {
		v.t.Fatalf(`expected %s to have Gid %d but, got: %d`, header.Name, v.gid, header.Gid)
	}

	// tar names and linknames should be Linux formatted paths, regardless of OS
	if header.Linkname != "../some-file.txt" {
		v.t.Fatalf(`expected link-file to have target %s got: %s`, link, header.Linkname)
	}
	if !header.ModTime.Equal(time.Date(1980, time.January, 1, 0, 0, 1, 0, time.UTC)) {
		v.t.Fatalf(`expected %s to have been normalized, got: %s`, header.Name, header.ModTime.String())
	}
}
