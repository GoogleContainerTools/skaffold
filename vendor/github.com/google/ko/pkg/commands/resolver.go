// Copyright 2018 ko Build Authors All Rights Reserved.
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

package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/google/go-containerregistry/pkg/name"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/google/ko/pkg/build"
	"github.com/google/ko/pkg/commands/options"
	"github.com/google/ko/pkg/publish"
	"github.com/google/ko/pkg/resolve"
)

// ua returns the ko user agent.
func ua() string {
	if v := version(); v != "" {
		return "ko/" + v
	}
	return "ko"
}

func gobuildOptions(bo *options.BuildOptions) ([]build.Option, error) {
	creationTime, err := getCreationTime()
	if err != nil {
		return nil, err
	}

	kodataCreationTime, err := getKoDataCreationTime()
	if err != nil {
		return nil, err
	}

	if len(bo.Platforms) == 0 && len(bo.DefaultPlatforms) > 0 {
		bo.Platforms = bo.DefaultPlatforms
	}

	if len(bo.Platforms) == 0 {
		envPlatform := "linux/amd64"

		goos, goarch, goarm := os.Getenv("GOOS"), os.Getenv("GOARCH"), os.Getenv("GOARM")

		// Default to linux/amd64 unless GOOS and GOARCH are set.
		if goos != "" && goarch != "" {
			envPlatform = path.Join(goos, goarch)
		}

		// Use GOARM for variant if it's set and GOARCH is arm.
		if strings.Contains(goarch, "arm") && goarm != "" {
			envPlatform = path.Join(envPlatform, "v"+goarm)
		}

		bo.Platforms = []string{envPlatform}
	} else {
		// Make sure these are all unset
		for _, env := range []string{"GOOS", "GOARCH", "GOARM"} {
			if s, ok := os.LookupEnv(env); ok {
				return nil, fmt.Errorf("cannot use --platform or defaultPlatforms in .ko.yaml or env KO_DEFAULTPLATFORMS combined with %s=%q", env, s)
			}
		}
	}

	opts := []build.Option{
		build.WithBaseImages(getBaseImage(bo)),
		build.WithDefaultEnv(bo.DefaultEnv),
		build.WithDefaultFlags(bo.DefaultFlags),
		build.WithDefaultLdflags(bo.DefaultLdflags),
		build.WithPlatforms(bo.Platforms...),
		build.WithJobs(bo.ConcurrentBuilds),
	}
	if creationTime != nil {
		opts = append(opts, build.WithCreationTime(*creationTime))
	}
	if kodataCreationTime != nil {
		opts = append(opts, build.WithKoDataCreationTime(*kodataCreationTime))
	}
	if bo.DisableOptimizations {
		opts = append(opts, build.WithDisabledOptimizations())
	}
	if bo.Debug {
		opts = append(opts, build.WithDebugger())
		opts = append(opts, build.WithDisabledOptimizations()) // also needed for Delve
	}
	switch bo.SBOM {
	case "none":
		opts = append(opts, build.WithDisabledSBOM())
	default: // "spdx"
		opts = append(opts, build.WithSPDX(version()))
	}
	opts = append(opts, build.WithTrimpath(bo.Trimpath))
	for _, lf := range bo.Labels {
		parts := strings.SplitN(lf, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label flag: %s", lf)
		}
		opts = append(opts, build.WithLabel(parts[0], parts[1]))
	}
	for _, an := range bo.Annotations {
		k, v, ok := strings.Cut(an, "=")
		if !ok {
			return nil, fmt.Errorf("missing '=' in annotation: %s", an)
		}
		opts = append(opts, build.WithAnnotation(k, v))
	}

	if bo.User != "" {
		opts = append(opts, build.WithUser(bo.User))
	}

	if bo.BuildConfigs != nil {
		opts = append(opts, build.WithConfig(bo.BuildConfigs))
	}

	if bo.SBOMDir != "" {
		opts = append(opts, build.WithSBOMDir(bo.SBOMDir))
	}

	return opts, nil
}

// NewBuilder creates a ko builder
func NewBuilder(ctx context.Context, bo *options.BuildOptions) (build.Interface, error) {
	return makeBuilder(ctx, bo)
}

func makeBuilder(ctx context.Context, bo *options.BuildOptions) (*build.Caching, error) {
	if err := bo.LoadConfig(); err != nil {
		return nil, err
	}
	opt, err := gobuildOptions(bo)
	if err != nil {
		return nil, fmt.Errorf("error setting up builder options: %w", err)
	}
	innerBuilder, err := build.NewGobuilds(ctx, bo.WorkingDirectory, bo.BuildConfigs, opt...)
	if err != nil {
		return nil, err
	}

	// tl;dr Wrap builder in a caching builder.
	//
	// The caching builder should on Build calls:
	//  - Check for a valid Build future
	//    - if a valid Build future exists at the time of the request,
	//      then block on it.
	//    - if it does not, then initiate and record a Build future.
	//
	// This will benefit the following key cases:
	// 1. When the same import path is referenced across multiple yaml files
	//    we can elide subsequent builds by blocking on the same image future.
	// 2. When an affected yaml file has multiple import paths (mostly unaffected)
	//    we can elide the builds of unchanged import paths.
	return build.NewCaching(innerBuilder)
}

// NewPublisher creates a ko publisher
func NewPublisher(po *options.PublishOptions) (publish.Interface, error) {
	return makePublisher(po)
}

func makePublisher(po *options.PublishOptions) (publish.Interface, error) {
	// use each tag only once
	po.Tags = unique(po.Tags)
	// Create the publish.Interface that we will use to publish image references
	// to either a docker daemon or a container image registry.
	innerPublisher, err := func() (publish.Interface, error) {
		repoName := po.DockerRepo
		namer := options.MakeNamer(po)
		// Default LocalDomain if unset.
		if po.LocalDomain == "" {
			po.LocalDomain = publish.LocalDomain
		}
		// If repoName is unset with --local, default it to the local domain.
		if po.Local && repoName == "" {
			repoName = po.LocalDomain
		}
		// When in doubt, if repoName is under the local domain, default to --local.
		po.Local = po.Local || strings.HasPrefix(repoName, po.LocalDomain)
		if po.Local {
			// TODO(jonjohnsonjr): I'm assuming that nobody will
			// use local with other publishers, but that might
			// not be true.
			po.LocalDomain = repoName
			return publish.NewDaemon(namer, po.Tags,
				publish.WithDockerClient(po.DockerClient),
				publish.WithLocalDomain(po.LocalDomain),
			)
		}
		if strings.HasPrefix(repoName, publish.KindDomain) {
			return publish.NewKindPublisher(repoName, namer, po.Tags), nil
		}

		if repoName == "" && po.Push {
			return nil, errors.New("KO_DOCKER_REPO environment variable is unset")
		}
		if _, err := name.NewRegistry(repoName); err != nil {
			if _, err := name.NewRepository(repoName); err != nil {
				return nil, fmt.Errorf("failed to parse %q as repository: %w", repoName, err)
			}
		}

		publishers := []publish.Interface{}
		if po.OCILayoutPath != "" {
			lp := publish.NewLayout(po.OCILayoutPath)
			publishers = append(publishers, lp)
		}
		if po.TarballFile != "" {
			tp := publish.NewTarball(po.TarballFile, repoName, namer, po.Tags)
			publishers = append(publishers, tp)
		}
		userAgent := ua()
		if po.UserAgent != "" {
			userAgent = po.UserAgent
		}
		if po.Push {
			dp, err := publish.NewDefault(repoName,
				publish.WithUserAgent(userAgent),
				publish.WithAuthFromKeychain(keychain),
				publish.WithNamer(namer),
				publish.WithTags(po.Tags),
				publish.WithTagOnly(po.TagOnly),
				publish.Insecure(po.InsecureRegistry),
				publish.WithJobs(po.Jobs),
			)
			if err != nil {
				return nil, err
			}
			publishers = append(publishers, dp)
		}

		// If not publishing, at least generate a digest to simulate
		// publishing.
		if len(publishers) == 0 {
			// If one or more tags are specified, use the first tag in the list
			var tag string
			if len(po.Tags) >= 1 {
				tag = po.Tags[0]
			}
			publishers = append(publishers, nopPublisher{
				repoName: repoName,
				namer:    namer,
				tag:      tag,
				tagOnly:  po.TagOnly,
			})
		}

		return publish.MultiPublisher(publishers...), nil
	}()
	if err != nil {
		return nil, err
	}

	if po.ImageRefsFile != "" {
		innerPublisher, err = publish.NewRecorder(innerPublisher, po.ImageRefsFile)
		if err != nil {
			return nil, err
		}
	}

	// Wrap publisher in a memoizing publisher implementation.
	return publish.NewCaching(innerPublisher)
}

// nopPublisher simulates publishing without actually publishing anything, to
// provide fallback behavior when the user configures no push destinations.
type nopPublisher struct {
	repoName string
	namer    publish.Namer
	tag      string
	tagOnly  bool
}

func (n nopPublisher) Publish(_ context.Context, br build.Result, s string) (name.Reference, error) {
	s = strings.TrimPrefix(s, build.StrictScheme)
	nm := n.namer(n.repoName, s)
	if n.tagOnly {
		if n.tag == "" {
			return nil, errors.New("must specify tag if requesting tag only")
		}
		return name.NewTag(fmt.Sprintf("%s:%s", nm, n.tag))
	}
	h, err := br.Digest()
	if err != nil {
		return nil, err
	}
	if n.tag == "" {
		return name.NewDigest(fmt.Sprintf("%s@%s", nm, h))
	}
	return name.NewDigest(fmt.Sprintf("%s:%s@%s", nm, n.tag, h))
}

func (n nopPublisher) Close() error { return nil }

// resolvedFuture represents a "future" for the bytes of a resolved file.
type resolvedFuture chan []byte

func ResolveFilesToWriter(
	ctx context.Context,
	builder *build.Caching,
	publisher publish.Interface,
	fo *options.FilenameOptions,
	so *options.SelectorOptions,
	out io.WriteCloser) error {
	defer out.Close()

	// By having this as a channel, we can hook this up to a filesystem
	// watcher and leave `fs` open to stream the names of yaml files
	// affected by code changes (including the modification of existing or
	// creation of new yaml files).
	fs := options.EnumerateFiles(fo)

	// This tracks filename -> []importpath
	var sm sync.Map

	// This tracks resolution errors and ensures we cancel other builds if an
	// individual build fails.
	errs, ctx := errgroup.WithContext(ctx)

	var futures []resolvedFuture
	for {
		// Each iteration, if there is anything in the list of futures,
		// listen to it in addition to the file enumerating channel.
		// A nil channel is never available to receive on, so if nothing
		// is available, this will result in us exclusively selecting
		// on the file enumerating channel.
		var bf resolvedFuture
		if len(futures) > 0 {
			bf = futures[0]
		} else if fs == nil {
			// There are no more files to enumerate and the futures
			// have been drained, so quit.
			break
		}

		select {
		case file, ok := <-fs:
			if !ok {
				// a nil channel is never available to receive on.
				// This allows us to drain the list of in-process
				// futures without this case of the select winning
				// each time.
				fs = nil
				break
			}

			// Make a new future to use to ship the bytes back and append
			// it to the list of futures (see comment below about ordering).
			ch := make(resolvedFuture)
			futures = append(futures, ch)

			// Kick off the resolution that will respond with its bytes on
			// the future.
			f := file // defensive copy
			errs.Go(func() error {
				defer close(ch)
				// Record the builds we do via this builder.
				recordingBuilder := &build.Recorder{
					Builder: builder,
				}
				b, err := resolveFile(ctx, f, recordingBuilder, publisher, so)
				if err != nil {
					// This error is sometimes expected during watch mode, so this
					// isn't fatal. Just print it and keep the watch open.
					return fmt.Errorf("error processing import paths in %q: %w", f, err)
				}
				// Associate with this file the collection of binary import paths.
				sm.Store(f, recordingBuilder.ImportPaths)
				ch <- b
				return nil
			})

		case b, ok := <-bf:
			// Once the head channel returns something, dequeue it.
			// We listen to the futures in order to be respectful of
			// the kubectl apply ordering, which matters!
			futures = futures[1:]
			if ok {
				// Write the next body and a trailing delimiter.
				// We write the delimiter LAST so that when streamed to
				// kubectl it knows that the resource is complete and may
				// be applied.
				out.Write(append(b, []byte("\n---\n")...))
			}
		}
	}

	// Make sure we exit with an error.
	// See https://github.com/ko-build/ko/issues/84
	return errs.Wait()
}

func resolveFile(
	ctx context.Context,
	f string,
	builder build.Interface,
	pub publish.Interface,
	so *options.SelectorOptions) (b []byte, err error) {
	var selector labels.Selector
	if so.Selector != "" {
		var err error
		selector, err = labels.Parse(so.Selector)

		if err != nil {
			return nil, fmt.Errorf("unable to parse selector: %w", err)
		}
	}

	if f == "-" {
		b, err = io.ReadAll(os.Stdin)
	} else {
		b, err = os.ReadFile(f)
	}
	if err != nil {
		return nil, err
	}

	var docNodes []*yaml.Node

	// The loop is to support multi-document yaml files.
	// This is handled by using a yaml.Decoder and reading objects until io.EOF, see:
	// https://godoc.org/gopkg.in/yaml.v3#Decoder.Decode
	decoder := yaml.NewDecoder(bytes.NewBuffer(b))
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		if selector != nil {
			if match, err := resolve.MatchesSelector(&doc, selector); err != nil {
				return nil, fmt.Errorf("error evaluating selector: %w", err)
			} else if !match {
				continue
			}
		}

		docNodes = append(docNodes, &doc)
	}

	if err := resolve.ImageReferences(ctx, docNodes, builder, pub); err != nil {
		return nil, fmt.Errorf("error resolving image references: %w", err)
	}

	buf := &bytes.Buffer{}
	e := yaml.NewEncoder(buf)
	e.SetIndent(2)

	for _, doc := range docNodes {
		err := e.Encode(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to encode output: %w", err)
		}
	}
	e.Close()

	return buf.Bytes(), nil
}

// create a set from the input slice
// preserving the order of unique elements
func unique(ss []string) []string {
	var (
		seen = make(map[string]struct{}, len(ss))
		uniq = make([]string, 0, len(ss))
	)
	for _, s := range ss {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			uniq = append(uniq, s)
		}
	}
	return uniq
}
