package str

import "fmt"

type Slice []string

func (s *Slice) String() string {
	return fmt.Sprintf("%+v", *s)
}

func (s *Slice) Set(value string) error {
	*s = append(*s, value)
	return nil
}
