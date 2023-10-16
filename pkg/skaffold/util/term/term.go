/*
Copyright 2020 The Skaffold Authors

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

package term

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/constants"
	"github.com/GoogleContainerTools/skaffold/v2/pkg/skaffold/util"
)

func IsTerminal(w io.Writer) (uintptr, bool) {
	type descriptor interface {
		Fd() uintptr
	}

	if f, ok := w.(descriptor); ok {
		termFd := f.Fd()
		isTerm := term.IsTerminal(int(termFd))
		return termFd, isTerm
	}

	return 0, false
}

func SupportsColor(ctx context.Context) (bool, error) {
	if runtime.GOOS == constants.Windows {
		return true, nil
	}

	cmd := exec.Command("tput", "colors")
	res, err := util.RunCmdOut(ctx, cmd)
	if err != nil {
		return false, fmt.Errorf("checking terminal colors: %w", err)
	}

	numColors, err := strconv.Atoi(strings.TrimSpace(string(res)))
	if err != nil {
		return false, err
	}

	return numColors > 0, nil
}

func WaitForKeyPress() error {
	// use rawMode so that we can read without the user to hit enter key.
	previousState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	defer term.Restore(int(os.Stdin.Fd()), previousState)

	reader := bufio.NewReader(os.Stdin)
	r, _, err := reader.ReadRune()
	if err != nil {
		return err
	}
	// control + c
	if r == 3 {
		return context.Canceled
	}
	return nil
}
