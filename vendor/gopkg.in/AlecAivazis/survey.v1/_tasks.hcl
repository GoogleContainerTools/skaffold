task "install-deps" {
    description = "Install all of package dependencies"
    pipeline = [
        "go get -t {{.files}}",
        # for autoplay tests
        "go get github.com/kr/pty"
    ]
}

task "tests" {
    description = "Run the test suite"
    command = "go test {{.files}}"
}

variables {
    files = "$(go list -v ./... | grep -iEv \"tests|examples\")"
}

