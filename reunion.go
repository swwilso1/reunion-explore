// Package reunion provides a Go API for parsing Reunion genealogy family files.
//
// Use Open() to parse a .familyfile13 or .familyfile14 bundle directory into a structured
// FamilyFile model that can be serialized to JSON.
package reunion

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kedoco/reunion-explore/model"
)

// Open parses the Reunion bundle at bundlePath and returns a FamilyFile.
// The version is detected from the bundle directory extension.
func Open(bundlePath string, opts *ParseOptions) (*model.FamilyFile, error) {
	if opts == nil {
		opts = &ParseOptions{}
	}

	v, err := detectVersion(bundlePath)
	if err != nil {
		return nil, err
	}

	vp, err := getParser(v)
	if err != nil {
		return nil, err
	}

	return vp.Parse(bundlePath, *opts)
}

func detectVersion(bundlePath string) (Version, error) {
	ext := filepath.Ext(bundlePath)
	if !strings.HasPrefix(ext, ".familyfile") {
		return 0, fmt.Errorf("%w: extension %q", ErrNotABundle, ext)
	}
	numStr := strings.TrimPrefix(ext, ".familyfile")
	if numStr == "" {
		return 0, fmt.Errorf("%w: no version number in extension %q", ErrUnsupportedVer, ext)
	}
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrUnsupportedVer, ext)
	}
	return Version(n), nil
}
