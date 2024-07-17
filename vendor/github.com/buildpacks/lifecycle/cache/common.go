package cache

import (
	"errors"
)

var errCacheCommitted = errors.New("cache cannot be modified after commit")
