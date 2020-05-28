package layer

import (
	"archive/tar"
	"io"
	"path"
	"strings"
)

type WindowsWriter struct {
	tarWriter          *tar.Writer
	writtenParentPaths map[string]bool
}

func NewWindowsWriter(fileWriter io.Writer) *WindowsWriter {
	return &WindowsWriter{
		tarWriter:          tar.NewWriter(fileWriter),
		writtenParentPaths: map[string]bool{},
	}
}

func (w *WindowsWriter) Write(content []byte) (int, error) {
	return w.tarWriter.Write(content)
}

func (w *WindowsWriter) WriteHeader(header *tar.Header) error {
	if err := w.initializeLayer(); err != nil {
		return err
	}

	header.Name = layerFilesPath(header.Name)

	err := w.writeParentPaths(header.Name)
	if err != nil {
		return err
	}

	if header.Typeflag == tar.TypeDir {
		return w.writeDirHeader(header)
	}
	return w.tarWriter.WriteHeader(header)
}

func (w *WindowsWriter) Close() (err error) {
	defer func() {
		closeErr := w.tarWriter.Close()
		if err == nil {
			err = closeErr
		}
	}()
	return w.initializeLayer()
}

func (w *WindowsWriter) Flush() error {
	return w.tarWriter.Flush()
}

func (w *WindowsWriter) writeParentPaths(childPath string) error {
	var parentDir string
	for _, pathPart := range strings.Split(path.Dir(childPath), "/") {
		parentDir = path.Join(parentDir, pathPart)

		if err := w.writeDirHeader(&tar.Header{
			Name:     parentDir,
			Typeflag: tar.TypeDir,
		}); err != nil {
			return err
		}
	}
	return nil
}

func layerFilesPath(origPath string) string {
	return path.Join("Files", origPath)
}

func (w *WindowsWriter) initializeLayer() error {
	if err := w.writeDirHeader(&tar.Header{
		Name:     "Files",
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}
	if err := w.writeDirHeader(&tar.Header{
		Name:     "Hives",
		Typeflag: tar.TypeDir,
	}); err != nil {
		return err
	}
	return nil
}

func (w *WindowsWriter) writeDirHeader(header *tar.Header) error {
	if w.writtenParentPaths[header.Name] {
		return nil
	}
	if err := w.tarWriter.WriteHeader(header); err != nil {
		return err
	}
	w.writtenParentPaths[header.Name] = true
	return nil
}
