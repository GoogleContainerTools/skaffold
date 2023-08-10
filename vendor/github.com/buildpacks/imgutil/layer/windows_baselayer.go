package layer

import (
	"archive/tar"
	"bytes"
	"io"
)

// Generate using `make generate`
//go:generate docker run --rm -v $PWD:/out/ bcdhive-generator -file=/out/layer/bcdhive_generated.go -package=layer -func=BaseLayerBCD

// Windows base layers must follow this pattern:
//  \-> UtilityVM/Files/EFI/Microsoft/Boot/BCD   (file must exist and a valid BCD format - from bcdhive_gen)
//  \-> Files/Windows/System32/config/DEFAULT   (file and must exist but can be empty)
//  \-> Files/Windows/System32/config/SAM       (file must exist but can be empty)
//  \-> Files/Windows/System32/config/SECURITY  (file must exist but can be empty)
//  \-> Files/Windows/System32/config/SOFTWARE  (file must exist but can be empty)
//  \-> Files/Windows/System32/config/SYSTEM    (file must exist but can be empty)
// Refs:
// https://github.com/microsoft/hcsshim/blob/master/internal/wclayer/legacy.go
// https://github.com/moby/moby/blob/master/daemon/graphdriver/windows/windows.go#L48
func WindowsBaseLayer() (io.Reader, error) {
	bcdBytes, err := BaseLayerBCD()
	if err != nil {
		return nil, err
	}

	layerBuffer := &bytes.Buffer{}
	tw := tar.NewWriter(layerBuffer)

	if err := tw.WriteHeader(&tar.Header{Name: "UtilityVM", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "UtilityVM/Files", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "UtilityVM/Files/EFI", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "UtilityVM/Files/EFI/Microsoft", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "UtilityVM/Files/EFI/Microsoft/Boot", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}

	if err := tw.WriteHeader(&tar.Header{Name: "UtilityVM/Files/EFI/Microsoft/Boot/BCD", Size: int64(len(bcdBytes)), Mode: 0644}); err != nil {
		return nil, err
	}
	if _, err := tw.Write(bcdBytes); err != nil {
		return nil, err
	}

	if err := tw.WriteHeader(&tar.Header{Name: "Files", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows/System32", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows/System32/config", Typeflag: tar.TypeDir}); err != nil {
		return nil, err
	}

	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows/System32/config/DEFAULT", Size: 0, Mode: 0644}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows/System32/config/SAM", Size: 0, Mode: 0644}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows/System32/config/SECURITY", Size: 0, Mode: 0644}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows/System32/config/SOFTWARE", Size: 0, Mode: 0644}); err != nil {
		return nil, err
	}
	if err := tw.WriteHeader(&tar.Header{Name: "Files/Windows/System32/config/SYSTEM", Size: 0, Mode: 0644}); err != nil {
		return nil, err
	}

	return layerBuffer, nil
}
