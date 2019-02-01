package main

import (
	"fmt"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"strings"
)

func main() {
	fmt.Println(strings.ReplaceAll(latest.Version, "skaffold/", ""))
}
