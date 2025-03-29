package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
	"os/exec"
	"testing"

	"gotest.tools/assert"
)

// you must `go generate` a valid stub before running tests
//go:generate go run ./ -file=bcdhive_stub.go -package=main -func=StubHiveBCD

func TestBaseLayerBCDMemoMatchesActual(t *testing.T) {
	bcdBytes, err := HiveBCD()
	assert.NilError(t, err)

	gzipBuffer := &bytes.Buffer{}
	gzipWriter, err := gzip.NewWriterLevel(gzipBuffer, gzip.BestCompression)
	assert.NilError(t, err)

	_, err = io.Copy(gzipWriter, bytes.NewBuffer(bcdBytes))
	assert.NilError(t, err)

	assert.NilError(t, gzipWriter.Close())

	expectedEncodedBCD := base64.StdEncoding.EncodeToString(gzipBuffer.Bytes())

	assert.Equal(t, encodedBytes, expectedEncodedBCD)
}

func TestBaseLayerBCDMemo(t *testing.T) {
	bcdBytes, err := StubHiveBCD()
	assert.NilError(t, err)

	assertIsBCDBaseLayer(t, bcdBytes)
}

func TestBaseLayerBCDActual(t *testing.T) {
	bcdBytes, err := HiveBCD()
	assert.NilError(t, err)

	assertIsBCDBaseLayer(t, bcdBytes)
}

func assertIsBCDBaseLayer(t *testing.T, bcdBytes []byte) {
	t.Helper()

	hiveFile, err := os.CreateTemp("", "")
	assert.NilError(t, err)
	defer hiveFile.Close()
	defer os.Remove(hiveFile.Name())

	_, err = io.Copy(hiveFile, bytes.NewBuffer(bcdBytes))
	assert.NilError(t, err)

	cmd := exec.Command("hivexregedit", "--export", hiveFile.Name(), "Description")
	descriptionRegOutput, err := cmd.CombinedOutput()
	assert.NilError(t, err)

	assert.Equal(t, string(descriptionRegOutput),
		`Windows Registry Editor Version 5.00

[\Description]
"FirmwareModified"=dword:00000001
"KeyName"=hex(1):42,00,43,00,44,00,30,00,30,00,30,00,30,00,30,00,30,00,30,00,30,00,00,00

`)

	cmd = exec.Command("hivexregedit", "--export", hiveFile.Name(), "Objects")
	objectsRegOutput, err := cmd.CombinedOutput()
	assert.NilError(t, err)

	assert.Equal(t, string(objectsRegOutput),
		`Windows Registry Editor Version 5.00

[\Objects]

[\Objects\{6a6c1f1b-59d4-11ea-9438-9402e6abd998}]

[\Objects\{6a6c1f1b-59d4-11ea-9438-9402e6abd998}\Description]
"Type"=dword:10200003

[\Objects\{6a6c1f1b-59d4-11ea-9438-9402e6abd998}\Elements]

[\Objects\{6a6c1f1b-59d4-11ea-9438-9402e6abd998}\Elements\12000004]
"Element"=hex(1):62,00,75,00,69,00,6c,00,64,00,70,00,61,00,63,00,6b,00,73,00,2e,00,69,00,6f,00,00,00

[\Objects\{9dea862c-5cdd-4e70-acc1-f32b344d4795}]

[\Objects\{9dea862c-5cdd-4e70-acc1-f32b344d4795}\Description]
"Type"=dword:10100002

[\Objects\{9dea862c-5cdd-4e70-acc1-f32b344d4795}\Elements]

[\Objects\{9dea862c-5cdd-4e70-acc1-f32b344d4795}\Elements\23000003]
"Element"=hex(1):7b,00,36,00,61,00,36,00,63,00,31,00,66,00,31,00,62,00,2d,00,35,00,39,00,64,00,34,00,2d,00,31,00,31,00,65,00,61,00,2d,00,39,00,34,00,33,00,38,00,2d,00,39,00,34,00,30,00,32,00,65,00,36,00,61,00,62,00,64,00,39,00,39,00,38,00,7d,00,00,00

`)
}
