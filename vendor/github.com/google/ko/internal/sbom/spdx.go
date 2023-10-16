// Copyright 2021 ko Build Authors All Rights Reserved.
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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/types"
	specsv1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sigstore/cosign/v2/pkg/oci"
)

type qualifier struct {
	key   string
	value string
}

// ociRef constructs a pURL for the OCI image according to:
// https://github.com/package-url/purl-spec/blob/master/PURL-TYPES.rst#oci
func ociRef(path string, imgDigest v1.Hash, qual ...qualifier) string {
	parts := strings.Split(path, "/")
	purl := fmt.Sprintf("pkg:oci/%s@%s", parts[len(parts)-1], imgDigest.String())
	if num := len(qual); num > 0 {
		qs := make(url.Values, num)
		for _, q := range qual {
			qs.Add(q.key, q.value)
		}
		purl = purl + "?" + qs.Encode()
	}
	return purl
}

const dateFormat = "2006-01-02T15:04:05Z"

func GenerateImageSPDX(koVersion string, mod []byte, img oci.SignedImage) ([]byte, error) {
	var err error
	mod, err = massageGoVersionM(mod)
	if err != nil {
		return nil, err
	}

	bi, err := debug.ParseBuildInfo(string(mod))
	if err != nil {
		return nil, err
	}

	imgDigest, err := img.Digest()
	if err != nil {
		return nil, err
	}
	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, err
	}
	m, err := img.Manifest()
	if err != nil {
		return nil, err
	}

	doc, imageID := starterDocument(koVersion, cfg.Created.Time, imgDigest)

	// image -> main package -> transitive deps
	//       -> base image
	doc.Packages = make([]Package, 0, 3+len(bi.Deps))
	doc.Relationships = make([]Relationship, 0, 3+len(bi.Deps))

	doc.Relationships = append(doc.Relationships, Relationship{
		Element: "SPDXRef-DOCUMENT",
		Type:    "DESCRIBES",
		Related: imageID,
	})

	doc.Packages = append(doc.Packages, Package{
		ID:   imageID,
		Name: imgDigest.String(),
		// TODO: PackageSupplier: "Organization: " + bs.Main.Path
		DownloadLocation: NOASSERTION,
		FilesAnalyzed:    false,
		// TODO: PackageHomePage:  "https://" + bi.Main.Path,
		LicenseConcluded: NOASSERTION,
		LicenseDeclared:  NOASSERTION,
		CopyrightText:    NOASSERTION,
		PrimaryPurpose:   "CONTAINER",
		ExternalRefs: []ExternalRef{{
			Category: "PACKAGE-MANAGER",
			Type:     "purl",
			Locator: ociRef("image", imgDigest, qualifier{
				key:   "mediaType",
				value: string(m.MediaType),
			}),
		}},
	})

	if err := addBaseImage(&doc, m.Annotations, imgDigest); err != nil {
		return nil, err
	}

	mainPackageID := modulePackageName(&bi.Main)

	doc.Relationships = append(doc.Relationships, Relationship{
		Element: imageID,
		Type:    "CONTAINS",
		Related: mainPackageID,
	})

	doc.Packages = append(doc.Packages, Package{
		Name: bi.Main.Path,
		ID:   mainPackageID,
		// TODO: PackageSupplier: "Organization: " + bs.Main.Path
		DownloadLocation: "https://" + bi.Main.Path,
		FilesAnalyzed:    false,
		// TODO: PackageHomePage:  "https://" + bi.Main.Path,
		LicenseConcluded: NOASSERTION,
		LicenseDeclared:  NOASSERTION,
		CopyrightText:    NOASSERTION,
		ExternalRefs: []ExternalRef{{
			Category: "PACKAGE-MANAGER",
			Type:     "purl",
			Locator:  goRef(&bi.Main),
		}},
	})

	for _, dep := range bi.Deps {
		depID := modulePackageName(dep)

		doc.Relationships = append(doc.Relationships, Relationship{
			Element: mainPackageID,
			Type:    "DEPENDS_ON",
			Related: depID,
		})

		pkg := Package{
			ID:      depID,
			Name:    dep.Path,
			Version: dep.Version,
			// TODO: PackageSupplier: "Organization: " + dep.Path
			DownloadLocation: fmt.Sprintf("https://proxy.golang.org/%s/@v/%s.zip", dep.Path, dep.Version),
			FilesAnalyzed:    false,
			LicenseConcluded: NOASSERTION,
			LicenseDeclared:  NOASSERTION,
			CopyrightText:    NOASSERTION,
			ExternalRefs: []ExternalRef{{
				Category: "PACKAGE-MANAGER",
				Type:     "purl",
				Locator:  goRef(dep),
			}},
		}

		if dep.Sum != "" {
			pkg.Checksums = []Checksum{{
				Algorithm: "SHA256",
				Value:     h1ToSHA256(dep.Sum),
			}}
		}

		doc.Packages = append(doc.Packages, pkg)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractDate(sii oci.SignedImageIndex) (*time.Time, error) {
	im, err := sii.IndexManifest()
	if err != nil {
		return nil, err
	}
	for _, desc := range im.Manifests {
		switch desc.MediaType {
		case types.OCIManifestSchema1, types.DockerManifestSchema2:
			si, err := sii.SignedImage(desc.Digest)
			if err != nil {
				return nil, err
			}
			cfg, err := si.ConfigFile()
			if err != nil {
				return nil, err
			}
			return &cfg.Created.Time, nil

		default:
			// We shouldn't need to handle nested indices, since we don't build
			// them, but if we do we will need to do some sort of recursion here.
			return nil, fmt.Errorf("unknown media type: %v", desc.MediaType)
		}
	}
	return nil, errors.New("unable to extract date, no imaged found")
}

func GenerateIndexSPDX(koVersion string, sii oci.SignedImageIndex) ([]byte, error) {
	indexDigest, err := sii.Digest()
	if err != nil {
		return nil, err
	}

	date, err := extractDate(sii)
	if err != nil {
		return nil, err
	}
	im, err := sii.IndexManifest()
	if err != nil {
		return nil, err
	}

	doc, indexID := starterDocument(koVersion, *date, indexDigest)
	doc.Packages = []Package{{
		ID:               indexID,
		Name:             indexDigest.String(),
		DownloadLocation: NOASSERTION,
		FilesAnalyzed:    false,
		LicenseConcluded: NOASSERTION,
		LicenseDeclared:  NOASSERTION,
		CopyrightText:    NOASSERTION,
		PrimaryPurpose:   "CONTAINER",
		Checksums: []Checksum{{
			Algorithm: strings.ToUpper(indexDigest.Algorithm),
			Value:     indexDigest.Hex,
		}},
		ExternalRefs: []ExternalRef{{
			Category: "PACKAGE-MANAGER",
			Type:     "purl",
			Locator: ociRef("index", indexDigest, qualifier{
				key:   "mediaType",
				value: string(im.MediaType),
			}),
		}},
	}}

	if err := addBaseImage(&doc, im.Annotations, indexDigest); err != nil {
		return nil, err
	}
	for _, desc := range im.Manifests {
		switch desc.MediaType {
		case types.OCIManifestSchema1, types.DockerManifestSchema2:
			si, err := sii.SignedImage(desc.Digest)
			if err != nil {
				return nil, err
			}

			imageDigest, err := si.Digest()
			if err != nil {
				return nil, err
			}

			depID := ociPackageName(imageDigest)

			doc.Relationships = append(doc.Relationships, Relationship{
				Element: ociPackageName(indexDigest),
				Type:    "VARIANT_OF",
				Related: depID,
			})

			qual := []qualifier{{
				key:   "mediaType",
				value: string(desc.MediaType),
			}, {
				key:   "arch",
				value: desc.Platform.Architecture,
			}, {
				key:   "os",
				value: desc.Platform.OS,
			}}
			if desc.Platform.Variant != "" {
				qual = append(qual, qualifier{
					key:   "variant",
					value: desc.Platform.Variant,
				})
			}
			if desc.Platform.OSVersion != "" {
				qual = append(qual, qualifier{
					key:   "os-version",
					value: desc.Platform.OSVersion,
				})
			}
			for _, feat := range desc.Platform.OSFeatures {
				qual = append(qual, qualifier{
					key:   "os-feature",
					value: feat,
				})
			}

			doc.Packages = append(doc.Packages, Package{
				ID:      depID,
				Name:    imageDigest.String(),
				Version: desc.Platform.String(),
				// TODO: PackageSupplier: "Organization: " + dep.Path
				DownloadLocation: NOASSERTION,
				FilesAnalyzed:    false,
				LicenseConcluded: NOASSERTION,
				LicenseDeclared:  NOASSERTION,
				CopyrightText:    NOASSERTION,
				PrimaryPurpose:   "CONTAINER",
				ExternalRefs: []ExternalRef{{
					Category: "PACKAGE-MANAGER",
					Type:     "purl",
					Locator:  ociRef("image", imageDigest, qual...),
				}},
				Checksums: []Checksum{{
					Algorithm: strings.ToUpper(imageDigest.Algorithm),
					Value:     imageDigest.Hex,
				}},
			})

		default:
			// We shouldn't need to handle nested indices, since we don't build
			// them, but if we do we will need to do some sort of recursion here.
			return nil, fmt.Errorf("unknown media type: %v", desc.MediaType)
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func ociPackageName(d v1.Hash) string {
	return fmt.Sprintf("SPDXRef-Package-%s-%s", d.Algorithm, d.Hex)
}

func starterDocument(koVersion string, date time.Time, d v1.Hash) (Document, string) {
	digestID := ociPackageName(d)
	return Document{
		ID:      "SPDXRef-DOCUMENT",
		Version: Version,
		CreationInfo: CreationInfo{
			Created:  date.Format(dateFormat),
			Creators: []string{"Tool: ko " + koVersion},
		},
		DataLicense:       "CC0-1.0",
		Name:              "sbom-" + d.String(),
		Namespace:         "http://spdx.org/spdxdocs/ko/" + d.String(),
		DocumentDescribes: []string{digestID},
	}, digestID
}

func addBaseImage(doc *Document, annotations map[string]string, h v1.Hash) error {
	// Check for the base image annotation.
	base, ok := annotations[specsv1.AnnotationBaseImageName]
	if !ok {
		return nil
	}
	rawHash, ok := annotations[specsv1.AnnotationBaseImageDigest]
	if !ok {
		return nil
	}
	ref, err := name.ParseReference(base)
	if err != nil {
		return err
	}
	hash, err := v1.NewHash(rawHash)
	if err != nil {
		return err
	}
	digest := ref.Context().Digest(hash.String())

	depID := ociPackageName(hash)

	doc.Relationships = append(doc.Relationships, Relationship{
		Element: ociPackageName(h),
		Type:    "DESCENDANT_OF",
		Related: depID,
	})

	qual := []qualifier{{
		key:   "repository_url",
		value: ref.Context().Name(),
	}}

	if t, ok := ref.(name.Tag); ok {
		qual = append(qual, qualifier{
			key:   "tag",
			value: t.Identifier(),
		})
	}

	doc.Packages = append(doc.Packages, Package{
		ID:      depID,
		Name:    digest.String(),
		Version: ref.String(),
		// TODO: PackageSupplier: "Organization: " + dep.Path
		DownloadLocation: NOASSERTION,
		FilesAnalyzed:    false,
		LicenseConcluded: NOASSERTION,
		LicenseDeclared:  NOASSERTION,
		CopyrightText:    NOASSERTION,
		ExternalRefs: []ExternalRef{{
			Category: "PACKAGE-MANAGER",
			Type:     "purl",
			Locator:  ociRef("image", hash, qual...),
		}},
		Checksums: []Checksum{{
			Algorithm: strings.ToUpper(hash.Algorithm),
			Value:     hash.Hex,
		}},
	})
	return nil
}

// Below this is forked from here:
// https://github.com/kubernetes-sigs/bom/blob/main/pkg/spdx/json/v2.2.2/types.go

/*
Copyright 2022 The Kubernetes Authors.

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

const (
	NOASSERTION = "NOASSERTION"
	Version     = "SPDX-2.3"
)

type Document struct {
	ID                   string                `json:"SPDXID"`
	Name                 string                `json:"name"`
	Version              string                `json:"spdxVersion"`
	CreationInfo         CreationInfo          `json:"creationInfo"`
	DataLicense          string                `json:"dataLicense"`
	Namespace            string                `json:"documentNamespace"`
	DocumentDescribes    []string              `json:"documentDescribes,omitempty"`
	Files                []File                `json:"files,omitempty"`
	Packages             []Package             `json:"packages,omitempty"`
	Relationships        []Relationship        `json:"relationships,omitempty"`
	ExternalDocumentRefs []ExternalDocumentRef `json:"externalDocumentRefs,omitempty"`
}

type CreationInfo struct {
	Created            string   `json:"created"` // Date
	Creators           []string `json:"creators,omitempty"`
	LicenseListVersion string   `json:"licenseListVersion,omitempty"`
}

type Package struct {
	ID                   string                   `json:"SPDXID"`
	Name                 string                   `json:"name"`
	Version              string                   `json:"versionInfo,omitempty"`
	FilesAnalyzed        bool                     `json:"filesAnalyzed"`
	LicenseDeclared      string                   `json:"licenseDeclared"`
	LicenseConcluded     string                   `json:"licenseConcluded"`
	Description          string                   `json:"description,omitempty"`
	DownloadLocation     string                   `json:"downloadLocation"`
	Originator           string                   `json:"originator,omitempty"`
	SourceInfo           string                   `json:"sourceInfo,omitempty"`
	CopyrightText        string                   `json:"copyrightText"`
	PrimaryPurpose       string                   `json:"primaryPackagePurpose,omitempty"`
	HasFiles             []string                 `json:"hasFiles,omitempty"`
	LicenseInfoFromFiles []string                 `json:"licenseInfoFromFiles,omitempty"`
	Checksums            []Checksum               `json:"checksums,omitempty"`
	ExternalRefs         []ExternalRef            `json:"externalRefs,omitempty"`
	VerificationCode     *PackageVerificationCode `json:"packageVerificationCode,omitempty"`
}

type PackageVerificationCode struct {
	Value         string   `json:"packageVerificationCodeValue"`
	ExcludedFiles []string `json:"packageVerificationCodeExcludedFiles,omitempty"`
}

type File struct {
	ID                string     `json:"SPDXID"`
	Name              string     `json:"fileName"`
	CopyrightText     string     `json:"copyrightText"`
	NoticeText        string     `json:"noticeText,omitempty"`
	LicenseConcluded  string     `json:"licenseConcluded"`
	Description       string     `json:"description,omitempty"`
	FileTypes         []string   `json:"fileTypes,omitempty"`
	LicenseInfoInFile []string   `json:"licenseInfoInFiles"` // List of licenses
	Checksums         []Checksum `json:"checksums"`
}

type Checksum struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"checksumValue"`
}

type ExternalRef struct {
	Category string `json:"referenceCategory"`
	Locator  string `json:"referenceLocator"`
	Type     string `json:"referenceType"`
}

type Relationship struct {
	Element string `json:"spdxElementId"`
	Type    string `json:"relationshipType"`
	Related string `json:"relatedSpdxElement"`
}

type ExternalDocumentRef struct {
	Checksum           Checksum `json:"checksum"`
	ExternalDocumentID string   `json:"externalDocumentId"`
	SPDXDocument       string   `json:"spdxDocument"`
}
