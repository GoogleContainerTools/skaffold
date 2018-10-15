/*
Copyright 2018 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package acr

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

const VERSION = "2018-03-28"

type AzureBlobStorage struct {
	UploadURL string
	Buffer    *bytes.Buffer
}

func NewBlobStorage(url string) AzureBlobStorage {
	return AzureBlobStorage{
		UploadURL: url,
		Buffer:    new(bytes.Buffer),
	}
}

//Maximum supported file size is currently 256Mb
//as stated here https://docs.microsoft.com/en-us/rest/api/storageservices/put-blob#remarks
func (s AzureBlobStorage) UploadFileToBlob() error {
	req, err := http.NewRequest("PUT", s.UploadURL, s.Buffer)
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
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		return errors.New("couldn't push tar to blob")
	}
	return nil
}
