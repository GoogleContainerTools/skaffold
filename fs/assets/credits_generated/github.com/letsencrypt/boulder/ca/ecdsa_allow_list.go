package ca

import (
	"sync"

	"github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/reloader"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/yaml.v3"
)

// ECDSAAllowList acts as a container for a map of Registration IDs, a
// mutex, and a file reloader. This allows the map of IDs to be updated
// safely if changes to the allow list are detected.
type ECDSAAllowList struct {
	sync.RWMutex
	regIDsMap map[int64]bool
	reloader  *reloader.Reloader
	logger    log.Logger
}

// Update is an exported method (typically specified as a callback to a
// file reloader) that replaces the inner `regIDsMap` with the contents
// of a YAML list (as bytes).
func (e *ECDSAAllowList) Update(contents []byte) error {
	var regIDs []int64
	err := yaml.Unmarshal(contents, &regIDs)
	if err != nil {
		return err
	}
	e.Lock()
	defer e.Unlock()
	e.regIDsMap = makeRegIDsMap(regIDs)
	return nil
}

// permitted checks if ECDSA issuance is permitted for the specified
// Registration ID.
func (e *ECDSAAllowList) permitted(regID int64) bool {
	e.RLock()
	defer e.RUnlock()
	return e.regIDsMap[regID]
}

// length returns the number of entries currently in the allow list.
func (e *ECDSAAllowList) length() int {
	e.RLock()
	defer e.RUnlock()
	return len(e.regIDsMap)
}

// Stop stops an active allow list reloader. Typically called during
// boulder-ca shutdown.
func (e *ECDSAAllowList) Stop() {
	e.Lock()
	defer e.Unlock()
	if e.reloader != nil {
		e.reloader.Stop()
	}
}

func makeRegIDsMap(regIDs []int64) map[int64]bool {
	regIDsMap := make(map[int64]bool)
	for _, regID := range regIDs {
		regIDsMap[regID] = true
	}
	return regIDsMap
}

// NewECDSAAllowListFromFile is exported to allow `boulder-ca` to
// construct a new `ECDSAAllowList` object. An initial entry count is
// returned to `boulder-ca` for logging purposes.
func NewECDSAAllowListFromFile(filename string, logger log.Logger, metric *prometheus.GaugeVec) (*ECDSAAllowList, int, error) {
	allowList := &ECDSAAllowList{logger: logger}
	// Create an allow list reloader. This also populates the inner
	// allowList regIDsMap.
	reloader, err := reloader.New(filename, allowList.Update, logger)
	if err != nil {
		return nil, 0, err
	}
	allowList.reloader = reloader
	return allowList, allowList.length(), nil
}
