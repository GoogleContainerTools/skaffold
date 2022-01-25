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
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/containers/common/libimage"
	"github.com/dustin/go-humanize"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output/log"
)

const (
	usageRetries       = 5
	usageRetryInterval = 500 * time.Millisecond
)

type pruner struct {
	localDocker     docker.LocalDaemon
	libimageRuntime *libimage.Runtime
	useLibImage     bool
	pruneChildren   bool
	pruneMutex      sync.Mutex
	prunedImgIDs    map[string]struct{}
}

func newPruner(useLibimage bool, libimageRuntime *libimage.Runtime, dockerAPI docker.LocalDaemon, pruneChildren bool) *pruner {
	return &pruner{
		useLibImage:     useLibimage,
		libimageRuntime: libimageRuntime,
		localDocker:     dockerAPI,
		pruneChildren:   pruneChildren,
		prunedImgIDs:    make(map[string]struct{}),
	}
}

type imageSummary struct {
	id      string
	created int64
}

func (p *pruner) listImages(ctx context.Context, name string) (imgs []imageSummary, err error) {
	if p.useLibImage {
		imgs, err = p.listImagesLibImage(ctx, name)
		if err != nil {
			return nil, err
		}
	} else {
		imgs, err = p.listImagesDocker(ctx, name)
		if err != nil {
			return nil, err
		}
	}
	if len(imgs) < 2 {
		// no need to sort
		return imgs, nil
	}

	sort.Slice(imgs, func(i, j int) bool {
		// reverse sort
		return imgs[i].created > imgs[j].created
	})

	return imgs, nil
}

func (p *pruner) listImagesDocker(ctx context.Context, name string) (sums []imageSummary, err error) {
	imgs, err := p.localDocker.ImageList(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("docker listing images: %w", err)
	}
	for _, img := range imgs {
		sums = append(sums, imageSummary{
			id:      img.ID,
			created: img.Created,
		})
	}
	return sums, nil
}

func (p *pruner) listImagesLibImage(ctx context.Context, name string) (sums []imageSummary, err error) {
	imgs, err := p.libimageRuntime.ListImages(ctx, []string{name}, &libimage.ListImagesOptions{})
	if err != nil {
		return nil, fmt.Errorf("libimage listing images: %w", err)
	}
	for _, img := range imgs {
		sums = append(sums, imageSummary{
			id:      img.ID(),
			created: img.Created().Unix(),
		})
	}
	return sums, nil
}
func (p *pruner) cleanup(ctx context.Context, sync bool, artifacts []string) {
	toPrune := p.collectImagesToPrune(ctx, artifacts)
	if len(toPrune) == 0 {
		return
	}

	if sync {
		err := p.runPrune(ctx, toPrune)
		if err != nil {
			log.Entry(ctx).Debugf("Failed to prune: %v", err)
		}
	} else {
		go func() {
			err := p.runPrune(ctx, toPrune)
			if err != nil {
				log.Entry(ctx).Debugf("Failed to prune: %v", err)
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
	log.Entry(ctx).Debugf("Going to prune: %v", ids)
	// docker API does not support concurrent prune/utilization info request
	// so let's serialize the access to it
	t0 := time.Now()
	p.pruneMutex.Lock()
	log.Entry(ctx).Tracef("Prune mutex wait time: %v", time.Since(t0))
	defer p.pruneMutex.Unlock()

	beforeDu, err := p.diskUsage(ctx)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		log.Entry(ctx).Debugf("Failed to get docker usage info: %v", err)
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
			log.Entry(ctx).Debugf("Failed to get docker usage info: %v", err)
			return nil
		}
		if beforeDu >= afterDu {
			log.Entry(ctx).Infof("%d image(s) pruned. Reclaimed disk space: %s",
				len(ids), humanize.Bytes(beforeDu-afterDu))
		} else {
			log.Entry(ctx).Infof("%d image(s) pruned", len(ids))
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
			log.Entry(ctx).Warnf("failed to list images: %v", err)
			continue
		}
		for i := imgNameCount[a]; i < len(imgs); i++ {
			rt = append(rt, imgs[i].id)
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
		log.Entry(ctx).Debugf("[%d of %d] failed to get disk usage: %v. Will retry in %v",
			retry, usageRetries, err, usageRetryInterval)
		time.Sleep(usageRetryInterval)
	}

	usage, err := p.localDocker.DiskUsage(ctx)
	if err == nil {
		return usage, nil
	}
	log.Entry(ctx).Debugf("Failed to get usage after: %v. giving up", err)
	return 0, err
}
