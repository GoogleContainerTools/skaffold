package paths

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var schemeRegexp = regexp.MustCompile(`^.+://.*`)

func IsURI(ref string) bool {
	return schemeRegexp.MatchString(ref)
}

func IsDir(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	return fileInfo.IsDir(), nil
}

func FilePathToURI(path string) (string, error) {
	var err error
	if !filepath.IsAbs(path) {
		path, err = filepath.Abs(path)
		if err != nil {
			return "", err
		}
	}

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(path, `\\`) {
			return "file://" + filepath.ToSlash(strings.TrimPrefix(path, `\\`)), nil
		}
		return "file:///" + filepath.ToSlash(path), nil
	}
	return "file://" + path, nil
}

// examples:
//
// - unix file: file://laptop/some%20dir/file.tgz
//
// - windows drive: file:///C:/Documents%20and%20Settings/file.tgz
//
// - windows share: file://laptop/My%20Documents/file.tgz
//
func URIToFilePath(uri string) (string, error) {
	var (
		osPath string
		err    error
	)

	osPath = filepath.FromSlash(strings.TrimPrefix(uri, "file://"))

	if osPath, err = url.PathUnescape(osPath); err != nil {
		return "", nil
	}

	if runtime.GOOS == "windows" {
		if strings.HasPrefix(osPath, `\`) {
			return strings.TrimPrefix(osPath, `\`), nil
		}
		return `\\` + osPath, nil
	}
	return osPath, nil
}

func ToAbsolute(uri, relativeTo string) (string, error) {
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", err
	}

	if parsed.Scheme == "" {
		if !filepath.IsAbs(parsed.Path) {
			absPath := filepath.Join(relativeTo, parsed.Path)
			return FilePathToURI(absPath)
		}
	}

	return uri, nil
}

func FilterReservedNames(path string) string {
	// The following keys are reserved on Windows
	// https://docs.microsoft.com/en-us/windows/win32/fileio/naming-a-file?redirectedfrom=MSDN#win32-file-namespaces
	reservedNameConversions := map[string]string{
		"aux": "a_u_x",
		"com": "c_o_m",
		"con": "c_o_n",
		"lpt": "l_p_t",
		"nul": "n_u_l",
		"prn": "p_r_n",
	}
	for k, v := range reservedNameConversions {
		path = strings.Replace(path, k, v, -1)
	}

	return path
}
