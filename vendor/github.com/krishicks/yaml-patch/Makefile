CGO_ENABLED=0

all: windows linux darwin

linux:
	GOOS=linux GOARCH=amd64 go build -o yaml_patch_linux cmd/yaml-patch/*.go

windows:
	GOOS=windows GOARCH=amd64 go build -o yaml_patch.exe cmd/yaml-patch/*.go

darwin:
	GOOS=darwin GOARCH=amd64 go build -o yaml_patch_darwin cmd/yaml-patch/*.go

clean:
	rm yaml_patch_linux
	rm yaml_patch.exe
	rm yaml_patch_darwin
