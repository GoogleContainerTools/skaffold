package builder

import (
	"encoding/json"
	"fmt"

	"github.com/buildpacks/pack/pkg/dist"

	"github.com/buildpacks/pack/internal/stack"
)

type LabelManager struct {
	inspectable Inspectable
}

func NewLabelManager(inspectable Inspectable) *LabelManager {
	return &LabelManager{inspectable: inspectable}
}

func (m *LabelManager) Metadata() (Metadata, error) {
	var parsedMetadata Metadata
	err := m.labelJSON(metadataLabel, &parsedMetadata)
	return parsedMetadata, err
}

func (m *LabelManager) StackID() (string, error) {
	return m.labelContent(stackLabel)
}

func (m *LabelManager) Mixins() ([]string, error) {
	parsedMixins := []string{}
	err := m.labelJSONDefaultEmpty(stack.MixinsLabel, &parsedMixins)
	return parsedMixins, err
}

func (m *LabelManager) Order() (dist.Order, error) {
	parsedOrder := dist.Order{}
	err := m.labelJSONDefaultEmpty(OrderLabel, &parsedOrder)
	return parsedOrder, err
}

func (m *LabelManager) BuildpackLayers() (dist.BuildpackLayers, error) {
	parsedLayers := dist.BuildpackLayers{}
	err := m.labelJSONDefaultEmpty(dist.BuildpackLayersLabel, &parsedLayers)
	return parsedLayers, err
}

func (m *LabelManager) labelContent(labelName string) (string, error) {
	content, err := m.inspectable.Label(labelName)
	if err != nil {
		return "", fmt.Errorf("getting label %s: %w", labelName, err)
	}

	if content == "" {
		return "", fmt.Errorf("builder missing label %s -- try recreating builder", labelName)
	}

	return content, nil
}

func (m *LabelManager) labelJSON(labelName string, targetObject interface{}) error {
	rawContent, err := m.labelContent(labelName)
	if err != nil {
		return err
	}

	err = json.Unmarshal([]byte(rawContent), targetObject)
	if err != nil {
		return fmt.Errorf("parsing label content for %s: %w", labelName, err)
	}

	return nil
}

func (m *LabelManager) labelJSONDefaultEmpty(labelName string, targetObject interface{}) error {
	rawContent, err := m.inspectable.Label(labelName)
	if err != nil {
		return fmt.Errorf("getting label %s: %w", labelName, err)
	}

	if rawContent == "" {
		return nil
	}

	err = json.Unmarshal([]byte(rawContent), targetObject)
	if err != nil {
		return fmt.Errorf("parsing label content for %s: %w", labelName, err)
	}

	return nil
}
