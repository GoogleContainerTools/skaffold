package godirwalk

// The functions in this file are mere wrappers of what is already provided by
// standard library, in order to provide the same API as this library provides.
//
// The scratch buffer parameter in these functions is the underscore because
// presently that parameter is ignored by the functions for this architecture.
//
// Please send PR or link to article if you know of a more performant way of
// enumerating directory contents and mode types on Windows.

import "os"

func readdirents(osDirname string, _ []byte) (Dirents, error) {
	dh, err := os.Open(osDirname)
	if err != nil {
		return nil, err
	}

	fileinfos, err := dh.Readdir(0)
	if er := dh.Close(); err == nil {
		err = er
	}
	if err != nil {
		return nil, err
	}

	entries := make(Dirents, len(fileinfos))
	for i, info := range fileinfos {
		entries[i] = &Dirent{name: info.Name(), modeType: info.Mode() & os.ModeType}
	}

	return entries, nil
}

func readdirnames(osDirname string, _ []byte) ([]string, error) {
	dh, err := os.Open(osDirname)
	if err != nil {
		return nil, err
	}

	entries, err := dh.Readdirnames(0)
	if er := dh.Close(); err == nil {
		err = er
	}
	if err != nil {
		return nil, err
	}

	return entries, nil
}
