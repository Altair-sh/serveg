package server

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type SharedLocation struct {
	// relative to config path at first,
	// then [normalize] makes it absolute
	Path string `toml:"path"`
	// [normalize] removes trailing '/'
	// so "/" becomes ""
	Url           string `toml:"url"`
	ReadPassword  string `toml:"read_password"`
	WritePassword string `toml:"write_password"`
}

var ErrSharePathEmpty = errors.New("share.path is empty")
var ErrShareUrlEmpty = errors.New("share.url is empty")
var ErrShareUrlDoesntStartWithSlash = errors.New("share.url doesn't start with slash")

func (sh *SharedLocation) normalize(configDir string) error {
	if sh.Path == "" {
		return ErrSharePathEmpty
	}
	// get absolute path for configDir
	configDirAbs, err := filepath.Abs(configDir)
	if err != nil {
		return err
	}
	// resolve symlinks
	configDirAbs, err = filepath.EvalSymlinks(configDirAbs)
	if err != nil {
		return err
	}
	// fix separators
	shPathOsSep, err := filepath.Localize(sh.Path)
	if err != nil {
		return err
	}
	// join configDir with sh.Path
	sh.Path = filepath.Join(configDirAbs, shPathOsSep)

	if sh.Url == "" {
		return ErrShareUrlEmpty
	}
	if sh.Url[0] != '/' {
		return ErrShareUrlDoesntStartWithSlash
	}
	// remove trailing slash
	sh.Url = strings.TrimRight(sh.Url, "/")

	return nil
}

func (sh *SharedLocation) String() string {
	var r, w string
	if sh.ReadPassword == "" {
		r = "pub"
	} else {
		r = "pas"
	}
	if sh.WritePassword == "" {
		w = "no "
	} else {
		w = "pas"
	}
	access := "{ " + "r:" + r + ", w:" + w + " }"
	url := sh.Url
	if url == "" {
		url = "/"
	}
	return "share " + access + " '" + url + "' -> '" + sh.Path + "'"
}

func (sh *SharedLocation) mkdir() error {
	_, err := os.Stat(sh.Path)
	if err == os.ErrNotExist {
		return os.MkdirAll(sh.Path, 0o755)
	}
	return err
}

type UrlStatus uint32

const (
	UrlFile = UrlStatus(iota)
	UrlDir
	UrlDoesntExistInShare
	UrlShareNotFound
	UrlForbidden
	UrlIncorrect
)

// returns calculated path, or input url if it doesn't belong to any share
func (shares Shares) MapUrl(url string) (string, *SharedLocation, UrlStatus) {
	if url == "" || url[0] != '/' {
		return "", nil, UrlIncorrect
	}
	for _, sh := range shares {
		pathInShare, ok := strings.CutPrefix(url, sh.Url)
		if !ok {
			continue
		}

		if pathInShare != "" && pathInShare[0] != '/' {
			// url is not sh.Url or child of it
			continue
		}
		pathAbs := filepath.Join(sh.Path, filepath.FromSlash(pathInShare))

		fi, err := os.Stat(pathAbs)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			return pathAbs, sh, UrlDoesntExistInShare
		case err != nil:
			return pathAbs, sh, UrlForbidden
		case fi.IsDir():
			return pathAbs, sh, UrlDir
		default:
			return pathAbs, sh, UrlFile
		}
	}
	return url, nil, UrlShareNotFound
}
