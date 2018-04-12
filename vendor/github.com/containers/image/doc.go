// Package image provides libraries and commands to interact with containers images.
//
// 	package main
//
// 	import (
// 		"fmt"
//
// 		"github.com/containers/image/docker"
// 	)
//
// 	func main() {
// 		ref, err := docker.ParseReference("//fedora")
// 		if err != nil {
// 			panic(err)
// 		}
// 		img, err := ref.NewImage(nil)
// 		if err != nil {
// 			panic(err)
// 		}
// 		defer img.Close()
// 		b, _, err := img.Manifest()
// 		if err != nil {
// 			panic(err)
// 		}
// 		fmt.Printf("%s", string(b))
// 	}
//
// TODO(runcom)
package image
