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
	"io"
	"sort"
	"sync"
	"time"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/docker/docker/api/types"
	"github.com/dustin/go-humanize"
	"github.com/sirupsen/logrus"
)

const (
	usageRetries       = 5
	usageRetryInterval = 500 * time.Millisecond
)

type pruner struct {
	localDocker   docker.LocalDaemon
	pruneChildren bool
	pruneMutex    sync.Mutex
}

func newPruner(dockerApi docker.LocalDaemon, pruneChildren bool) *pruner {
	return &pruner{
		localDocker:   dockerApi,
		pruneChildren: pruneChildren,
	}
}

func (p *pruner) listUniqImages(ctx context.Context, name string) ([]types.ImageSummary, error) {
	imgs, err := p.localDocker.ImageList(ctx, name)
	if err != nil {
		return nil, err
	}
	if len(imgs) < 2 {
		return imgs, nil
	}

	sort.Slice(imgs, func(i, j int) bool {
		// reverse sort
		return imgs[i].Created > imgs[j].Created
	})

	// keep only uniq images (an image can have more than one tag)
	uqIdx := 0
	for i, img := range imgs {
		if imgs[i].ID != imgs[uqIdx].ID {
			uqIdx++
			imgs[uqIdx] = img
		}
	}
	return imgs[:uqIdx+1], nil
}

func (p *pruner) startCleanupOldImages(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) {
	toPrune := p.collectImagesToPrune(ctx, artifacts)
	if len(toPrune) > 0 {
		go p.runPrune(ctx, out, toPrune)
	}
}

func (p *pruner) cleanupOldImages(ctx context.Context, out io.Writer, artifacts []*latest.Artifact) {
	toPrune := p.collectImagesToPrune(ctx, artifacts)
	if len(toPrune) > 0 {
		p.runPrune(ctx, out, toPrune)
	}
}

func (p *pruner) runPrune(ctx context.Context, out io.Writer, ids []string) {
	logrus.Debugf("Going to prune: %v", ids)
	// docker API does not support concurrent prune/utilization info request
	// so let's serialize the access to it
	t0 := time.Now()
	p.pruneMutex.Lock()
	logrus.Tracef("Prune mutex wait time: %v", time.Since(t0))
	defer p.pruneMutex.Unlock()

	beforeDu, err := p.diskUsage(ctx)
	if err != nil {
		logrus.Warnf("Failed to get docker usage info: %v", err)
	}

	err = p.localDocker.Prune(ctx, out, ids, p.pruneChildren)
	if err != nil {
		logrus.Warnf("Failed to prune: %v", err)
		return
	}
	// do not print usage report, if initial 'du' failed
	if beforeDu > 0 {
		afterDu, err := p.diskUsage(ctx)
		if err != nil {
			logrus.Warnf("Failed to get docker usage info: %v", err)
			return
		}
		if beforeDu > afterDu {
			logrus.Infof("%d image(s) pruned. Reclaimed disk space: %s",
				len(ids), humanize.Bytes(beforeDu-afterDu))
		} else {
			logrus.Infof("%d image(s) pruned", len(ids))
		}
	}
}

func (p *pruner) collectImagesToPrune(ctx context.Context, artifacts []*latest.Artifact) []string {
	// in case we're trying to build multiple images with the same ref in the same pipeline
	imgNameCount := make(map[string]int)
	for _, a := range artifacts {
		imgNameCount[a.ImageName]++
	}
	var rt []string
	for _, a := range artifacts {
		imgs, err := p.listUniqImages(ctx, a.ImageName)
		if err != nil {
			logrus.Warnf("failed to list images: %v", err)
			continue
		}
		for i := imgNameCount[a.ImageName]; i < len(imgs); i++ {
			rt = append(rt, imgs[i].ID)
		}
	}
	return rt
}

func (p *pruner) diskUsage(ctx context.Context) (uint64, error) {
	for retry := 0; retry < usageRetries-1; retry++ {
		if ctx.Err() != nil {
			return 0, ctx.Err()
		}
		usage, err := p.localDocker.DiskUsage(ctx)
		if err == nil {
			return usage, nil
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
