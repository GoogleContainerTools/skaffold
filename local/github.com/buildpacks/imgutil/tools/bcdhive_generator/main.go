package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"text/template" //nolint:gosec
)

const tmplPath = `bcdhive_encoded.go.tmpl`

type tmplData struct{ PackageName, FuncName, EncodedBytes string }

func main() {
	outputFilePathPtr := flag.String("file", "", "generated file path")
	outputPackageNamePtr := flag.String("package", "", "generated package name")
	outputFuncNamePtr := flag.String("func", "", "generated function name")
	flag.Parse()

	if err := run(tmplPath, *outputFilePathPtr, *outputPackageNamePtr, *outputFuncNamePtr); err != nil {
		log.Fatal(err)
	}
}

func run(tmplPath, outputFilePath, outputPackageName, outputFuncName string) error {
	outputFile, err := os.OpenFile(filepath.Clean(outputFilePath), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	bcdBytes, err := HiveBCD()
	if err != nil {
		return err
	}

	encodedBCD, err := encodeData(bcdBytes)
	if err != nil {
		return err
	}

	out, err := generateGoSource(tmplPath, outputPackageName, outputFuncName, encodedBCD)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprint(outputFile, out); err != nil {
		return err
	}

	return nil
}

func encodeData(rawBytes []byte) (string, error) {
	gzipBuffer := &bytes.Buffer{}
	gzipWriter, err := gzip.NewWriterLevel(gzipBuffer, gzip.BestCompression)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(gzipWriter, bytes.NewBuffer(rawBytes)); err != nil {
		return "", err
	}

	if err := gzipWriter.Close(); err != nil {
		return "", err
	}

	encoded := base64.StdEncoding.EncodeToString(gzipBuffer.Bytes())
	return encoded, nil
}

func generateGoSource(tmplPath, packageName, funcName, bcdData string) (string, error) {
	tmpl, err := ioutil.ReadFile(filepath.Clean(tmplPath))
	if err != nil {
		return "", err
	}

	t := template.Must(template.New("tmpl").Parse(string(tmpl)))

	buf := &bytes.Buffer{}
	if err := t.Execute(buf, tmplData{packageName, funcName, bcdData}); err != nil {
		return "", err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return "", err
	}

	return string(src), nil
}
