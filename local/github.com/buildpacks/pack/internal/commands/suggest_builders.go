package commands

import (
	"fmt"
	"sort"
	"sync"
	"text/tabwriter"

	"github.com/spf13/cobra"

	bldr "github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

// Deprecated: Use `builder suggest` instead.
func SuggestBuilders(logger logging.Logger, inspector BuilderInspector) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "suggest-builders",
		Hidden:  true,
		Args:    cobra.NoArgs,
		Short:   "Display list of recommended builders",
		Example: "pack suggest-builders",
		Run: func(cmd *cobra.Command, s []string) {
			deprecationWarning(logger, "suggest-builder", "builder suggest")
			suggestBuilders(logger, inspector)
		},
	}

	return cmd
}

func suggestSettingBuilder(logger logging.Logger, inspector BuilderInspector) {
	logger.Info("Please select a default builder with:")
	logger.Info("")
	logger.Info("\tpack config default-builder <builder-image>")
	logger.Info("")
	suggestBuilders(logger, inspector)
}

func suggestBuilders(logger logging.Logger, client BuilderInspector) {
	suggestedBuilders := []bldr.KnownBuilder{}
	for _, knownBuilder := range bldr.KnownBuilders {
		if knownBuilder.Suggested {
			suggestedBuilders = append(suggestedBuilders, knownBuilder)
		}
	}
	WriteSuggestedBuilder(logger, client, suggestedBuilders)
}

func WriteSuggestedBuilder(logger logging.Logger, inspector BuilderInspector, builders []bldr.KnownBuilder) {
	sort.Slice(builders, func(i, j int) bool {
		if builders[i].Vendor == builders[j].Vendor {
			return builders[i].Image < builders[j].Image
		}

		return builders[i].Vendor < builders[j].Vendor
	})

	logger.Info("Suggested builders:")

	// Fetch descriptions concurrently.
	descriptions := make([]string, len(builders))

	var wg sync.WaitGroup
	wg.Add(len(builders))

	for i, builder := range builders {
		go func(w *sync.WaitGroup, i int, builder bldr.KnownBuilder) {
			descriptions[i] = getBuilderDescription(builder, inspector)
			w.Done()
		}(&wg, i, builder)
	}

	wg.Wait()

	tw := tabwriter.NewWriter(logger.Writer(), 10, 10, 5, ' ', tabwriter.TabIndent)
	for i, builder := range builders {
		fmt.Fprintf(tw, "\t%s:\t%s\t%s\t\n", builder.Vendor, style.Symbol(builder.Image), descriptions[i])
	}
	fmt.Fprintln(tw)

	logging.Tip(logger, "Learn more about a specific builder with:")
	logger.Info("\tpack builder inspect <builder-image>")
}

func getBuilderDescription(builder bldr.KnownBuilder, inspector BuilderInspector) string {
	info, err := inspector.InspectBuilder(builder.Image, false)
	if err == nil && info != nil && info.Description != "" {
		return info.Description
	}

	return builder.DefaultDescription
}
