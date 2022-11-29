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

package format

import (
	"encoding/json"
	"errors"
	"io"

	sErrors "github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/errors"
	"github.com/GoogleContainerTools/skaffold/v2/proto/v1"
)

type Formatter interface {
	Write(interface{}) error
	WriteErr(error) error
}

type JSONFormatter struct {
	Out io.Writer
}

func (j JSONFormatter) Write(data interface{}) error {
	return json.NewEncoder(j.Out).Encode(data)
}

type jsonErrorOutput struct {
	ErrorCode    string `json:"errorCode"`
	ErrorMessage string `json:"errorMessage"`
}

func (j JSONFormatter) WriteErr(err error) error {
	var sErr sErrors.Error
	var jsonErr jsonErrorOutput
	if errors.As(err, &sErr) {
		jsonErr = jsonErrorOutput{ErrorCode: sErr.StatusCode().String(), ErrorMessage: sErr.Error()}
	} else {
		jsonErr = jsonErrorOutput{ErrorCode: proto.StatusCode_INSPECT_UNKNOWN_ERR.String(), ErrorMessage: err.Error()}
	}
	return json.NewEncoder(j.Out).Encode(jsonErr)
}
