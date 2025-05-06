package pkg

import (
	"fmt"

	"go.rtnl.ai/x/semver"
)

// Version component constants for the current build.
const (
	VersionMajor         = 1
	VersionMinor         = 0
	VersionPatch         = 0
	VersionReleaseLevel  = "rc"
	VersionReleaseNumber = 8
)

// Set the GitVersion via -ldflags="-X 'github.com/trisacrypto/envoy/pkg.GitVersion=$(git rev-parse --short HEAD)'"
var GitVersion string

// Version returns the semantic version for the current build.
func Version(short bool) string {
	vers := semver.Version{
		Major: VersionMajor,
		Minor: VersionMinor,
		Patch: VersionPatch,
	}

	if VersionReleaseLevel != "" && VersionReleaseLevel != "final" {
		if VersionReleaseNumber > 0 {
			vers.PreRelease = fmt.Sprintf("%s.%d", VersionReleaseLevel, VersionReleaseNumber)
		} else {
			vers.PreRelease = VersionReleaseLevel
		}
	}

	if short {
		return vers.Short()
	}
	return vers.String()
}
