package cache

const (
	Image Type = iota
	Volume
	Bind
)

type Type int
