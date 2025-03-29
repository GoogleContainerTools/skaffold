package writer

import (
	"bytes"
	"fmt"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/buildpacks/pack/internal/inspectimage"
	"github.com/buildpacks/pack/pkg/client"

	strs "github.com/buildpacks/pack/internal/strings"
	"github.com/buildpacks/pack/internal/style"
	"github.com/buildpacks/pack/pkg/logging"
)

type HumanReadable struct{}

func NewHumanReadable() *HumanReadable {
	return &HumanReadable{}
}

func (h *HumanReadable) Print(
	logger logging.Logger,
	generalInfo inspectimage.GeneralInfo,
	local, remote *client.ImageInfo,
	localErr, remoteErr error,
) error {
	if local == nil && remote == nil {
		return fmt.Errorf("unable to find image '%s' locally or remotely", generalInfo.Name)
	}

	logger.Infof("Inspecting image: %s\n", style.Symbol(generalInfo.Name))

	if err := writeRemoteImageInfo(logger, generalInfo, remote, remoteErr); err != nil {
		return err
	}

	if err := writeLocalImageInfo(logger, generalInfo, local, localErr); err != nil {
		return err
	}

	return nil
}

func writeLocalImageInfo(
	logger logging.Logger,
	generalInfo inspectimage.GeneralInfo,
	local *client.ImageInfo,
	localErr error) error {
	logger.Info("\nLOCAL:\n")

	if localErr != nil {
		logger.Errorf("%s\n", localErr)
		return nil
	}

	localDisplay := inspectimage.NewInfoDisplay(local, generalInfo)
	if localDisplay == nil {
		logger.Info("(not present)\n")
		return nil
	}

	err := writeImageInfo(logger, localDisplay)
	if err != nil {
		return fmt.Errorf("writing local builder info: %w", err)
	}

	return nil
}

func writeRemoteImageInfo(
	logger logging.Logger,
	generalInfo inspectimage.GeneralInfo,
	remote *client.ImageInfo,
	remoteErr error) error {
	logger.Info("\nREMOTE:\n")

	if remoteErr != nil {
		logger.Errorf("%s\n", remoteErr)
		return nil
	}

	remoteDisplay := inspectimage.NewInfoDisplay(remote, generalInfo)
	if remoteDisplay == nil {
		logger.Info("(not present)\n")
		return nil
	}

	err := writeImageInfo(logger, remoteDisplay)
	if err != nil {
		return fmt.Errorf("writing remote builder info: %w", err)
	}

	return nil
}

func writeImageInfo(
	logger logging.Logger,
	info *inspectimage.InfoDisplay,
) error {
	imgTpl := getImageTemplate(info)
	remoteOutput, err := getInspectImageOutput(imgTpl, info)
	if err != nil {
		logger.Error(err.Error())
		return err
	} else {
		logger.Info(remoteOutput.String())
		return nil
	}
}

func getImageTemplate(info *inspectimage.InfoDisplay) *template.Template {
	imgTpl := template.Must(template.New("runImages").
		Funcs(template.FuncMap{"StringsJoin": strings.Join}).
		Funcs(template.FuncMap{"StringsValueOrDefault": strs.ValueOrDefault}).
		Parse(runImagesTemplate))
	imgTpl = template.Must(imgTpl.New("buildpacks").Parse(buildpacksTemplate))

	imgTpl = template.Must(imgTpl.New("processes").Parse(processesTemplate))

	imgTpl = template.Must(imgTpl.New("rebasable").Parse(rebasableTemplate))

	if info != nil && info.Extensions != nil {
		imgTpl = template.Must(imgTpl.New("extensions").Parse(extensionsTemplate))
		imgTpl = template.Must(imgTpl.New("image").Parse(imageWithExtensionTemplate))
	} else {
		imgTpl = template.Must(imgTpl.New("image").Parse(imageTemplate))
	}
	return imgTpl
}

func getInspectImageOutput(
	tpl *template.Template,
	info *inspectimage.InfoDisplay) (*bytes.Buffer, error) {
	if info == nil {
		return bytes.NewBuffer([]byte("(not present)")), nil
	}
	buf := bytes.NewBuffer(nil)
	tw := tabwriter.NewWriter(buf, 0, 0, 8, ' ', 0)
	defer func() {
		tw.Flush()
	}()
	if err := tpl.Execute(tw, &struct {
		Info *inspectimage.InfoDisplay
	}{
		info,
	}); err != nil {
		return bytes.NewBuffer(nil), err
	}
	return buf, nil
}

var runImagesTemplate = `
Run Images:
{{- range $_, $m := .Info.RunImageMirrors }}
  {{- if $m.UserConfigured }}
  {{$m.Name}}	(user-configured)
  {{- else }}
  {{$m.Name}}
  {{- end }}  
{{- end }}
{{- if not .Info.RunImageMirrors }}
  (none)
{{- end }}`

var buildpacksTemplate = `
Buildpacks:
{{- if .Info.Buildpacks }}
  ID	VERSION	HOMEPAGE
{{- range $_, $b := .Info.Buildpacks }}
  {{ $b.ID }}	{{ $b.Version }}	{{ StringsValueOrDefault $b.Homepage "-" }}
{{- end }}
{{- else }}
  (buildpack metadata not present)
{{- end }}`

var extensionsTemplate = `
Extensions:
{{- if .Info.Extensions }}
  ID	VERSION	HOMEPAGE
{{- range $_, $b := .Info.Extensions }}
  {{ $b.ID }}	{{ $b.Version }}	{{ StringsValueOrDefault $b.Homepage "-" }}
{{- end }}
{{- else }}
  (extension metadata not present)
{{- end }}`

var processesTemplate = `
{{- if .Info.Processes }}

Processes:
  TYPE	SHELL	COMMAND	ARGS	WORK DIR
  {{- range $_, $p := .Info.Processes }}
    {{- if $p.Default }}
  {{ (printf "%s %s" $p.Type "(default)") }}	{{ $p.Shell }}	{{ $p.Command }}	{{ StringsJoin $p.Args " "  }}	{{ $p.WorkDir }}
    {{- else }}
  {{ $p.Type }}	{{ $p.Shell }}	{{ $p.Command }}	{{ StringsJoin $p.Args " " }}	{{ $p.WorkDir }}
    {{- end }}
  {{- end }}
{{- end }}`

var rebasableTemplate = `

Rebasable: 
{{- if or .Info.Rebasable (eq .Info.Rebasable true)  }} true 
{{- else }} false 
{{- end }}`

var imageTemplate = `
Stack: {{ .Info.StackID }}

Base Image:
{{- if .Info.Base.Reference}}
  Reference: {{ .Info.Base.Reference }}
{{- end}}
  Top Layer: {{ .Info.Base.TopLayer }}
{{ template "runImages" . }}
{{- template "rebasable" . }}
{{ template "buildpacks" . }}{{ template "processes" . }}`

var imageWithExtensionTemplate = `
Stack: {{ .Info.StackID }}

Base Image:
{{- if .Info.Base.Reference}}
  Reference: {{ .Info.Base.Reference }}
{{- end}}
  Top Layer: {{ .Info.Base.TopLayer }}
{{ template "runImages" . }}
{{- template "rebasable" . }}
{{ template "buildpacks" . }}
{{ template "extensions" . -}}
{{ template "processes" . }}`
