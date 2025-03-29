package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	basePackage := os.Args[1]
	src := os.Args[2]

	messages, err := collectImportErrors(src, basePackage)
	if err != nil {
		log.Fatal(err)
	}

	for _, message := range messages {
		fmt.Println(message)
	}

	if len(messages) > 0 {
		os.Exit(1)
	}
}

// collectImportErrors runs in addition to `gofmt` to check that imports are properly organized in groups:
// + there's at most 2 blank lines between imports,
// + the `github.com/buildpacks/pack` imports must come in the last import group.
func collectImportErrors(root, basePackage string) ([]string, error) {
	var list []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			if isIgnoredDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasSuffix(info.Name(), ".go") {
			messages, err := checkImports(path, basePackage)
			if err != nil {
				return err
			}

			list = append(list, messages...)
		}

		return nil
	})

	return list, err
}

func isIgnoredDir(name string) bool {
	return name == "vendor" ||
		(strings.HasPrefix(".", name) && name != ".")
}

func checkImports(path, basePackage string) ([]string, error) {
	file, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var (
		inImport   bool
		blankLines int
		last       string
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "import (") {
			inImport = true
		} else if inImport {
			if line == "" {
				blankLines++
			} else if line == ")" {
				break
			} else {
				last = strings.TrimSpace(line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var messages []string

	if blankLines == 2 {
		if !strings.Contains(last, basePackage) {
			messages = append(messages, fmt.Sprintf("%q must have pack imports last", path))
		}
	} else if blankLines > 2 {
		messages = append(messages, fmt.Sprintf("%q contains more than 3 groups of imports", path))
	}

	return messages, nil
}
