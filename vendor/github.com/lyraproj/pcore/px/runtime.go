package px

import (
	"github.com/lyraproj/semver/semver"
)

type (
	URI string
)

const (
	KeyPcoreUri     = `pcore_uri`
	KeyPcoreVersion = `pcore_version`

	RuntimeNameAuthority = URI(`http://puppet.com/2016.1/runtime`)
	PcoreUri             = URI(`http://puppet.com/2016.1/pcore`)
)

var PcoreVersion, _ = semver.NewVersion3(1, 0, 0, ``, ``)
var ParsablePcoreVersions, _ = semver.ParseVersionRange(`1.x`)
