/*
Copyright 2020 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package local

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
)

const (
	usageRetries       = 5
	usageRetryInterval = 500 * time.Millisecond
)

type pruner struct {
	localDocker   docker.LocalDaemon
	pruneChildren bool
	pruneMutex    sync.Mutex
	prunedImgIDs  map[string]struct{}
}

func newPruner(dockerAPI docker.LocalDaemon, pruneChildren bool) *pruner {
	return &pruner{
		localDocker:   dockerAPI,
		pruneChildren: pruneChildren,
		prunedImgIDs:  make(map[string]struct{}),
	}
}

func (p *pruner) listImages(ctx context.Context, name string) ([]types.ImageSummary, error) {
	imgs, err := p.localDocker.ImageList(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(imgs) < 2 {
		// no need to sort
		return imgs, nil
	}

	sort.Slice(imgs, func(i, j int) bool {
		// reverse sort
		return imgs[i].Created > imgs[j].Created
	})

	return imgs, nil
}

func (p *pruner) cleanup(ctx context.Context, sync bool, artifacts []string) {
	toPrune := p.collectImagesToPrune(ctx, artifacts)
	if len(toPrune) == 0 {
		return
	}

	if sync {
		err := p.runPrune(ctx, toPrune)
		if err != nil {
			logrus.Debugf("Failed to prune: %v", err)
		}
	} else {
		go func() {
			err := p.runPrune(ctx, toPrune)
			if err != nil {
				logrus.Debugf("Failed to prune: %v", err)
			}
		}()
	}
}

func (p *pruner) asynchronousCleanupOldImages(ctx context.Context, artifacts []string) {
	p.cleanup(ctx, false /*async*/, artifacts)
}

func (p *pruner) synchronousCleanupOldImages(ctx context.Context, artifacts []string) {
	p.cleanup(ctx, true /*sync*/, artifacts)
}

func (p *pruner) isPruned(id string) bool {
	p.pruneMutex.Lock()
	defer p.pruneMutex.Unlock()
	_, pruned := p.prunedImgIDs[id]
	return pruned
}

func (p *pruner) runPrune(ctx context.Context, ids []string) error {
	logrus.Debugf("Going to prune: %v", ids)
	// docker API does not support concurrent prune/utilization info request
	// so let's serialize the access to it
	t0 := time.Now()
	p.pruneMutex.Lock()
	logrus.Tracef("Prune mutex wait time: %v", time.Since(t0))
	defer p.pruneMutex.Unlock()

	beforeDu, err := p.diskUsage(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		logrus.Debugf("Failed to get docker usage info: %v", err)
	}

	pruned, err := p.localDocker.Prune(ctx, ids, p.pruneChildren)
	for _, pi := range pruned {
		p.prunedImgIDs[pi] = struct{}{}
	}
	if err != nil {
		return err
	}
	// do not print usage report, if initial 'du' failed
	if beforeDu > 0 {
		afterDu, err := p.diskUsage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			logrus.Debugf("Failed to get docker usage info: %v", err)
			return nil
		}
		if beforeDu >= afterDu {
			logrus.Infof("%d image(s) pruned. Reclaimed disk space: %s",
				len(ids), humanize.Bytes(beforeDu-afterDu))
		} else {
			logrus.Infof("%d image(s) pruned", len(ids))
		}
	}
	return nil
}

func (p *pruner) collectImagesToPrune(ctx context.Context, artifacts []string) []string {
	// in case we're trying to build multiple images with the same ref in the same pipeline
	imgNameCount := make(map[string]int)
	for _, a := range artifacts {
		imgNameCount[a]++
	}
	imgProcessed := make(map[string]struct{})
	var rt []string
	for _, a := range artifacts {
		if _, ok := imgProcessed[a]; ok {
			continue
		}
		imgProcessed[a] = struct{}{}

		imgs, err := p.listImages(ctx, a)
		if err != nil {
			switch err {
			case context.Canceled, context.DeadlineExceeded:
				return []string{}
			}
			logrus.Warnf("failed to list images: %v", err)
			continue
		}
		for i := imgNameCount[a]; i < len(imgs); i++ {
			rt = append(rt, imgs[i].ID)
		}
	}
	return rt
}

func (p *pruner) diskUsage(ctx context.Context) (uint64, error) {
	for retry := 0; retry < usageRetries-1; retry++ {
		usage, err := p.localDocker.DiskUsage(ctx)
		if err == nil {
			return usage, nil
		}
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		// DiskUsage(..) may return "operation in progress" error.
		logrus.Debugf("[%d of %d] failed to get disk usage: %v. Will retry in %v",
			retry, usageRetries, err, usageRetryInterval)
		time.Sleep(usageRetryInterval)
	}

	usage, err := p.localDocker.DiskUsage(ctx)
	if err == nil {
		return usage, nil
	}
	logrus.Debugf("Failed to get usage after: %v. giving up", err)
	return 0, err
}
