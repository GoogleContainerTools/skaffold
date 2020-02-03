package local

type IDIdentifier struct {
	ImageID string
}

func (i IDIdentifier) String() string {
	return i.ImageID
}
