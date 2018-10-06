package acr

import (
	"bufio"
	"bytes"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"time"
)

const VERSION = "2018-03-28"

type AzureBlobStorage struct {
	UploadUrl string
	Bytes     bytes.Buffer
}

func NewBlobStorage(url string) AzureBlobStorage {
	return AzureBlobStorage{
		UploadUrl: url,
	}
}

func (s AzureBlobStorage) Writer() io.Writer {
	return bufio.NewWriter(&s.Bytes)
}

func (s AzureBlobStorage) UploadFileToBlob() error {
	req, err := http.NewRequest("PUT", s.UploadUrl, bytes.NewBuffer(s.Bytes.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Add("x-ms-blob-type", "BlockBlob")
	req.Header.Add("x-ms-version", VERSION)
	req.Header.Add("x-ms-date", time.Now().String())
	req.Header.Add("Content-Length", string(s.Bytes.Len()))

	client := http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusCreated {
		return errors.New("couldn't file to blob.")
	}
	return nil
}
