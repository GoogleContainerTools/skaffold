// Copyright 2022 Google LLC All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sbom

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/sigstore/cosign/pkg/oci"
)

func h1ToSHA256(s string) string {
	if !strings.HasPrefix(s, "h1:") {
		return ""
	}
	b, err := base64.StdEncoding.DecodeString(s[3:])
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func GenerateImageCycloneDX(mod []byte) ([]byte, error) {
	var err error
	mod, err = massageGoVersionM(mod)
	if err != nil {
		return nil, err
	}

	bi, err := ParseBuildInfo(string(mod))
	if err != nil {
		return nil, err
	}

	doc := document{
		BOMFormat:   "CycloneDX",
		SpecVersion: "1.4",
		Version:     1,
		Metadata: metadata{
			Component: component{
				BOMRef:  bomRef(&bi.Main),
				Type:    "application",
				Name:    bi.Main.Path,
				Version: bi.Main.Version,
				Purl:    bomRef(&bi.Main),
				ExternalReferences: []externalReference{{
					URL:  "https://" + bi.Main.Path,
					Type: "vcs",
				}},
			},
			Properties: []property{{
				Name:  "cdx:gomod:binary:name",
				Value: "out",
			}},
			// TODO: include all hashes
			// TODO: include go version
			// TODO: include bi.Settings?
		},
		Dependencies: []dependency{{
			Ref: bomRef(&bi.Main),
		}},
		Compositions: []composition{{
			Aggregate: "complete",
			Dependencies: []string{
				bomRef(&bi.Main),
			},
		}, {
			Aggregate:    "unknown",
			Dependencies: []string{},
		}},
	}
	for _, dep := range bi.Deps {
		// Don't include replaced deps
		if dep.Replace != nil {
			continue
		}
		comp := component{
			BOMRef:  bomRef(dep),
			Type:    "library",
			Name:    dep.Path,
			Version: dep.Version,
			Scope:   "required",
			Purl:    bomRef(dep),
			ExternalReferences: []externalReference{{
				URL:  "https://" + dep.Path,
				Type: "vcs",
			}},
		}
		if dep.Sum != "" {
			comp.Hashes = []hash{{
				Alg:     "SHA-256",
				Content: h1ToSHA256(dep.Sum),
			}}
		}
		doc.Components = append(doc.Components, comp)
		doc.Dependencies[0].DependsOn = append(doc.Dependencies[0].DependsOn, bomRef(dep))
		doc.Dependencies = append(doc.Dependencies, dependency{
			Ref: bomRef(dep),
		})

		doc.Compositions[1].Dependencies = append(doc.Compositions[1].Dependencies, bomRef(dep))
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func GenerateIndexCycloneDX(sii oci.SignedImageIndex) ([]byte, error) {
	return nil, nil
}

type document struct {
	BOMFormat    string        `json:"bomFormat"`
	SpecVersion  string        `json:"specVersion"`
	Version      int           `json:"version"`
	Metadata     metadata      `json:"metadata"`
	Components   []component   `json:"components,omitempty"`
	Dependencies []dependency  `json:"dependencies,omitempty"`
	Compositions []composition `json:"compositions,omitempty"`
}
type metadata struct {
	Component  component  `json:"component"`
	Properties []property `json:"properties,omitempty"`
}
type component struct {
	BOMRef             string              `json:"bom-ref"`
	Type               string              `json:"type"`
	Name               string              `json:"name"`
	Version            string              `json:"version"`
	Scope              string              `json:"scope,omitempty"`
	Hashes             []hash              `json:"hashes,omitempty"`
	Purl               string              `json:"purl"`
	ExternalReferences []externalReference `json:"externalReferences"`
}
type hash struct {
	Alg     string `json:"alg"`
	Content string `json:"content"`
}
type externalReference struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}
type property struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type dependency struct {
	Ref       string   `json:"ref"`
	DependsOn []string `json:"dependsOn,omitempty"`
}
type composition struct {
	Aggregate    string   `json:"aggregate"`
	Dependencies []string `json:"dependencies,omitempty"`
}
