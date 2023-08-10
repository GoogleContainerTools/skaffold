package layer

import (
	"archive/tar"
	"fmt"
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

	if err := w.validateHeader(header); err != nil {
		return err
	}
	header.Name = layerFilesPath(header.Name)

	err := w.writeParentPaths(header.Name)
	if err != nil {
		return err
	}

	header.Format = tar.FormatPAX
	if header.PAXRecords == nil {
		header.PAXRecords = map[string]string{}
	}
	ensureSecurityDescriptor(header)

	if header.Typeflag == tar.TypeDir {
		return w.writeDirHeader(header)
	}
	return w.tarWriter.WriteHeader(header)
}

func ensureSecurityDescriptor(header *tar.Header) {
	if _, ok := header.PAXRecords["MSWINDOWS.rawsd"]; !ok {
		if header.Uid == 0 && header.Gid == 0 {
			header.PAXRecords["MSWINDOWS.rawsd"] = AdministratratorOwnerAndGroupSID
		} else {
			header.PAXRecords["MSWINDOWS.rawsd"] = UserOwnerAndGroupSID
		}
	}
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

func (w *WindowsWriter) validateHeader(header *tar.Header) error {
	if !path.IsAbs(header.Name) {
		return fmt.Errorf("invalid header name: must be absolute, posix path: %s", header.Name)
	}
	return nil
}
