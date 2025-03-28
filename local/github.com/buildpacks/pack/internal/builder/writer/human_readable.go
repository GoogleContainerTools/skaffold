package writer

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"text/template"

	strs "github.com/buildpacks/pack/internal/strings"
	"github.com/buildpacks/pack/pkg/client"

	"github.com/buildpacks/pack/internal/style"

	"github.com/buildpacks/pack/pkg/dist"

	pubbldr "github.com/buildpacks/pack/builder"

	"github.com/buildpacks/pack/internal/config"

	"github.com/buildpacks/pack/internal/builder"
	"github.com/buildpacks/pack/pkg/logging"
)

const (
	writerMinWidth     = 0
	writerTabWidth     = 0
	buildpacksTabWidth = 8
	extensionsTabWidth = 8
	defaultTabWidth    = 4
	writerPadChar      = ' '
	writerFlags        = 0
	none               = "(none)"

	outputTemplate = `
{{ if ne .Info.Description "" -}}
Description: {{ .Info.Description }}

{{ end -}}
{{- if ne .Info.CreatedBy.Name "" -}}
Created By:
  Name: {{ .Info.CreatedBy.Name }}
  Version: {{ .Info.CreatedBy.Version }}

{{ end -}}

Trusted: {{.Trusted}}

{{ if ne .Info.Stack "" -}}Stack:
  ID: {{ .Info.Stack }}{{ end -}}
{{- if .Verbose}}
{{- if ne (len .Info.Mixins) 0 }}
  Mixins:
{{- end }}
{{- range $index, $mixin := .Info.Mixins }}
    {{ $mixin }}
{{- end }}
{{- end }}
{{ .Lifecycle }}
{{ .RunImages }}
{{ .Buildpacks }}
{{ .Order }}
{{- if ne .Extensions "" }}
{{ .Extensions }}
{{- end }}
{{- if ne .OrderExtensions "" }}
{{ .OrderExtensions }}
{{- end }}`
)

type HumanReadable struct{}

func NewHumanReadable() *HumanReadable {
	return &HumanReadable{}
}

func (h *HumanReadable) Print(
	logger logging.Logger,
	localRunImages []config.RunImage,
	local, remote *client.BuilderInfo,
	localErr, remoteErr error,
	builderInfo SharedBuilderInfo,
) error {
	if local == nil && remote == nil {
		return fmt.Errorf("unable to find builder '%s' locally or remotely", builderInfo.Name)
	}

	if builderInfo.IsDefault {
		logger.Infof("Inspecting default builder: %s\n", style.Symbol(builderInfo.Name))
	} else {
		logger.Infof("Inspecting builder: %s\n", style.Symbol(builderInfo.Name))
	}

	logger.Info("\nREMOTE:\n")
	err := writeBuilderInfo(logger, localRunImages, remote, remoteErr, builderInfo)
	if err != nil {
		return fmt.Errorf("writing remote builder info: %w", err)
	}
	logger.Info("\nLOCAL:\n")
	err = writeBuilderInfo(logger, localRunImages, local, localErr, builderInfo)
	if err != nil {
		return fmt.Errorf("writing local builder info: %w", err)
	}

	return nil
}

func writeBuilderInfo(
	logger logging.Logger,
	localRunImages []config.RunImage,
	info *client.BuilderInfo,
	err error,
	sharedInfo SharedBuilderInfo,
) error {
	if err != nil {
		logger.Errorf("%s\n", err)
		return nil
	}

	if info == nil {
		logger.Info("(not present)\n")
		return nil
	}

	var warnings []string

	runImagesString, runImagesWarnings, err := runImagesOutput(info.RunImages, localRunImages, sharedInfo.Name)
	if err != nil {
		return fmt.Errorf("compiling run images output: %w", err)
	}
	orderString, orderWarnings, err := detectionOrderOutput(info.Order, sharedInfo.Name)
	if err != nil {
		return fmt.Errorf("compiling detection order output: %w", err)
	}

	var orderExtString string
	var orderExtWarnings []string

	if info.Extensions != nil {
		orderExtString, orderExtWarnings, err = detectionOrderExtOutput(info.OrderExtensions, sharedInfo.Name)
		if err != nil {
			return fmt.Errorf("compiling detection order extensions output: %w", err)
		}
	}
	buildpacksString, buildpacksWarnings, err := buildpacksOutput(info.Buildpacks, sharedInfo.Name)
	if err != nil {
		return fmt.Errorf("compiling buildpacks output: %w", err)
	}
	lifecycleString, lifecycleWarnings := lifecycleOutput(info.Lifecycle, sharedInfo.Name)

	var extensionsString string
	var extensionsWarnings []string

	if info.Extensions != nil {
		extensionsString, extensionsWarnings, err = extensionsOutput(info.Extensions, sharedInfo.Name)
		if err != nil {
			return fmt.Errorf("compiling extensions output: %w", err)
		}
	}

	warnings = append(warnings, runImagesWarnings...)
	warnings = append(warnings, orderWarnings...)
	warnings = append(warnings, buildpacksWarnings...)
	warnings = append(warnings, lifecycleWarnings...)
	if info.Extensions != nil {
		warnings = append(warnings, extensionsWarnings...)
		warnings = append(warnings, orderExtWarnings...)
	}
	outputTemplate, _ := template.New("").Parse(outputTemplate)

	err = outputTemplate.Execute(
		logger.Writer(),
		&struct {
			Info            client.BuilderInfo
			Verbose         bool
			Buildpacks      string
			RunImages       string
			Order           string
			Trusted         string
			Lifecycle       string
			Extensions      string
			OrderExtensions string
		}{
			*info,
			logger.IsVerbose(),
			buildpacksString,
			runImagesString,
			orderString,
			stringFromBool(sharedInfo.Trusted),
			lifecycleString,
			extensionsString,
			orderExtString,
		},
	)

	for _, warning := range warnings {
		logger.Warn(warning)
	}

	return err
}

type trailingSpaceStrippingWriter struct {
	output io.Writer

	potentialDiscard []byte
}

func (w *trailingSpaceStrippingWriter) Write(p []byte) (n int, err error) {
	var doWrite []byte

	for _, b := range p {
		switch b {
		case writerPadChar:
			w.potentialDiscard = append(w.potentialDiscard, b)
		case '\n':
			w.potentialDiscard = []byte{}
			doWrite = append(doWrite, b)
		default:
			doWrite = append(doWrite, w.potentialDiscard...)
			doWrite = append(doWrite, b)
			w.potentialDiscard = []byte{}
		}
	}

	if len(doWrite) > 0 {
		actualWrote, err := w.output.Write(doWrite)
		if err != nil {
			return actualWrote, err
		}
	}

	return len(p), nil
}

func stringFromBool(subject bool) string {
	if subject {
		return "Yes"
	}

	return "No"
}

func runImagesOutput(
	runImages []pubbldr.RunImageConfig,
	localRunImages []config.RunImage,
	builderName string,
) (string, []string, error) {
	output := "Run Images:\n"

	tabWriterBuf := bytes.Buffer{}

	localMirrorTabWriter := tabwriter.NewWriter(&tabWriterBuf, writerMinWidth, writerTabWidth, defaultTabWidth, writerPadChar, writerFlags)
	err := writeLocalMirrors(localMirrorTabWriter, runImages, localRunImages)
	if err != nil {
		return "", []string{}, fmt.Errorf("writing local mirrors: %w", err)
	}

	var warnings []string

	if len(runImages) == 0 {
		warnings = append(
			warnings,
			fmt.Sprintf("%s does not specify a run image", builderName),
			"Users must build with an explicitly specified run image",
		)
	} else {
		for _, runImage := range runImages {
			if runImage.Image != "" {
				_, err = fmt.Fprintf(localMirrorTabWriter, "  %s\n", runImage.Image)
				if err != nil {
					return "", []string{}, fmt.Errorf("writing to tabwriter: %w", err)
				}
			}
			for _, m := range runImage.Mirrors {
				_, err = fmt.Fprintf(localMirrorTabWriter, "  %s\n", m)
				if err != nil {
					return "", []string{}, fmt.Errorf("writing to tab writer: %w", err)
				}
			}
			err = localMirrorTabWriter.Flush()
			if err != nil {
				return "", []string{}, fmt.Errorf("flushing tab writer: %w", err)
			}
		}
	}
	runImageOutput := tabWriterBuf.String()
	if runImageOutput == "" {
		runImageOutput = fmt.Sprintf("  %s\n", none)
	}

	output += runImageOutput

	return output, warnings, nil
}

func writeLocalMirrors(logWriter io.Writer, runImages []pubbldr.RunImageConfig, localRunImages []config.RunImage) error {
	for _, i := range localRunImages {
		for _, ri := range runImages {
			if i.Image == ri.Image {
				for _, m := range i.Mirrors {
					_, err := fmt.Fprintf(logWriter, "  %s\t(user-configured)\n", m)
					if err != nil {
						return fmt.Errorf("writing local mirror: %s: %w", m, err)
					}
				}
			}
		}
	}

	return nil
}

func extensionsOutput(extensions []dist.ModuleInfo, builderName string) (string, []string, error) {
	output := "Extensions:\n"

	if len(extensions) == 0 {
		return fmt.Sprintf("%s  %s\n", output, none), nil, nil
	}

	var (
		tabWriterBuf         = bytes.Buffer{}
		spaceStrippingWriter = &trailingSpaceStrippingWriter{
			output: &tabWriterBuf,
		}
		extensionsTabWriter = tabwriter.NewWriter(spaceStrippingWriter, writerMinWidth, writerPadChar, extensionsTabWidth, writerPadChar, writerFlags)
	)

	_, err := fmt.Fprint(extensionsTabWriter, "  ID\tNAME\tVERSION\tHOMEPAGE\n")
	if err != nil {
		return "", []string{}, fmt.Errorf("writing to tab writer: %w", err)
	}

	for _, b := range extensions {
		_, err = fmt.Fprintf(extensionsTabWriter, "  %s\t%s\t%s\t%s\n", b.ID, strs.ValueOrDefault(b.Name, "-"), b.Version, strs.ValueOrDefault(b.Homepage, "-"))
		if err != nil {
			return "", []string{}, fmt.Errorf("writing to tab writer: %w", err)
		}
	}

	err = extensionsTabWriter.Flush()
	if err != nil {
		return "", []string{}, fmt.Errorf("flushing tab writer: %w", err)
	}

	output += tabWriterBuf.String()
	return output, []string{}, nil
}

func buildpacksOutput(buildpacks []dist.ModuleInfo, builderName string) (string, []string, error) {
	output := "Buildpacks:\n"

	if len(buildpacks) == 0 {
		warnings := []string{
			fmt.Sprintf("%s has no buildpacks", builderName),
			"Users must supply buildpacks from the host machine",
		}

		return fmt.Sprintf("%s  %s\n", output, none), warnings, nil
	}

	var (
		tabWriterBuf         = bytes.Buffer{}
		spaceStrippingWriter = &trailingSpaceStrippingWriter{
			output: &tabWriterBuf,
		}
		buildpacksTabWriter = tabwriter.NewWriter(spaceStrippingWriter, writerMinWidth, writerPadChar, buildpacksTabWidth, writerPadChar, writerFlags)
	)

	_, err := fmt.Fprint(buildpacksTabWriter, "  ID\tNAME\tVERSION\tHOMEPAGE\n")
	if err != nil {
		return "", []string{}, fmt.Errorf("writing to tab writer: %w", err)
	}

	for _, b := range buildpacks {
		_, err = fmt.Fprintf(buildpacksTabWriter, "  %s\t%s\t%s\t%s\n", b.ID, strs.ValueOrDefault(b.Name, "-"), b.Version, strs.ValueOrDefault(b.Homepage, "-"))
		if err != nil {
			return "", []string{}, fmt.Errorf("writing to tab writer: %w", err)
		}
	}

	err = buildpacksTabWriter.Flush()
	if err != nil {
		return "", []string{}, fmt.Errorf("flushing tab writer: %w", err)
	}

	output += tabWriterBuf.String()
	return output, []string{}, nil
}

const lifecycleFormat = `
Lifecycle:
  Version: %s
  Buildpack APIs:
    Deprecated: %s
    Supported: %s
  Platform APIs:
    Deprecated: %s
    Supported: %s
`

func lifecycleOutput(lifecycleInfo builder.LifecycleDescriptor, builderName string) (string, []string) {
	var warnings []string

	version := none
	if lifecycleInfo.Info.Version != nil {
		version = lifecycleInfo.Info.Version.String()
	}

	if version == none {
		warnings = append(warnings, fmt.Sprintf("%s does not specify a Lifecycle version", builderName))
	}

	supportedBuildpackAPIs := stringFromAPISet(lifecycleInfo.APIs.Buildpack.Supported)
	if supportedBuildpackAPIs == none {
		warnings = append(warnings, fmt.Sprintf("%s does not specify supported Lifecycle Buildpack APIs", builderName))
	}

	supportedPlatformAPIs := stringFromAPISet(lifecycleInfo.APIs.Platform.Supported)
	if supportedPlatformAPIs == none {
		warnings = append(warnings, fmt.Sprintf("%s does not specify supported Lifecycle Platform APIs", builderName))
	}

	return fmt.Sprintf(
		lifecycleFormat,
		version,
		stringFromAPISet(lifecycleInfo.APIs.Buildpack.Deprecated),
		supportedBuildpackAPIs,
		stringFromAPISet(lifecycleInfo.APIs.Platform.Deprecated),
		supportedPlatformAPIs,
	), warnings
}

func stringFromAPISet(versions builder.APISet) string {
	if len(versions) == 0 {
		return none
	}

	return strings.Join(versions.AsStrings(), ", ")
}

const (
	branchPrefix     = " ├ "
	lastBranchPrefix = " └ "
	trunkPrefix      = " │ "
)

func detectionOrderOutput(order pubbldr.DetectionOrder, builderName string) (string, []string, error) {
	output := "Detection Order:\n"

	if len(order) == 0 {
		warnings := []string{
			fmt.Sprintf("%s has no buildpacks", builderName),
			"Users must build with explicitly specified buildpacks",
		}

		return fmt.Sprintf("%s  %s\n", output, none), warnings, nil
	}

	tabWriterBuf := bytes.Buffer{}
	spaceStrippingWriter := &trailingSpaceStrippingWriter{
		output: &tabWriterBuf,
	}

	detectionOrderTabWriter := tabwriter.NewWriter(spaceStrippingWriter, writerMinWidth, writerTabWidth, defaultTabWidth, writerPadChar, writerFlags)
	err := writeDetectionOrderGroup(detectionOrderTabWriter, order, "")
	if err != nil {
		return "", []string{}, fmt.Errorf("writing detection order group: %w", err)
	}
	err = detectionOrderTabWriter.Flush()
	if err != nil {
		return "", []string{}, fmt.Errorf("flushing tab writer: %w", err)
	}

	output += tabWriterBuf.String()
	return output, []string{}, nil
}

func detectionOrderExtOutput(order pubbldr.DetectionOrder, builderName string) (string, []string, error) {
	output := "Detection Order (Extensions):\n"

	if len(order) == 0 {
		return fmt.Sprintf("%s  %s\n", output, none), nil, nil
	}

	tabWriterBuf := bytes.Buffer{}
	spaceStrippingWriter := &trailingSpaceStrippingWriter{
		output: &tabWriterBuf,
	}

	detectionOrderExtTabWriter := tabwriter.NewWriter(spaceStrippingWriter, writerMinWidth, writerTabWidth, defaultTabWidth, writerPadChar, writerFlags)
	err := writeDetectionOrderGroup(detectionOrderExtTabWriter, order, "")
	if err != nil {
		return "", []string{}, fmt.Errorf("writing detection order group: %w", err)
	}
	err = detectionOrderExtTabWriter.Flush()
	if err != nil {
		return "", []string{}, fmt.Errorf("flushing tab writer: %w", err)
	}

	output += tabWriterBuf.String()
	return output, []string{}, nil
}

func writeDetectionOrderGroup(writer io.Writer, order pubbldr.DetectionOrder, prefix string) error {
	groupNumber := 0

	for i, orderEntry := range order {
		lastInGroup := i == len(order)-1
		includesSubGroup := len(orderEntry.GroupDetectionOrder) > 0

		orderPrefix, err := writeAndUpdateEntryPrefix(writer, lastInGroup, prefix)
		if err != nil {
			return fmt.Errorf("writing detection group prefix: %w", err)
		}

		if includesSubGroup {
			groupPrefix := orderPrefix

			if orderEntry.ID != "" {
				err = writeDetectionOrderBuildpack(writer, orderEntry)
				if err != nil {
					return fmt.Errorf("writing detection order buildpack: %w", err)
				}

				if lastInGroup {
					_, err = fmt.Fprintf(writer, "%s%s", groupPrefix, lastBranchPrefix)
					if err != nil {
						return fmt.Errorf("writing to detection order group writer: %w", err)
					}
					groupPrefix = fmt.Sprintf("%s   ", groupPrefix)
				} else {
					_, err = fmt.Fprintf(writer, "%s%s", orderPrefix, lastBranchPrefix)
					if err != nil {
						return fmt.Errorf("writing to detection order group writer: %w", err)
					}
					groupPrefix = fmt.Sprintf("%s   ", groupPrefix)
				}
			}

			groupNumber++
			_, err = fmt.Fprintf(writer, "Group #%d:\n", groupNumber)
			if err != nil {
				return fmt.Errorf("writing to detection order group writer: %w", err)
			}
			err = writeDetectionOrderGroup(writer, orderEntry.GroupDetectionOrder, groupPrefix)
			if err != nil {
				return fmt.Errorf("writing detection order group: %w", err)
			}
		} else {
			err := writeDetectionOrderBuildpack(writer, orderEntry)
			if err != nil {
				return fmt.Errorf("writing detection order buildpack: %w", err)
			}
		}
	}

	return nil
}

func writeAndUpdateEntryPrefix(writer io.Writer, last bool, prefix string) (string, error) {
	if last {
		_, err := fmt.Fprintf(writer, "%s%s", prefix, lastBranchPrefix)
		if err != nil {
			return "", fmt.Errorf("writing detection order prefix: %w", err)
		}
		return fmt.Sprintf("%s%s", prefix, "   "), nil
	}

	_, err := fmt.Fprintf(writer, "%s%s", prefix, branchPrefix)
	if err != nil {
		return "", fmt.Errorf("writing detection order prefix: %w", err)
	}
	return fmt.Sprintf("%s%s", prefix, trunkPrefix), nil
}

func writeDetectionOrderBuildpack(writer io.Writer, entry pubbldr.DetectionOrderEntry) error {
	_, err := fmt.Fprintf(
		writer,
		"%s\t%s%s\n",
		entry.FullName(),
		stringFromOptional(entry.Optional),
		stringFromCyclical(entry.Cyclical),
	)

	if err != nil {
		return fmt.Errorf("writing buildpack in detection order: %w", err)
	}

	return nil
}

func stringFromOptional(optional bool) string {
	if optional {
		return "(optional)"
	}

	return ""
}

func stringFromCyclical(cyclical bool) string {
	if cyclical {
		return "[cyclic]"
	}

	return ""
}
