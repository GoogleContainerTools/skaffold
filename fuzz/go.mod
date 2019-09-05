module fuzz.test/fuzz

go 1.13

replace github.com/GoogleContainerTools/skaffold => ../

replace gopkg.in/russross/blackfriday.v2 v2.0.1 => github.com/russross/blackfriday/v2 v2.0.1

require (
	github.com/GoogleContainerTools/skaffold v0.0.0-00010101000000-000000000000
	github.com/dvyukov/go-fuzz v0.0.0-20190808141544-193030f1cb16
	github.com/fuzzitdev/fuzzit/v2 v2.4.46
	golang.org/x/net v0.0.0-20190909003024-a7b16738d86b
)
