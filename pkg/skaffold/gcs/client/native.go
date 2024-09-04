/*
Copyright 2024 The Skaffold Authors

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

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

// Obj contains information about the GCS object.
type uriInfo struct {
	// Bucket is the name of the GCS bucket.
	Bucket string

	// ObjPath is the path, with or without wildcards, of the specified object(s) in the GCS bucket.
	ObjPath string
}

func (o uriInfo) Full() string {
	return o.Bucket + "/" + o.ObjPath
}

type Native struct{}

// Downloads the content that match the given src uri and subfolders.
func (n *Native) DownloadRecursive(ctx context.Context, src, dst string) error {
	sc, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating GCS Client: %w", err)
	}
	defer sc.Close()

	uriInfo, err := n.parseGCSURI(src)
	if err != nil {
		return err
	}
	bucket := sc.Bucket(uriInfo.Bucket)

	files, err := n.filesToDownload(ctx, bucket, uriInfo)
	if err != nil {
		return err
	}

	for uri, localPath := range files {
		fullPath := filepath.Join(dst, localPath)
		if err := n.downloadFile(ctx, bucket, fullPath, uri); err != nil {
			return err
		}
	}

	return nil
}

// Uploads a single file to the given dst.
func (n *Native) UploadFile(ctx context.Context, src, dst string) error {
	sc, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("error creating GCS Client: %w", err)
	}
	defer sc.Close()

	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	uinfo, err := n.parseGCSURI(dst)
	if err != nil {
		return err
	}
	bucket := sc.Bucket(uinfo.Bucket)

	isDirectory, err := n.isGCSDirectory(ctx, bucket, uinfo)
	if err != nil {
		return err
	}

	dstObj := uinfo.ObjPath
	if isDirectory {
		dstObj, err = url.JoinPath(dstObj, filepath.Base(src))
		if err != nil {
			return err
		}
	}

	wc := bucket.Object(dstObj).NewWriter(ctx)
	if _, err = io.Copy(wc, f); err != nil {
		return fmt.Errorf("error copying file to GCS: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("error closing GCS writer: %w", err)
	}
	return nil
}

func (n *Native) parseGCSURI(uri string) (uriInfo, error) {
	var gcsobj uriInfo
	u, err := url.Parse(uri)
	if err != nil {
		return uriInfo{}, fmt.Errorf("cannot parse URI %q: %w", uri, err)
	}
	if u.Scheme != "gs" {
		return uriInfo{}, fmt.Errorf("URI scheme is %q, must be 'gs'", u.Scheme)
	}
	if u.Host == "" {
		return uriInfo{}, errors.New("bucket name is empty")
	}
	gcsobj.Bucket = u.Host
	gcsobj.ObjPath = strings.TrimLeft(strings.ReplaceAll(uri, "gs://"+u.Host, ""), "/")

	return gcsobj, nil
}

func (n *Native) filesToDownload(ctx context.Context, bucket *storage.BucketHandle, uinfo uriInfo) (map[string]string, error) {
	uriToLocalPath := map[string]string{}

	exactMatches, err := n.listObjects(ctx, bucket, &storage.Query{MatchGlob: uinfo.ObjPath})
	if err != nil {
		return nil, err
	}

	for _, match := range exactMatches {
		uriToLocalPath[match] = filepath.Base(match)
	}

	recursiveMatches, err := n.recursiveListing(ctx, bucket, uinfo)
	if err != nil {
		return nil, err
	}

	for _, match := range recursiveMatches {
		uriToLocalPath[match] = match
	}

	return uriToLocalPath, nil
}

func (n *Native) listObjects(ctx context.Context, bucket *storage.BucketHandle, q *storage.Query) ([]string, error) {
	matches := []string{}
	it := bucket.Objects(ctx, q)

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to iterate objects: %v", err)
		}

		if attrs.Name != "" {
			matches = append(matches, attrs.Name)
		}
	}
	return matches, nil
}

func (n *Native) recursiveListing(ctx context.Context, bucket *storage.BucketHandle, uinfo uriInfo) (map[string]string, error) {
	uriToLocalPath := map[string]string{}
	recursiveURI := n.uriForRecursiveSearch(uinfo.ObjPath)
	recursiveMatches, err := n.listObjects(ctx, bucket, &storage.Query{MatchGlob: recursiveURI})
	if err != nil {
		return nil, err
	}

	prefixRemovalURI := n.uriForPrefixRemoval(uinfo.Full())
	prefixRemovalRegex, err := n.wildcardToRegex(prefixRemovalURI)
	if err != nil {
		return nil, err
	}

	shouldRecreateFolders := !strings.Contains(uinfo.ObjPath, "**")
	for _, match := range recursiveMatches {
		destPath := filepath.Base(match)
		if shouldRecreateFolders {
			matchWithBucket := uinfo.Bucket + "/" + match
			destPath = string(prefixRemovalRegex.ReplaceAll([]byte(matchWithBucket), []byte("")))
		}
		uriToLocalPath[match] = destPath
	}

	return uriToLocalPath, nil
}

func (n *Native) uriForRecursiveSearch(src string) string {
	// when we want to list all the bucket
	if src == "" {
		return "**"
	}

	// a/b** or a/b/**
	if strings.HasSuffix(src, "**") {
		return src
	}

	// a/b* and a/b/* become a/b** and a/b/**
	if strings.HasSuffix(src, "*") {
		return src + "*"
	}
	// a/b/ becomes a/b/**
	if strings.HasSuffix(src, "/") {
		return src + "**"
	}

	// a/b becomes a/b/**
	return src + "/**"
}

func (n *Native) uriForPrefixRemoval(src string) string {
	if strings.HasSuffix(src, "/*") {
		return strings.TrimSuffix(src, "*")
	}
	src = strings.TrimSuffix(src, "/")
	idx := strings.LastIndex(src, "/")
	return src[:idx+1]
}

func (n *Native) wildcardToRegex(wildcard string) (*regexp.Regexp, error) {
	// Escape special regex characters that might be present in the wildcard
	escaped := regexp.QuoteMeta(wildcard)

	escaped = strings.ReplaceAll(escaped, "\\*", "[^/]*")
	escaped = strings.ReplaceAll(escaped, "\\?", "[^/]") // Match any single character except '/'
	escaped = strings.ReplaceAll(escaped, "\\[", "[")
	escaped = strings.ReplaceAll(escaped, "\\]", "]")
	regexStr := "^" + escaped

	return regexp.Compile(regexStr)
}

func (n *Native) downloadFile(ctx context.Context, bucket *storage.BucketHandle, localPath, uri string) error {
	dir := filepath.Dir(localPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	reader, err := bucket.Object(uri).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("failed to read object: %v", err)
	}
	defer reader.Close()

	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to copy object to file: %v", err)
	}

	return nil
}

func (n *Native) isGCSDirectory(ctx context.Context, bucket *storage.BucketHandle, uinfo uriInfo) (bool, error) {
	if uinfo.ObjPath == "" {
		return true, nil
	}

	if strings.HasSuffix(uinfo.ObjPath, "/") {
		return true, nil
	}

	q := &storage.Query{Prefix: uinfo.ObjPath + "/"}
	matches, err := n.listObjects(ctx, bucket, q)
	if err != nil {
		return false, err
	}

	if len(matches) > 0 {
		return true, nil
	}

	return false, nil
}
