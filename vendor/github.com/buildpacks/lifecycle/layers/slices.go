package layers

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/archive"
)

type Slice struct {
	Paths []string `toml:"paths"`
}

// SliceLayers divides dir into layers using slices using the following process:
// * Given n slices SliceLayers will return n+1 layers
// * The first n layers will contain files matched by the any Path in the nth Slice
// * The final layer will contain any files in dir that were not included in a previous layer
// Some layers may be empty
func (f *Factory) SliceLayers(dir string, slices []Slice) ([]Layer, error) {
	var sliceLayers []Layer
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	sdir, err := newSlicableDir(dir)
	if err != nil {
		return nil, err
	}

	// add one layer per slice
	for i, slice := range slices {
		layerID := fmt.Sprintf("slice-%d", i+1)
		layer, err := f.createLayerFromSlice(slice, sdir, layerID)
		if err != nil {
			return nil, err
		}
		sliceLayers = append(sliceLayers, layer)
	}

	// add remaining files in a single layer
	layerID := fmt.Sprintf("slice-%d", len(slices)+1)
	finalLayer, err := f.createLayerFromFiles(layerID, sdir, sdir.remainingFiles())
	if err != nil {
		return nil, err
	}
	return append(sliceLayers, finalLayer), nil
}

func (f *Factory) createLayerFromSlice(slice Slice, sdir *sliceableDir, layerID string) (Layer, error) {
	var matches []string
	for _, path := range slice.Paths {
		globMatches, err := glob(sdir, path)
		if err != nil {
			return Layer{}, err
		}
		matches = append(matches, globMatches...)
	}
	return f.createLayerFromFiles(layerID, sdir, sdir.sliceFiles(matches))
}

func glob(sdir *sliceableDir, pattern string) ([]string, error) {
	pattern = filepath.Clean(pattern)
	var matches []string
	if err := filepath.Walk(sdir.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == sdir.path {
			return nil
		}
		relPath, err := filepath.Rel(sdir.path, path)
		if err != nil {
			return err
		}
		match, err := filepath.Match(pattern, relPath)
		if err != nil {
			return errors.Wrapf(err, "failed to check if '%s' matches '%s'", relPath, pattern)
		}
		if match {
			matches = append(matches, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return matches, nil
}

func (f *Factory) createLayerFromFiles(layerID string, sdir *sliceableDir, files []archive.PathInfo) (layer Layer, err error) {
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return f.writeLayer(layerID, func(tw *archive.NormalizingTarWriter) error {
		if len(files) != 0 {
			if err := archive.AddFilesToArchive(tw, sdir.parentDirs); err != nil {
				return err
			}
			tw.WithUID(f.UID)
			tw.WithGID(f.GID)
			return archive.AddFilesToArchive(tw, files)
		}
		return nil
	})
}

type sliceableDir struct {
	path        string                 // path to slicableDir
	slicedFiles map[string]bool        // map showing which paths are already sliced
	pathInfos   map[string]os.FileInfo // map of path to file info
	subDirs     map[string][]string    // map dirs to children
	parentDirs  []archive.PathInfo     // parents of the slicableDir
}

func newSlicableDir(appDir string) (*sliceableDir, error) {
	sdir := &sliceableDir{
		path:        appDir,
		slicedFiles: map[string]bool{},
		pathInfos:   map[string]os.FileInfo{},
		subDirs:     map[string][]string{},
	}
	if err := filepath.Walk(appDir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		sdir.slicedFiles[path] = false
		sdir.pathInfos[path] = fi
		if fi.IsDir() {
			children, err := ioutil.ReadDir(path)
			if err != nil {
				return err
			}
			for _, child := range children {
				sdir.subDirs[path] = append(sdir.subDirs[path], filepath.Join(path, child.Name()))
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	parentDirs, err := parents(appDir)
	if err != nil {
		return nil, err
	}
	sdir.parentDirs = parentDirs
	return sdir, nil
}

func (sd *sliceableDir) sliceFiles(paths []string) []archive.PathInfo {
	slicedFiles := map[string]os.FileInfo{}
	for _, match := range paths {
		sd.addMatchedFiles(slicedFiles, match)
	}
	return sd.fillInMissingParents(slicedFiles)
}

func (sd *sliceableDir) addMatchedFiles(matchedFiles map[string]os.FileInfo, match string) {
	if added, ok := sd.slicedFiles[match]; !ok || added {
		// don't add files that live outside the app dir
		// don't add files were already added
		return
	}
	if children, ok := sd.subDirs[match]; ok {
		for _, child := range children {
			sd.addMatchedFiles(matchedFiles, child)
		}
	}
	matchedFiles[match] = sd.pathInfos[match]
	sd.slicedFiles[match] = true
}

func (sd *sliceableDir) fillInMissingParents(matchedFiles map[string]os.FileInfo) []archive.PathInfo {
	// add parent dirs for matched files if they are missing
	var parentToCheck []string
	var files []archive.PathInfo
	addedParents := map[string]struct{}{}
	for path, info := range matchedFiles {
		files = append(files, archive.PathInfo{
			Path: path,
			Info: info,
		})
		parents := sd.fileParents(path)
		for _, parent := range parents {
			if _, ok := matchedFiles[parent.Path]; ok {
				// don't add if it was slices as part of a match
				continue
			}
			if _, ok := addedParents[parent.Path]; ok {
				// don't add if it was already added
				continue
			}
			parentToCheck = append(parentToCheck, parent.Path)
			addedParents[parent.Path] = struct{}{}
			files = append(files, parent)
		}
	}

	// sort the dirs by their path depth (deepest -> most shallow) so we always process children first
	sort.SliceStable(parentToCheck, func(i, j int) bool {
		return len(strings.Split(parentToCheck[i], string(os.PathSeparator))) > len(strings.Split(parentToCheck[j], string(os.PathSeparator)))
	})

	// if all children are slices, mark the dir as sliced
	for _, dir := range parentToCheck {
		allChildrenAdded := true
		if children, ok := sd.subDirs[dir]; ok {
			for _, child := range children {
				if added, ok := sd.slicedFiles[child]; ok && !added {
					allChildrenAdded = false
					break
				}
			}
		}
		sd.slicedFiles[dir] = allChildrenAdded
	}
	return files
}

func (sd *sliceableDir) remainingFiles() []archive.PathInfo {
	var files []archive.PathInfo
	for path, info := range sd.pathInfos {
		if added, ok := sd.slicedFiles[path]; !ok || added {
			continue
		}
		files = append(files, archive.PathInfo{
			Path: path,
			Info: info,
		})
		sd.slicedFiles[path] = true
	}
	return files
}

// return parents within the sliceableDir
func (sd *sliceableDir) fileParents(file string) []archive.PathInfo {
	parent := filepath.Dir(file)
	if parent == sd.path {
		return []archive.PathInfo{
			{Path: sd.path, Info: sd.pathInfos[sd.path]},
		}
	}
	return append(sd.fileParents(parent), archive.PathInfo{
		Path: parent,
		Info: sd.pathInfos[parent],
	})
}
