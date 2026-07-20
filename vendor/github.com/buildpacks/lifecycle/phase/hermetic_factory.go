package phase

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/buildpacks/lifecycle/api"
	"github.com/buildpacks/lifecycle/buildpack"
	"github.com/buildpacks/lifecycle/log"
	"github.com/buildpacks/lifecycle/platform/files"
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

func (f *HermeticFactory) getOrderWithSystem(orderPath string, systemPath string, logger log.Logger) (order buildpack.Order, hasExtensions bool, err error) {
	// Read the base order
	orderBp, orderExt, orderErr := f.configHandler.ReadOrder(orderPath)
	if orderErr != nil {
		err = errors.Wrap(orderErr, "reading order")
		return
	}
	if len(orderExt) > 0 {
		hasExtensions = true
	}

	// Read and merge system buildpacks only if Platform API >= 0.15
	if f.platformAPI.AtLeast("0.15") {
		system, sysErr := f.configHandler.ReadSystem(systemPath, logger)
		if sysErr != nil {
			err = errors.Wrap(sysErr, "reading system")
			return
		}

		// Merge system buildpacks with order
		orderBp = mergeSystemBuildpacks(orderBp, system, logger)
	}

	if err = f.verifyOrder(orderBp, orderExt, logger); err != nil {
		return
	}
	order = PrependExtensions(orderBp, orderExt)
	return
}

// mergeSystemBuildpacks merges system.pre and system.post buildpacks with the order
func mergeSystemBuildpacks(order buildpack.Order, system files.System, logger log.Logger) buildpack.Order {
	if len(system.Pre.Buildpacks) == 0 && len(system.Post.Buildpacks) == 0 {
		return order
	}

	var merged buildpack.Order

	// For each group in the order, prepend and append system buildpacks (skipping duplicates)
	for _, group := range order {
		// Filter out pre-buildpacks that are already in the group
		preBuildpacks := filterDuplicates(convertSystemToGroupElements(system.Pre.Buildpacks), group.Group)

		// Filter out post-buildpacks that are already in the group
		postBuildpacks := filterDuplicates(convertSystemToGroupElements(system.Post.Buildpacks), group.Group)

		if len(preBuildpacks) > 0 {
			logger.Debugf("Prepending %d system buildpack(s) to group", len(preBuildpacks))
		}
		if len(postBuildpacks) > 0 {
			logger.Debugf("Appending %d system buildpack(s) to group", len(postBuildpacks))
		}

		newGroup := buildpack.Group{
			Group:           append(append(preBuildpacks, group.Group...), postBuildpacks...),
			GroupExtensions: group.GroupExtensions,
		}
		merged = append(merged, newGroup)
	}

	return merged
}

// filterDuplicates filters out buildpacks from systemBps that already exist in existingGroup (by ID only)
func filterDuplicates(systemBps []buildpack.GroupElement, existingGroup []buildpack.GroupElement) []buildpack.GroupElement {
	var filtered []buildpack.GroupElement
	for _, sysBp := range systemBps {
		duplicate := false
		for _, existing := range existingGroup {
			if sysBp.ID == existing.ID {
				duplicate = true
				break
			}
		}
		if !duplicate {
			filtered = append(filtered, sysBp)
		}
	}
	return filtered
}

// convertSystemToGroupElements converts system buildpack references to group elements
func convertSystemToGroupElements(systemBps []files.SystemBuildpack) []buildpack.GroupElement {
	var elements []buildpack.GroupElement
	for _, sysBp := range systemBps {
		elements = append(elements, buildpack.GroupElement{
			ID:      sysBp.ID,
			Version: sysBp.Version,
		})
	}
	return elements
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
