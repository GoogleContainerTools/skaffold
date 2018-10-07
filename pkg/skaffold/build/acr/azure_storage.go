package acr

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

const VERSION = "2018-03-28"

type AzureBlobStorage struct {
	UploadUrl string
	Buffer    *bytes.Buffer
}

func NewBlobStorage(url string) AzureBlobStorage {
	return AzureBlobStorage{
		UploadUrl: url,
		Buffer:    new(bytes.Buffer),
	}
}

//Maximum supported file size is currently 256Mb
//as stated here https://docs.microsoft.com/en-us/rest/api/storageservices/put-blob#remarks
func (s AzureBlobStorage) UploadFileToBlob() error {
	req, err := http.NewRequest("PUT", s.UploadUrl, s.Buffer)
	if err != nil {
		return err
	}
	req.Header.Add("x-ms-blob-type", "BlockBlob")
	req.Header.Add("x-ms-version", VERSION)
	req.Header.Add("x-ms-date", time.Now().String())
	req.Header.Add("Content-Length", fmt.Sprint(s.Buffer.Len()))

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
