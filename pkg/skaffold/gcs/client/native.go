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

var GetBucketManager = getBucketManager

// bucketHandler defines the available interactions with a GCS bucket.
type bucketHandler interface {
	// ListObjects lists the objects that match the given query.
	ListObjects(ctx context.Context, q *storage.Query) ([]string, error)
	// DownloadObject downloads the object with the given uri in the localPath.
	DownloadObject(ctx context.Context, localPath, uri string) error
	// UploadObject creates a files with the given content with the objName.
	UploadObject(ctx context.Context, objName string, content *os.File) error
	// Close closes the bucket handler connection.
	Close()
}

// uriInfo contains information about the GCS object URI.
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
	uriInfo, err := n.parseGCSURI(src)
	if err != nil {
		return err
	}

	bucket, err := GetBucketManager(ctx, uriInfo.Bucket)
	if err != nil {
		return err
	}
	defer bucket.Close()

	files, err := n.filesToDownload(ctx, bucket, uriInfo)
	if err != nil {
		return err
	}

	for uri, localPath := range files {
		fullPath := filepath.Join(dst, localPath)
		dir := filepath.Dir(fullPath)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return fmt.Errorf("failed to create directory: %v", err)
			}
		}

		if err := bucket.DownloadObject(ctx, fullPath, uri); err != nil {
			return err
		}
	}

	return nil
}

// Uploads a single file to the given dst.
func (n *Native) UploadFile(ctx context.Context, src, dst string) error {
	f, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer f.Close()

	urinfo, err := n.parseGCSURI(dst)
	if err != nil {
		return err
	}

	bucket, err := GetBucketManager(ctx, urinfo.Bucket)
	if err != nil {
		return err
	}

	isDirectory, err := n.isGCSDirectory(ctx, bucket, urinfo)
	if err != nil {
		return err
	}

	dstObj := urinfo.ObjPath
	if isDirectory {
		dstObj, err = url.JoinPath(dstObj, filepath.Base(src))
		if err != nil {
			return err
		}
	}

	return bucket.UploadObject(ctx, dstObj, f)
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
	// If we do this with the url package it will scape the `?` character, breaking the glob.
	gcsobj.ObjPath = strings.TrimLeft(strings.ReplaceAll(uri, "gs://"+u.Host, ""), "/")

	return gcsobj, nil
}

func (n *Native) filesToDownload(ctx context.Context, bucket bucketHandler, urinfo uriInfo) (map[string]string, error) {
	uriToLocalPath := map[string]string{}

	// The exact match is with the original glob expression. This could be:
	// 1. a/b/c -> It will return the file `c` under a/b/ if it exists
	// 2. a/b/c* -> It will return any file under a/b/ that starts with c, e.g, c1, c-other, etc
	// 3. a/b/c** -> It will return any file that starts with 'c', and files inside any folder starting with 'c'. It is doing the recursion already
	exactMatches, err := bucket.ListObjects(ctx, &storage.Query{MatchGlob: urinfo.ObjPath})
	if err != nil {
		return nil, err
	}

	for _, match := range exactMatches {
		uriToLocalPath[match] = filepath.Base(match)
	}

	// Then, to mimic gsutil behavior, we assume the last part of the glob is a folder, so we complete the
	// URI with the necessary wildcard to list the folder recursively.
	recursiveMatches, err := n.recursiveListing(ctx, bucket, urinfo)
	if err != nil {
		return nil, err
	}

	for uri, match := range recursiveMatches {
		uriToLocalPath[uri] = match
	}

	return uriToLocalPath, nil
}

func (n *Native) recursiveListing(ctx context.Context, bucket bucketHandler, urinfo uriInfo) (map[string]string, error) {
	uriToLocalPath := map[string]string{}
	recursiveURI := n.uriForRecursiveSearch(urinfo.ObjPath)
	recursiveMatches, err := bucket.ListObjects(ctx, &storage.Query{MatchGlob: recursiveURI})
	if err != nil {
		return nil, err
	}

	prefixRemovalURI := n.uriForPrefixRemoval(urinfo.Full())
	prefixRemovalRegex, err := n.wildcardToRegex(prefixRemovalURI)
	if err != nil {
		return nil, err
	}

	// For glob patterns that have `**` (anywhere), gsutil doesn't recreate the folder structure.
	shouldRecreateFolders := !strings.Contains(urinfo.ObjPath, "**")
	for _, match := range recursiveMatches {
		destPath := filepath.Base(match)
		if shouldRecreateFolders {
			matchWithBucket := urinfo.Bucket + "/" + match
			destPath = string(prefixRemovalRegex.ReplaceAll([]byte(matchWithBucket), []byte("")))
		}
		uriToLocalPath[match] = destPath
	}

	return uriToLocalPath, nil
}

// uriForRecursiveSearch returns a modified URI is to cover globs like a/*/d*, to remove its prefix:
// For the case where the bucket has the following files:
// - a/b/d/sub1/file1
// - a/c/d/sub2/sub3/file2
// - a/e/d2/file2
// The resulting files + folders should be:
// - d/sub1/file1
// - d/sub2/sub3/file2
// - d2/file2
func (n *Native) uriForRecursiveSearch(uri string) string {
	// when we want to list all the bucket
	if uri == "" {
		return "**"
	}
	// uri is a/b** or a/b/**
	if strings.HasSuffix(uri, "**") {
		return uri
	}
	// a/b* and a/b/* become a/b** and a/b/**
	if strings.HasSuffix(uri, "*") {
		return uri + "*"
	}
	// a/b/ becomes a/b/**
	if strings.HasSuffix(uri, "/") {
		return uri + "**"
	}
	// a/b becomes a/b/**
	return uri + "/**"
}

func (n *Native) uriForPrefixRemoval(uri string) string {
	if strings.HasSuffix(uri, "/*") {
		return strings.TrimSuffix(uri, "*")
	}
	uri = strings.TrimSuffix(uri, "/")
	idx := strings.LastIndex(uri, "/")
	return uri[:idx+1]
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

func (n *Native) isGCSDirectory(ctx context.Context, bucket bucketHandler, urinfo uriInfo) (bool, error) {
	if urinfo.ObjPath == "" {
		return true, nil
	}

	if strings.HasSuffix(urinfo.ObjPath, "/") {
		return true, nil
	}

	q := &storage.Query{Prefix: urinfo.ObjPath + "/"}
	// GCS doesn't support empty "folders".
	matches, err := bucket.ListObjects(ctx, q)
	if err != nil {
		return false, err
	}

	if len(matches) > 0 {
		return true, nil
	}

	return false, nil
}

func getBucketManager(ctx context.Context, bucketName string) (bucketHandler, error) {
	sc, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("error creating GCS Client: %w", err)
	}

	return nativeBucketHandler{
		storageClient: sc,
		bucket:        sc.Bucket(bucketName),
	}, nil
}

// nativeBucketHandler implements a handler using the Cloud client libraries.
type nativeBucketHandler struct {
	storageClient *storage.Client
	bucket        *storage.BucketHandle
}

func (nb nativeBucketHandler) ListObjects(ctx context.Context, q *storage.Query) ([]string, error) {
	matches := []string{}
	it := nb.bucket.Objects(ctx, q)

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

func (nb nativeBucketHandler) DownloadObject(ctx context.Context, localPath, uri string) error {
	reader, err := nb.bucket.Object(uri).NewReader(ctx)
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

func (nb nativeBucketHandler) UploadObject(ctx context.Context, objName string, content *os.File) error {
	wc := nb.bucket.Object(objName).NewWriter(ctx)
	if _, err := io.Copy(wc, content); err != nil {
		return fmt.Errorf("error copying file to GCS: %w", err)
	}
	if err := wc.Close(); err != nil {
		return fmt.Errorf("error closing GCS writer: %w", err)
	}
	return nil
}

func (nb nativeBucketHandler) Close() {
	nb.storageClient.Close()
}
