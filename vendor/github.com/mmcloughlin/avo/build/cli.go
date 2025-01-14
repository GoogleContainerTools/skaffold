package build

import (
	"flag"
	"io"
	"log"
	"os"
	"runtime/pprof"

	"github.com/mmcloughlin/avo/pass"
	"github.com/mmcloughlin/avo/printer"
)

// Config contains options for an avo main function.
type Config struct {
	ErrOut     io.Writer
	MaxErrors  int // max errors to report; 0 means unlimited
	CPUProfile io.WriteCloser
	Passes     []pass.Interface
}

// Main is the standard main function for an avo program. This extracts the
// result from the build Context (logging and exiting on error), and performs
// configured passes.
func Main(cfg *Config, context *Context) int {
	diag := log.New(cfg.ErrOut, "", 0)

	if cfg.CPUProfile != nil {
		defer cfg.CPUProfile.Close()
		if err := pprof.StartCPUProfile(cfg.CPUProfile); err != nil {
			diag.Println("could not start CPU profile: ", err)
			return 1
		}
		defer pprof.StopCPUProfile()
	}

	f, err := context.Result()
	if err != nil {
		LogError(diag, err, cfg.MaxErrors)
		return 1
	}

	p := pass.Concat(cfg.Passes...)
	if err := p.Execute(f); err != nil {
		diag.Println(err)
		return 1
	}

	return 0
}

// Flags represents CLI flags for an avo program.
type Flags struct {
	errout    *outputValue
	allerrors bool
	cpuprof   *outputValue
	pkg       string
	printers  []*printerValue
}

// NewFlags initializes avo flags for the given FlagSet.
func NewFlags(fs *flag.FlagSet) *Flags {
	f := &Flags{}

	f.errout = newOutputValue(os.Stderr)
	fs.Var(f.errout, "log", "diagnostics output")

	fs.BoolVar(&f.allerrors, "e", false, "no limit on number of errors reported")

	f.cpuprof = newOutputValue(nil)
	fs.Var(f.cpuprof, "cpuprofile", "write cpu profile to `file`")

	fs.StringVar(&f.pkg, "pkg", "", "package name (defaults to current directory name)")

	goasm := newPrinterValue(printer.NewGoAsm, os.Stdout)
	fs.Var(goasm, "out", "assembly output")
	f.printers = append(f.printers, goasm)

	stubs := newPrinterValue(printer.NewStubs, nil)
	fs.Var(stubs, "stubs", "go stub file")
	f.printers = append(f.printers, stubs)

	return f
}

// Config builds a configuration object based on flag values.
func (f *Flags) Config() *Config {
	pc := printer.NewGoRunConfig()
	if f.pkg != "" {
		pc.Pkg = f.pkg
	}
	passes := []pass.Interface{pass.Compile}
	for _, pv := range f.printers {
		p := pv.Build(pc)
		if p != nil {
			passes = append(passes, p)
		}
	}

	cfg := &Config{
		ErrOut:     f.errout.w,
		MaxErrors:  10,
		CPUProfile: f.cpuprof.w,
		Passes:     passes,
	}

	if f.allerrors {
		cfg.MaxErrors = 0
	}

	return cfg
}

type outputValue struct {
	w        io.WriteCloser
	filename string
}

func newOutputValue(dflt io.WriteCloser) *outputValue {
	return &outputValue{w: dflt}
}

func (o *outputValue) String() string {
	if o == nil {
		return ""
	}
	return o.filename
}

func (o *outputValue) Set(s string) error {
	o.filename = s
	if s == "-" {
		o.w = nopwritecloser{os.Stdout}
		return nil
	}
	f, err := os.Create(s)
	if err != nil {
		return err
	}
	o.w = f
	return nil
}

type printerValue struct {
	*outputValue
	Builder printer.Builder
}

func newPrinterValue(b printer.Builder, dflt io.WriteCloser) *printerValue {
	return &printerValue{
		outputValue: newOutputValue(dflt),
		Builder:     b,
	}
}

func (p *printerValue) Build(cfg printer.Config) pass.Interface {
	if p.outputValue.w == nil {
		return nil
	}
	return &pass.Output{
		Writer:  p.outputValue.w,
		Printer: p.Builder(cfg),
	}
}

// nopwritecloser wraps a Writer and provides a null implementation of Close().
type nopwritecloser struct {
	io.Writer
}

func (nopwritecloser) Close() error { return nil }
