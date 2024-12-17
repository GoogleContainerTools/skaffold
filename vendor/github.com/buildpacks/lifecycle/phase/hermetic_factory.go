package phase

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/log"
)

// HermeticFactory is used to construct lifecycle phases that do NOT require access to an image repository.
type HermeticFactory struct {
	platformAPI   *api.Version
	apiVerifier   BuildpackAPIVerifier
	configHandler ConfigHandler
	dirStore      DirStore
}

// NewHermeticFactory is used to construct a new HermeticFactory.
func NewHermeticFactory(
	platformAPI *api.Version,
	apiVerifier BuildpackAPIVerifier,
	configHandler ConfigHandler,
	dirStore DirStore,
) *HermeticFactory {
	return &HermeticFactory{
		platformAPI:   platformAPI,
		apiVerifier:   apiVerifier,
		configHandler: configHandler,
		dirStore:      dirStore,
	}
}

func (f *HermeticFactory) getExtensions(groupPath string, logger log.Logger) ([]buildpack.GroupElement, error) {
	group, err := f.configHandler.ReadGroup(groupPath)
	if err != nil {
		return nil, fmt.Errorf("reading group: %w", err)
	}
	if err = f.verifyGroup(group.GroupExtensions, logger); err != nil {
		return nil, err
	}
	return group.GroupExtensions, nil
}

func (f *HermeticFactory) getOrder(path string, logger log.Logger) (order buildpack.Order, hasExtensions bool, err error) {
	orderBp, orderExt, orderErr := f.configHandler.ReadOrder(path)
	if orderErr != nil {
		err = errors.Wrap(orderErr, "reading order")
		return
	}
	if len(orderExt) > 0 {
		hasExtensions = true
	}
	if err = f.verifyOrder(orderBp, orderExt, logger); err != nil {
		return
	}
	order = PrependExtensions(orderBp, orderExt)
	return
}

func (f *HermeticFactory) verifyGroup(group []buildpack.GroupElement, logger log.Logger) error {
	for _, groupEl := range group {
		if err := f.apiVerifier.VerifyBuildpackAPI(groupEl.Kind(), groupEl.String(), groupEl.API, logger); err != nil {
			return err
		}
	}
	return nil
}

func (f *HermeticFactory) verifyOrder(orderBp buildpack.Order, orderExt buildpack.Order, logger log.Logger) error {
	for _, group := range append(orderBp, orderExt...) {
		for _, groupEl := range group.Group {
			module, err := f.dirStore.Lookup(groupEl.Kind(), groupEl.ID, groupEl.Version)
			if err != nil {
				return err
			}
			if err = f.apiVerifier.VerifyBuildpackAPI(groupEl.Kind(), groupEl.String(), module.API(), logger); err != nil {
				return err
			}
		}
	}
	return nil
}
