package builder

import (
	pubbldr "github.com/buildpacks/pack/builder"
	"github.com/buildpacks/pack/pkg/dist"
)

type DetectionOrderCalculator struct{}

func NewDetectionOrderCalculator() *DetectionOrderCalculator {
	return &DetectionOrderCalculator{}
}

type detectionOrderRecurser struct {
	layers   dist.BuildpackLayers
	maxDepth int
}

func newDetectionOrderRecurser(layers dist.BuildpackLayers, maxDepth int) *detectionOrderRecurser {
	return &detectionOrderRecurser{
		layers:   layers,
		maxDepth: maxDepth,
	}
}

func (c *DetectionOrderCalculator) Order(
	order dist.Order,
	layers dist.BuildpackLayers,
	maxDepth int,
) (pubbldr.DetectionOrder, error) {
	recurser := newDetectionOrderRecurser(layers, maxDepth)

	return recurser.detectionOrderFromOrder(order, dist.BuildpackRef{}, 0, map[string]interface{}{}), nil
}

func (r *detectionOrderRecurser) detectionOrderFromOrder(
	order dist.Order,
	parentBuildpack dist.BuildpackRef,
	currentDepth int,
	visited map[string]interface{},
) pubbldr.DetectionOrder {
	var detectionOrder pubbldr.DetectionOrder
	for _, orderEntry := range order {
		visitedCopy := copyMap(visited)
		groupDetectionOrder := r.detectionOrderFromGroup(orderEntry.Group, currentDepth, visitedCopy)

		detectionOrderEntry := pubbldr.DetectionOrderEntry{
			BuildpackRef:        parentBuildpack,
			GroupDetectionOrder: groupDetectionOrder,
		}

		detectionOrder = append(detectionOrder, detectionOrderEntry)
	}

	return detectionOrder
}

func (r *detectionOrderRecurser) detectionOrderFromGroup(
	group []dist.BuildpackRef,
	currentDepth int,
	visited map[string]interface{},
) pubbldr.DetectionOrder {
	var groupDetectionOrder pubbldr.DetectionOrder

	for _, bp := range group {
		_, bpSeen := visited[bp.FullName()]
		if !bpSeen {
			visited[bp.FullName()] = true
		}

		layer, ok := r.layers.Get(bp.ID, bp.Version)
		if ok && len(layer.Order) > 0 && r.shouldGoDeeper(currentDepth) && !bpSeen {
			groupOrder := r.detectionOrderFromOrder(layer.Order, bp, currentDepth+1, visited)
			groupDetectionOrder = append(groupDetectionOrder, groupOrder...)
		} else {
			groupDetectionOrderEntry := pubbldr.DetectionOrderEntry{
				BuildpackRef: bp,
				Cyclical:     bpSeen,
			}
			groupDetectionOrder = append(groupDetectionOrder, groupDetectionOrderEntry)
		}
	}

	return groupDetectionOrder
}

func (r *detectionOrderRecurser) shouldGoDeeper(currentDepth int) bool {
	if r.maxDepth == pubbldr.OrderDetectionMaxDepth {
		return true
	}

	if currentDepth < r.maxDepth {
		return true
	}

	return false
}

func copyMap(toCopy map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(toCopy))
	for key := range toCopy {
		result[key] = true
	}

	return result
}
