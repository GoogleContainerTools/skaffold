/*
Copyright 2021 The Skaffold Authors

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

package inspect

import (
	"encoding/json"
	"errors"
	"io"

	sErrors "github.com/GoogleContainerTools/skaffold/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/proto/v1"
)

type formatter interface {
	Write(interface{}) error
	WriteErr(error) error
}

func getOutputFormatter(out io.Writer, _ string) formatter {
	// TODO: implement other output formatters. Currently only JSON is implemented
	return jsonFormatter{out: out}
}

type jsonFormatter struct {
	out io.Writer
}

func (j jsonFormatter) Write(data interface{}) error {
	return json.NewEncoder(j.out).Encode(data)
}

type jsonErrorOutput struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func (j jsonFormatter) WriteErr(err error) error {
	var sErr sErrors.Error
	var jsonErr jsonErrorOutput
	if errors.As(err, &sErr) {
		jsonErr = jsonErrorOutput{ErrorCode: sErr.StatusCode().String(), ErrorMessage: sErr.Error()}
	} else {
		jsonErr = jsonErrorOutput{ErrorCode: proto.StatusCode_UNKNOWN_ERROR.String(), ErrorMessage: err.Error()}
	}
	return json.NewEncoder(j.out).Encode(jsonErr)
}
