package reunion

import (
	"fmt"

	"github.com/kedoco/reunion-explore/model"
)

// Version identifies a Reunion file format version.
type Version int

const (
	Version10 Version = 10
	Version13 Version = 13
	Version14 Version = 14
)

// ParseOptions controls parsing behavior.
type ParseOptions struct {
	// MaxErrors is the maximum number of non-fatal errors to collect before
	// aborting. Zero means no limit.
	MaxErrors int
}

// VersionParser is the interface each version-specific parser must implement.
type VersionParser interface {
	Version() Version
	CanParse(bundlePath string) (bool, error)
	Parse(bundlePath string, opts ParseOptions) (*model.FamilyFile, error)
}

var registry = map[Version]VersionParser{}

// RegisterVersion registers a version-specific parser.
func RegisterVersion(vp VersionParser) {
	registry[vp.Version()] = vp
}

func getParser(v Version) (VersionParser, error) {
	vp, ok := registry[v]
	if !ok {
		return nil, fmt.Errorf("%w: version %d", ErrUnsupportedVer, v)
	}
	return vp, nil
}
