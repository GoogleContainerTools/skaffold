// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use: "licenses",
	}

	// Flags shared between subcommands
	confidenceThreshold float64
)

func init() {
	rootCmd.PersistentFlags().Float64Var(&confidenceThreshold, "confidence_threshold", 0.9, "Minimum confidence required in order to positively identify a license.")
}

func main() {
	flag.Parse()
	rootCmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)

	if err := rootCmd.Execute(); err != nil {
		glog.Exit(err)
	}
}

// Unvendor removes the "*/vendor/" prefix from the given import path, if present.
func unvendor(importPath string) string {
	if vendorerAndVendoree := strings.SplitN(importPath, "/vendor/", 2); len(vendorerAndVendoree) == 2 {
		return vendorerAndVendoree[1]
	}
	return importPath
}
