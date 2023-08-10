package cli

import (
	"os"

	"github.com/spf13/pflag"
)

// OverrideDefaultName conditionally allows overriding the default cluster name
// by setting the KIND_CLUSTER_NAME environment variable
// only if --name wasn't set explicitly
func OverrideDefaultName(fs *pflag.FlagSet) {
	if !fs.Changed("name") {
		if name := os.Getenv("KIND_CLUSTER_NAME"); name != "" {
			_ = fs.Set("name", name)
		}
	}
}
