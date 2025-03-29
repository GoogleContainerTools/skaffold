package fakes

type FakeInspectable struct {
	ReturnForLabel string

	ErrorForLabel error

	ReceivedName string
}

func (f *FakeInspectable) Label(name string) (string, error) {
	f.ReceivedName = name

	return f.ReturnForLabel, f.ErrorForLabel
}
