# semgroup [![](https://github.com/fatih/semgroup/workflows/build/badge.svg)](https://github.com/fatih/semgroup/actions) [![PkgGoDev](https://pkg.go.dev/badge/github.com/fatih/semgroup)](https://pkg.go.dev/github.com/fatih/semgroup)

semgroup provides synchronization and error propagation, for groups of goroutines working on subtasks of a common task. It uses a weighted semaphore implementation to make sure that only a number of maximum tasks can be run at any time.

Unlike [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup), it doesn't return the first non-nil error, rather it accumulates all errors and returns a set of errors, allowing each task to fullfil their task. 


# Install

```bash
go get github.com/fatih/semgroup
```

# Example

With no errors:

```go
package main

import (
	"context"
	"fmt"

	"github.com/fatih/semgroup"
)

func main() {
	const maxWorkers = 2
	s := semgroup.NewGroup(context.Background(), maxWorkers)

	visitors := []int{5, 2, 10, 8, 9, 3, 1}

	for _, v := range visitors {
		v := v

		s.Go(func() error {
			fmt.Println("Visits: ", v)
			return nil
		})
	}

	// Wait for all visits to complete. Any errors are accumulated.
	if err := s.Wait(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// Visits: 2
	// Visits: 10
	// Visits: 8
	// Visits: 9
	// Visits: 3
	// Visits: 1
	// Visits: 5
}
```

With errors:


```go
package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/fatih/semgroup"
)

func main() {
	const maxWorkers = 2
	s := semgroup.NewGroup(context.Background(), maxWorkers)

	visitors := []int{1, 1, 1, 1, 2, 2, 1, 1, 2}

	for _, v := range visitors {
		v := v

		s.Go(func() error {
			if v != 1 {
				return errors.New("only one visitor is allowed")
			}
			return nil
		})
	}

	// Wait for all visits to complete. Any errors are accumulated.
	if err := s.Wait(); err != nil {
		fmt.Println(err)
	}

	// Output:
	// 3 error(s) occurred:
	// * only one visitor is allowed
	// * only one visitor is allowed
	// * only one visitor is allowed
}
```

