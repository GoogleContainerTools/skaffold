// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package cache

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/mitchellh/go-homedir"
	"github.com/sirupsen/logrus"

	ecrconfig "github.com/awslabs/amazon-ecr-credential-helper/ecr-login/config"
)

func BuildCredentialsCache(config aws.Config, cacheDir string) CredentialsCache {
	if os.Getenv("AWS_ECR_DISABLE_CACHE") != "" {
		logrus.Debug("Cache disabled due to AWS_ECR_DISABLE_CACHE")
		return NewNullCredentialsCache()
	}

	if cacheDir == "" {
		//Get cacheDir from env var "AWS_ECR_CACHE_DIR" or set to default
		cacheDir = ecrconfig.GetCacheDir()
	}

	cacheDir, err := homedir.Expand(cacheDir)
	if err != nil {
		logrus.WithError(err).Debug("Could not expand cache path, disabling cache")
		return NewNullCredentialsCache()
	}

	cacheFilename := "cache.json"

	credentials, err := config.Credentials.Retrieve(context.TODO())
	if err != nil {
		logrus.WithError(err).Debug("Could not fetch credentials for cache prefix, disabling cache")
		return NewNullCredentialsCache()
	}

	return NewFileCredentialsCache(cacheDir, cacheFilename, credentialsCachePrefix(config.Region, credentials), credentialsPublicCacheKey(credentials))
}

// Determine a key prefix for a credentials cache. Because auth tokens are scoped to an account and region, rely on provided
// region, as well as hash of the access key.
func credentialsCachePrefix(region string, credentials aws.Credentials) string {
	return fmt.Sprintf("%s-%s-", region, checksum(credentials.AccessKeyID))
}

func credentialsPublicCacheKey(credentials aws.Credentials) string {
	return fmt.Sprintf("%s-%s", ServiceECRPublic, checksum(credentials.AccessKeyID))
}

// Base64 encodes an MD5 checksum. Relied on for uniqueness, and not for cryptographic security.
func checksum(text string) string {
	hasher := md5.New()
	data := hasher.Sum([]byte(text))
	return base64.StdEncoding.EncodeToString(data)
}
