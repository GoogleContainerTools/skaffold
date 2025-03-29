package builder

type LabelManagerProvider struct{}

func NewLabelManagerProvider() *LabelManagerProvider {
	return &LabelManagerProvider{}
}

func (p *LabelManagerProvider) BuilderLabelManager(inspectable Inspectable) LabelInspector {
	return NewLabelManager(inspectable)
}
