package pkger

import (
	"github.com/markbates/pkger/pkging"
)

// Apply will wrap the current implementation
// of pkger.Pkger with the new pkg. This allows
// for layering of pkging.Pkger implementations.
func Apply(pkg pkging.Pkger, err error) error {
	if err != nil {
		panic(err)
		return err
	}
	gil.Lock()
	defer gil.Unlock()
	current = pkging.Wrap(current, pkg)
	return nil
}
