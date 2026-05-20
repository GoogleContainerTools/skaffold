package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/internal/pathutil"
	giturl "github.com/go-git/go-git/v5/internal/url"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/format/index"
	"github.com/go-git/go-git/v5/plumbing/transport"
)

var (
	ErrSubmoduleAlreadyInitialized = errors.New("submodule already initialized")
	ErrSubmoduleNotInitialized     = errors.New("submodule not initialized")
)

// Submodule a submodule allows you to keep another Git repository in a
// subdirectory of your repository.
type Submodule struct {
	// initialized defines if a submodule was already initialized.
	initialized bool

	c *config.Submodule
	w *Worktree
}

// Config returns the submodule config
func (s *Submodule) Config() *config.Submodule {
	return s.c
}

// Init initialize the submodule reading the recorded Entry in the index for
// the given submodule
func (s *Submodule) Init() error {
	cfg, err := s.w.r.Config()
	if err != nil {
		return err
	}

	_, ok := cfg.Submodules[s.c.Name]
	if ok {
		return ErrSubmoduleAlreadyInitialized
	}

	s.initialized = true

	cfg.Submodules[s.c.Name] = s.c
	return s.w.r.Storer.SetConfig(cfg)
}

// Status returns the status of the submodule.
func (s *Submodule) Status() (*SubmoduleStatus, error) {
	idx, err := s.w.r.Storer.Index()
	if err != nil {
		return nil, err
	}

	return s.status(idx)
}

func (s *Submodule) status(idx *index.Index) (*SubmoduleStatus, error) {
	status := &SubmoduleStatus{
		Path: s.c.Path,
	}

	e, err := idx.Entry(s.c.Path)
	if err != nil && err != index.ErrEntryNotFound {
		return nil, err
	}

	if e != nil {
		status.Expected = e.Hash
	}

	if !s.initialized {
		return status, nil
	}

	r, err := s.Repository()
	if err != nil {
		return nil, err
	}

	head, err := r.Head()
	if err == nil {
		status.Current = head.Hash()
	}

	if err != nil && err == plumbing.ErrReferenceNotFound {
		err = nil
	}

	return status, err
}

// Repository returns the Repository represented by this submodule
func (s *Submodule) Repository() (*Repository, error) {
	if !s.initialized {
		return nil, ErrSubmoduleNotInitialized
	}

	storer, err := s.w.r.Storer.Module(s.c.Name)
	if err != nil {
		return nil, err
	}

	_, err = storer.Reference(plumbing.HEAD)
	if err != nil && err != plumbing.ErrReferenceNotFound {
		return nil, err
	}

	var exists bool
	if err == nil {
		exists = true
	}

	// s.c.Path is sourced from the worktree's .gitmodules and is
	// therefore tree-controlled. Apply the strict tree-path validator
	// before chroot — the wrapper's tolerant validPath would let a
	// final-position .git component through (e.g. "submodule/.git"),
	// which a malicious .gitmodules could use to chroot the submodule
	// worktree into the repository's actual .git directory.
	if err := pathutil.ValidTreePath(s.c.Path); err != nil {
		return nil, err
	}

	var worktree billy.Filesystem
	if worktree, err = s.w.Filesystem.Chroot(s.c.Path); err != nil {
		return nil, err
	}

	if exists {
		return Open(storer, worktree)
	}

	r, err := Init(storer, worktree)
	if err != nil {
		return nil, err
	}

	moduleEndpoint, err := transport.NewEndpoint(s.c.URL)
	if err != nil {
		return nil, err
	}

	// A relative submodule URL such as "../X.git" must resolve against
	// the parent repository's remote URL, not against the process CWD.
	// Detect relativity from the raw configured URL because
	// transport.NewEndpoint normalizes local paths to absolute form via
	// filepath.Abs, which would otherwise mask the relative form here.
	if giturl.IsLocalEndpoint(s.c.URL) &&
		!path.IsAbs(s.c.URL) && !filepath.IsAbs(s.c.URL) {

		base, err := defaultRemote(s.w.r)
		if err != nil {
			return nil, fmt.Errorf("resolving relative submodule URL: %w", err)
		}

		rootEndpoint, err := transport.NewEndpoint(base.URLs[0])
		if err != nil {
			return nil, err
		}

		rootEndpoint.Path = path.Join(rootEndpoint.Path, s.c.URL)
		*moduleEndpoint = *rootEndpoint
	}

	_, err = r.CreateRemote(&config.RemoteConfig{
		Name: DefaultRemoteName,
		URLs: []string{moduleEndpoint.String()},
	})

	return r, err
}

// defaultRemote returns the remote that relative submodule URLs are
// resolved against, mirroring canonical Git's repo_default_remote
// (remote.c) and resolve_relative_url (builtin/submodule--helper.c):
//
//  1. if HEAD is on a branch with branch.<name>.remote configured,
//     use that remote;
//  2. else if exactly one remote is configured, use it;
//  3. otherwise fall back to DefaultRemoteName ("origin").
//
// Each rule falls through unconditionally: a branch lookup that
// finds the branch but with an empty Remote does not short-circuit
// rule (2). Returns an error when the chosen remote is not configured.
func defaultRemote(r *Repository) (*config.RemoteConfig, error) {
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}

	if ref, err := r.Reference(plumbing.HEAD, false); err == nil &&
		ref.Type() == plumbing.SymbolicReference &&
		ref.Target().IsBranch() {
		if b, ok := cfg.Branches[ref.Target().Short()]; ok && b.Remote != "" {
			return lookupRemote(cfg, b.Remote)
		}
	}

	if len(cfg.Remotes) == 1 {
		for name := range cfg.Remotes {
			return lookupRemote(cfg, name)
		}
	}

	return lookupRemote(cfg, DefaultRemoteName)
}

func lookupRemote(cfg *config.Config, name string) (*config.RemoteConfig, error) {
	rc, ok := cfg.Remotes[name]
	if !ok {
		return nil, fmt.Errorf("remote %q not found", name)
	}
	if len(rc.URLs) == 0 {
		return nil, fmt.Errorf("remote %q has no configured URL", name)
	}
	return rc, nil
}

// Update the registered submodule to match what the superproject expects, the
// submodule should be initialized first calling the Init method or setting in
// the options SubmoduleUpdateOptions.Init equals true
func (s *Submodule) Update(o *SubmoduleUpdateOptions) error {
	return s.UpdateContext(context.Background(), o)
}

// UpdateContext the registered submodule to match what the superproject
// expects, the submodule should be initialized first calling the Init method or
// setting in the options SubmoduleUpdateOptions.Init equals true.
//
// The provided Context must be non-nil. If the context expires before the
// operation is complete, an error is returned. The context only affects the
// transport operations.
func (s *Submodule) UpdateContext(ctx context.Context, o *SubmoduleUpdateOptions) error {
	return s.update(ctx, o, plumbing.ZeroHash)
}

func (s *Submodule) update(ctx context.Context, o *SubmoduleUpdateOptions, forceHash plumbing.Hash) error {
	if !s.initialized && !o.Init {
		return ErrSubmoduleNotInitialized
	}

	if !s.initialized && o.Init {
		if err := s.Init(); err != nil {
			return err
		}
	}

	idx, err := s.w.r.Storer.Index()
	if err != nil {
		return err
	}

	hash := forceHash
	if hash.IsZero() {
		e, err := idx.Entry(s.c.Path)
		if err != nil {
			return err
		}

		hash = e.Hash
	}

	r, err := s.Repository()
	if err != nil {
		return err
	}

	if err := s.fetchAndCheckout(ctx, r, o, hash); err != nil {
		return err
	}

	return s.doRecursiveUpdate(ctx, r, o)
}

func (s *Submodule) doRecursiveUpdate(ctx context.Context, r *Repository, o *SubmoduleUpdateOptions) error {
	if o.RecurseSubmodules == NoRecurseSubmodules {
		return nil
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	l, err := w.Submodules()
	if err != nil {
		return err
	}

	new := &SubmoduleUpdateOptions{}
	*new = *o

	new.RecurseSubmodules--
	return l.UpdateContext(ctx, new)
}

func (s *Submodule) fetchAndCheckout(
	ctx context.Context, r *Repository, o *SubmoduleUpdateOptions, hash plumbing.Hash,
) error {
	if !o.NoFetch {
		err := r.FetchContext(ctx, &FetchOptions{Auth: o.Auth, Depth: o.Depth})
		if err != nil && err != NoErrAlreadyUpToDate {
			return err
		}
	}

	w, err := r.Worktree()
	if err != nil {
		return err
	}

	// Handle a case when submodule refers to an orphaned commit that's still reachable
	// through Git server using a special protocol capability[1].
	//
	// [1]: https://git-scm.com/docs/protocol-capabilities#_allow_reachable_sha1_in_want
	if !o.NoFetch {
		if _, err := w.r.Object(plumbing.AnyObject, hash); err != nil {
			refSpec := config.RefSpec("+" + hash.String() + ":" + hash.String())

			err := r.FetchContext(ctx, &FetchOptions{
				Auth:     o.Auth,
				RefSpecs: []config.RefSpec{refSpec},
				Depth:    o.Depth,
			})
			if err != nil && err != NoErrAlreadyUpToDate && err != ErrExactSHA1NotSupported {
				return err
			}
		}
	}

	if err := w.Checkout(&CheckoutOptions{Hash: hash}); err != nil {
		return err
	}

	head := plumbing.NewHashReference(plumbing.HEAD, hash)
	return r.Storer.SetReference(head)
}

// Submodules list of several submodules from the same repository.
type Submodules []*Submodule

// Init initializes the submodules in this list.
func (s Submodules) Init() error {
	for _, sub := range s {
		if err := sub.Init(); err != nil {
			return err
		}
	}

	return nil
}

// Update updates all the submodules in this list.
func (s Submodules) Update(o *SubmoduleUpdateOptions) error {
	return s.UpdateContext(context.Background(), o)
}

// UpdateContext updates all the submodules in this list.
//
// The provided Context must be non-nil. If the context expires before the
// operation is complete, an error is returned. The context only affects the
// transport operations.
func (s Submodules) UpdateContext(ctx context.Context, o *SubmoduleUpdateOptions) error {
	for _, sub := range s {
		if err := sub.UpdateContext(ctx, o); err != nil {
			return err
		}
	}

	return nil
}

// Status returns the status of the submodules.
func (s Submodules) Status() (SubmodulesStatus, error) {
	var list SubmodulesStatus

	var r *Repository
	for _, sub := range s {
		if r == nil {
			r = sub.w.r
		}

		idx, err := r.Storer.Index()
		if err != nil {
			return nil, err
		}

		status, err := sub.status(idx)
		if err != nil {
			return nil, err
		}

		list = append(list, status)
	}

	return list, nil
}

// SubmodulesStatus contains the status for all submodiles in the worktree
type SubmodulesStatus []*SubmoduleStatus

// String is equivalent to `git submodule status`
func (s SubmodulesStatus) String() string {
	buf := bytes.NewBuffer(nil)
	for _, sub := range s {
		fmt.Fprintln(buf, sub)
	}

	return buf.String()
}

// SubmoduleStatus contains the status for a submodule in the worktree
type SubmoduleStatus struct {
	Path     string
	Current  plumbing.Hash
	Expected plumbing.Hash
	Branch   plumbing.ReferenceName
}

// IsClean is the HEAD of the submodule is equals to the expected commit
func (s *SubmoduleStatus) IsClean() bool {
	return s.Current == s.Expected
}

// String is equivalent to `git submodule status <submodule>`
//
// This will print the SHA-1 of the currently checked out commit for a
// submodule, along with the submodule path and the output of git describe fo
// the SHA-1. Each SHA-1 will be prefixed with - if the submodule is not
// initialized, + if the currently checked out submodule commit does not match
// the SHA-1 found in the index of the containing repository.
func (s *SubmoduleStatus) String() string {
	var extra string
	var status = ' '

	if s.Current.IsZero() {
		status = '-'
	} else if !s.IsClean() {
		status = '+'
	}

	if len(s.Branch) != 0 {
		extra = string(s.Branch[5:])
	} else if !s.Current.IsZero() {
		extra = s.Current.String()[:7]
	}

	if extra != "" {
		extra = fmt.Sprintf(" (%s)", extra)
	}

	return fmt.Sprintf("%c%s %s%s", status, s.Expected, s.Path, extra)
}
