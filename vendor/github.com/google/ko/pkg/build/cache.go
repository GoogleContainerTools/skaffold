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

package build

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/logs"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/partial"
)

type diffIDToDescriptor map[v1.Hash]v1.Descriptor
type buildIDToDiffID map[string]v1.Hash

type layerCache struct {
	buildToDiff map[string]buildIDToDiffID
	diffToDesc  map[string]diffIDToDescriptor
	sync.Mutex
}

type layerFactory func() (v1.Layer, error)

func (c *layerCache) get(ctx context.Context, file string, miss layerFactory) (v1.Layer, error) {
	if os.Getenv("KOCACHE") == "" {
		return miss()
	}

	// Cache hit.
	if diffid, desc, err := c.getMeta(ctx, file); err != nil {
		logs.Debug.Printf("getMeta(%q): %v", file, err)
	} else {
		return &lazyLayer{
			diffid:     *diffid,
			desc:       *desc,
			buildLayer: miss,
		}, nil
	}

	// Cache miss.
	layer, err := miss()
	if err != nil {
		return nil, fmt.Errorf("miss(%q): %w", file, err)
	}
	if err := c.put(ctx, file, layer); err != nil {
		log.Printf("failed to cache metadata %s: %v", file, err)
	}
	return layer, nil
}

func (c *layerCache) getMeta(ctx context.Context, file string) (*v1.Hash, *v1.Descriptor, error) {
	buildid, err := getBuildID(ctx, file)
	if err != nil {
		return nil, nil, err
	}

	if buildid == "" {
		return nil, nil, fmt.Errorf("no buildid for %q", file)
	}

	// TODO: Implement better per-file locking.
	c.Lock()
	defer c.Unlock()

	btod, err := c.readBuildToDiff(file)
	if err != nil {
		return nil, nil, err
	}
	dtod, err := c.readDiffToDesc(file)
	if err != nil {
		return nil, nil, err
	}

	diffid, ok := btod[buildid]
	if !ok {
		return nil, nil, fmt.Errorf("no diffid for %q", buildid)
	}

	desc, ok := dtod[diffid]
	if !ok {
		return nil, nil, fmt.Errorf("no desc for %q", diffid)
	}

	return &diffid, &desc, nil
}

// Compute new layer metadata and cache it in-mem and on-disk.
func (c *layerCache) put(ctx context.Context, file string, layer v1.Layer) error {
	buildid, err := getBuildID(ctx, file)
	if err != nil {
		return err
	}

	desc, err := partial.Descriptor(layer)
	if err != nil {
		return err
	}

	diffid, err := layer.DiffID()
	if err != nil {
		return err
	}

	btod, ok := c.buildToDiff[file]
	if !ok {
		btod = buildIDToDiffID{}
	}
	btod[buildid] = diffid

	dtod, ok := c.diffToDesc[file]
	if !ok {
		dtod = diffIDToDescriptor{}
	}
	dtod[diffid] = *desc

	// TODO: Implement better per-file locking.
	c.Lock()
	defer c.Unlock()

	btodf, err := os.OpenFile(filepath.Join(filepath.Dir(file), "buildid-to-diffid"), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("opening buildid-to-diffid: %w", err)
	}
	defer btodf.Close()

	dtodf, err := os.OpenFile(filepath.Join(filepath.Dir(file), "diffid-to-descriptor"), os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		return fmt.Errorf("opening diffid-to-descriptor: %w", err)
	}
	defer dtodf.Close()

	enc := json.NewEncoder(btodf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&btod); err != nil {
		return err
	}

	enc = json.NewEncoder(dtodf)
	enc.SetIndent("", "  ")
	return enc.Encode(&dtod)
}

func (c *layerCache) readDiffToDesc(file string) (diffIDToDescriptor, error) {
	if dtod, ok := c.diffToDesc[file]; ok {
		return dtod, nil
	}

	dtodf, err := os.Open(filepath.Join(filepath.Dir(file), "diffid-to-descriptor"))
	if err != nil {
		return nil, fmt.Errorf("opening diffid-to-descriptor: %w", err)
	}
	defer dtodf.Close()

	var dtod diffIDToDescriptor
	if err := json.NewDecoder(dtodf).Decode(&dtod); err != nil {
		return nil, err
	}
	c.diffToDesc[file] = dtod
	return dtod, nil
}

func (c *layerCache) readBuildToDiff(file string) (buildIDToDiffID, error) {
	if btod, ok := c.buildToDiff[file]; ok {
		return btod, nil
	}

	btodf, err := os.Open(filepath.Join(filepath.Dir(file), "buildid-to-diffid"))
	if err != nil {
		return nil, fmt.Errorf("opening buildid-to-diffid: %w", err)
	}
	defer btodf.Close()

	var btod buildIDToDiffID
	if err := json.NewDecoder(btodf).Decode(&btod); err != nil {
		return nil, err
	}
	c.buildToDiff[file] = btod
	return btod, nil
}

func getBuildID(ctx context.Context, file string) (string, error) {
	gobin := getGoBinary()

	cmd := exec.CommandContext(ctx, gobin, "tool", "buildid", file)
	var output bytes.Buffer
	cmd.Stderr = &output
	cmd.Stdout = &output

	if err := cmd.Run(); err != nil {
		log.Printf("Unexpected error running \"go tool buildid %s\": %v\n%v", err, file, output.String())
		return "", fmt.Errorf("go tool buildid %s: %w", file, err)
	}
	return strings.TrimSpace(output.String()), nil
}
