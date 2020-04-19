package util

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/sirupsen/logrus"
)

const GsutilExec = "gsutil"

type Gsutil struct{}

// Copy calls `gsutil cp -r <source_url> <destination_url>
func (g *Gsutil) Copy(ctx context.Context, src, dst string, recursive bool) error {
	if _, err := exec.LookPath(GsutilExec); err != nil {
		return err
	}
	args := []string{"cp", "-r", src, dst}
	// remove the -r flag
	if !recursive {
		args = append(args[:1], args[2:]...)
	}
	cmd := exec.CommandContext(ctx, GsutilExec, args...)
	out, err := RunCmdOut(cmd)
	if err != nil {
		return fmt.Errorf("copy file(s) with %s failed: %v", GsutilExec, err.Error())
	}
	logrus.Info(out)
	return nil
}
