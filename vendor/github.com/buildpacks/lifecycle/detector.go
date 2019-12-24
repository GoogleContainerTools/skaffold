package lifecycle

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
)

const (
	CodeDetectPass = 0
	CodeDetectFail = 100
)

var ErrFail = errors.New("no buildpacks participating")

type Buildpack struct {
	ID       string `toml:"id" json:"id"`
	Version  string `toml:"version" json:"version"`
	Optional bool   `toml:"optional,omitempty" json:"optional,omitempty"`
}

func (bp Buildpack) dir() string {
	return escapeID(bp.ID)
}

func (bp Buildpack) String() string {
	return bp.ID + "@" + bp.Version
}

func (bp Buildpack) noOpt() Buildpack {
	bp.Optional = false
	return bp
}

type BuildPlan struct {
	Entries []BuildPlanEntry `toml:"entries"`
}

type BuildPlanEntry struct {
	Providers []Buildpack `toml:"providers"`
	Requires  []Require   `toml:"requires"`
}

func (be BuildPlanEntry) noOpt() BuildPlanEntry {
	var out []Buildpack
	for _, p := range be.Providers {
		out = append(out, p.noOpt())
	}
	be.Providers = out
	return be
}

type Require struct {
	Name     string                 `toml:"name" json:"name"`
	Version  string                 `toml:"version" json:"version"`
	Metadata map[string]interface{} `toml:"metadata" json:"metadata"`
}

type Provide struct {
	Name string `toml:"name"`
}

type DetectConfig struct {
	FullEnv       []string
	ClearEnv      []string
	AppDir        string
	PlatformDir   string
	BuildpacksDir string
	Logger        Logger
	runs          *sync.Map
}

func (bp Buildpack) lookup(buildpacksDir string) (*buildpackTOML, error) {
	bpTOML := buildpackTOML{}
	bpPath, err := filepath.Abs(filepath.Join(buildpacksDir, bp.dir(), bp.Version))
	if err != nil {
		return nil, err
	}
	tomlPath := filepath.Join(bpPath, "buildpack.toml")
	if _, err := toml.DecodeFile(tomlPath, &bpTOML); err != nil {
		return nil, err
	}
	bpTOML.Path = bpPath
	return &bpTOML, nil
}

func (c *DetectConfig) process(done []Buildpack) ([]Buildpack, []BuildPlanEntry, error) {
	var runs []detectRun
	for _, bp := range done {
		t, ok := c.runs.Load(bp.String())
		if !ok {
			return nil, nil, errors.Errorf("missing detection of '%s'", bp)
		}
		run := t.(detectRun)
		if len(run.Output) > 0 {
			c.Logger.Debugf("======== Output: %s ========", bp)
			c.Logger.Debug(string(run.Output))
		}
		if run.Err != nil {
			c.Logger.Debugf("======== Error: %s ========", bp)
			c.Logger.Debug(run.Err.Error())
		}
		runs = append(runs, run)
	}

	c.Logger.Debugf("======== Results ========")

	results := detectResults{}
	detected := true
	for i, bp := range done {
		run := runs[i]
		switch run.Code {
		case CodeDetectPass:
			c.Logger.Debugf("pass: %s", bp)
			results = append(results, detectResult{bp, run})
		case CodeDetectFail:
			if bp.Optional {
				c.Logger.Debugf("skip: %s", bp)
			} else {
				c.Logger.Debugf("fail: %s", bp)
			}
			detected = detected && bp.Optional
		case -1:
			c.Logger.Debugf("err:  %s", bp)
			detected = detected && bp.Optional
		default:
			c.Logger.Debugf("err:  %s (%d)", bp, run.Code)
			detected = detected && bp.Optional
		}
	}
	if !detected {
		return nil, nil, ErrFail
	}

	i := 0
	deps, trial, err := results.runTrials(func(trial detectTrial) (depMap, detectTrial, error) {
		i++
		return c.runTrial(i, trial)
	})
	if err != nil {
		return nil, nil, err
	}

	if len(done) != len(trial) {
		c.Logger.Infof("%d of %d buildpacks participating", len(trial), len(done))
	}

	maxLength := 0
	for _, t := range trial {
		l := len(t.ID)
		if l > maxLength {
			maxLength = l
		}
	}

	f := fmt.Sprintf("%%-%ds %%s", maxLength)

	for _, t := range trial {
		c.Logger.Infof(f, t.ID, t.Version)
	}

	var found []Buildpack
	for _, r := range trial {
		found = append(found, r.Buildpack.noOpt())
	}
	var plan []BuildPlanEntry
	for _, dep := range deps {
		plan = append(plan, dep.BuildPlanEntry.noOpt())
	}
	return found, plan, nil
}

func (c *DetectConfig) runTrial(i int, trial detectTrial) (depMap, detectTrial, error) {
	c.Logger.Debugf("Resolving plan... (try #%d)", i)

	var deps depMap
	retry := true
	for retry {
		retry = false
		deps = newDepMap(trial)

		if err := deps.eachUnmetRequire(func(name string, bp Buildpack) error {
			retry = true
			if !bp.Optional {
				c.Logger.Debugf("fail: %s requires %s", bp, name)
				return ErrFail
			}
			c.Logger.Debugf("skip: %s requires %s", bp, name)
			trial = trial.remove(bp)
			return nil
		}); err != nil {
			return nil, nil, err
		}

		if err := deps.eachUnmetProvide(func(name string, bp Buildpack) error {
			retry = true
			if !bp.Optional {
				c.Logger.Debugf("fail: %s provides unused %s", bp, name)
				return ErrFail
			}
			c.Logger.Debugf("skip: %s provides unused %s", bp, name)
			trial = trial.remove(bp)
			return nil
		}); err != nil {
			return nil, nil, err
		}
	}

	if len(trial) == 0 {
		c.Logger.Debugf("fail: no viable buildpacks in group")
		return nil, nil, ErrFail
	}
	return deps, trial, nil
}

func (bp *buildpackTOML) Detect(c *DetectConfig) detectRun {
	appDir, err := filepath.Abs(c.AppDir)
	if err != nil {
		return detectRun{Code: -1, Err: err}
	}
	platformDir, err := filepath.Abs(c.PlatformDir)
	if err != nil {
		return detectRun{Code: -1, Err: err}
	}
	planDir, err := ioutil.TempDir("", "plan.")
	if err != nil {
		return detectRun{Code: -1, Err: err}
	}
	defer os.RemoveAll(planDir)
	planPath := filepath.Join(planDir, "plan.toml")
	if err := ioutil.WriteFile(planPath, nil, 0777); err != nil {
		return detectRun{Code: -1, Err: err}
	}
	out := &bytes.Buffer{}
	cmd := exec.Command(filepath.Join(bp.Path, "bin", "detect"), platformDir, planPath)
	cmd.Dir = appDir
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Env = c.FullEnv
	if bp.Buildpack.ClearEnv {
		cmd.Env = c.ClearEnv
	}

	if err := cmd.Run(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			if status, ok := err.Sys().(syscall.WaitStatus); ok {
				return detectRun{Code: status.ExitStatus(), Output: out.Bytes()}
			}
		}
		return detectRun{Code: -1, Err: err, Output: out.Bytes()}
	}
	var t detectRun
	if _, err := toml.DecodeFile(planPath, &t); err != nil {
		return detectRun{Code: -1, Err: err}
	}
	t.Output = out.Bytes()
	return t
}

type BuildpackGroup struct {
	Group []Buildpack `toml:"group"`
}

func (bg BuildpackGroup) Detect(c *DetectConfig) (BuildpackGroup, BuildPlan, error) {
	if c.runs == nil {
		c.runs = &sync.Map{}
	}
	bps, entries, err := bg.detect(nil, &sync.WaitGroup{}, c)
	return BuildpackGroup{Group: bps}, BuildPlan{Entries: entries}, err
}

func (bg BuildpackGroup) detect(done []Buildpack, wg *sync.WaitGroup, c *DetectConfig) ([]Buildpack, []BuildPlanEntry, error) {
	for i, bp := range bg.Group {
		key := bp.String()
		if hasID(done, bp.ID) {
			continue
		}
		info, err := bp.lookup(c.BuildpacksDir)
		if err != nil {
			return nil, nil, err
		}
		if info.Order != nil {
			// TODO: double-check slice safety here
			// FIXME: cyclical references lead to infinite recursion
			return info.Order.detect(done, bg.Group[i+1:], bp.Optional, wg, c)
		}
		done = append(done, bp)
		wg.Add(1)
		go func() {
			if _, ok := c.runs.Load(key); !ok {
				c.runs.Store(key, info.Detect(c))
			}
			wg.Done()
		}()
	}

	wg.Wait()

	return c.process(done)
}

func (bg BuildpackGroup) append(group ...BuildpackGroup) BuildpackGroup {
	for _, g := range group {
		bg.Group = append(bg.Group, g.Group...)
	}
	return bg
}

type BuildpackOrder []BuildpackGroup

func (bo BuildpackOrder) Detect(c *DetectConfig) (BuildpackGroup, BuildPlan, error) {
	if c.runs == nil {
		c.runs = &sync.Map{}
	}
	bps, entries, err := bo.detect(nil, nil, false, &sync.WaitGroup{}, c)
	return BuildpackGroup{Group: bps}, BuildPlan{Entries: entries}, err
}

func (bo BuildpackOrder) detect(done, next []Buildpack, optional bool, wg *sync.WaitGroup, c *DetectConfig) ([]Buildpack, []BuildPlanEntry, error) {
	ngroup := BuildpackGroup{Group: next}
	for _, group := range bo {
		// FIXME: double-check slice safety here
		found, plan, err := group.append(ngroup).detect(done, wg, c)
		if err == ErrFail {
			wg = &sync.WaitGroup{}
			continue
		}
		return found, plan, err
	}
	if optional {
		return ngroup.detect(done, wg, c)
	}
	return nil, nil, ErrFail
}

func hasID(bps []Buildpack, id string) bool {
	for _, bp := range bps {
		if bp.ID == id {
			return true
		}
	}
	return false
}

type detectRun struct {
	planSections
	Or     []planSections `toml:"or"`
	Output []byte         `toml:"-"`
	Code   int            `toml:"-"`
	Err    error          `toml:"-"`
}

type planSections struct {
	Requires []Require `toml:"requires"`
	Provides []Provide `toml:"provides"`
}

type detectResult struct {
	Buildpack
	detectRun
}

func (r *detectResult) options() []detectOption {
	var out []detectOption
	for i, sections := range append([]planSections{r.planSections}, r.Or...) {
		bp := r.Buildpack
		bp.Optional = bp.Optional && i == len(r.Or)
		out = append(out, detectOption{bp, sections})
	}
	return out
}

type detectResults []detectResult
type trialFunc func(detectTrial) (depMap, detectTrial, error)

func (rs detectResults) runTrials(f trialFunc) (depMap, detectTrial, error) {
	return rs.runTrialsFrom(nil, f)
}

func (rs detectResults) runTrialsFrom(prefix detectTrial, f trialFunc) (depMap, detectTrial, error) {
	if len(rs) == 0 {
		deps, trial, err := f(prefix)
		return deps, trial, err
	}

	var lastErr error
	for _, option := range rs[0].options() {
		deps, trial, err := rs[1:].runTrialsFrom(append(prefix, option), f)
		if err == nil {
			return deps, trial, nil
		}
		lastErr = err
	}
	return nil, nil, lastErr
}

type detectOption struct {
	Buildpack
	planSections
}

type detectTrial []detectOption

func (ts detectTrial) remove(bp Buildpack) detectTrial {
	var out detectTrial
	for _, t := range ts {
		if t.Buildpack != bp {
			out = append(out, t)
		}
	}
	return out
}

type depEntry struct {
	BuildPlanEntry
	earlyRequires []Buildpack
	extraProvides []Buildpack
}

type depMap map[string]depEntry

func newDepMap(trial detectTrial) depMap {
	m := depMap{}
	for _, option := range trial {
		for _, p := range option.Provides {
			m.provide(option.Buildpack, p)
		}
		for _, r := range option.Requires {
			m.require(option.Buildpack, r)
		}
	}
	return m
}

func (m depMap) provide(bp Buildpack, provide Provide) {
	entry := m[provide.Name]
	entry.extraProvides = append(entry.extraProvides, bp)
	m[provide.Name] = entry
}

func (m depMap) require(bp Buildpack, require Require) {
	entry := m[require.Name]
	entry.Providers = append(entry.Providers, entry.extraProvides...)
	entry.extraProvides = nil

	if len(entry.Providers) == 0 {
		entry.earlyRequires = append(entry.earlyRequires, bp)
	} else {
		entry.Requires = append(entry.Requires, require)
	}
	m[require.Name] = entry
}

func (m depMap) eachUnmetProvide(f func(name string, bp Buildpack) error) error {
	for name, entry := range m {
		if len(entry.extraProvides) != 0 {
			for _, bp := range entry.extraProvides {
				if err := f(name, bp); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (m depMap) eachUnmetRequire(f func(name string, bp Buildpack) error) error {
	for name, entry := range m {
		if len(entry.earlyRequires) != 0 {
			for _, bp := range entry.earlyRequires {
				if err := f(name, bp); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
