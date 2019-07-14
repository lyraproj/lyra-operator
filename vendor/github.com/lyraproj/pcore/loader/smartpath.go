package loader

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lyraproj/pcore/px"
)

type (
	Instantiator func(ctx px.Context, loader ContentProvidingLoader, tn px.TypedName, sources []string)

	SmartPath interface {
		Loader() px.Loader
		GenericPath() string
		EffectivePath(name px.TypedName) string
		Extension() string
		RelativePath() string
		Namespaces() []px.Namespace
		IsMatchMany() bool
		PreferredOrigin(i []string) string
		TypedNames(nameAuthority px.URI, relativePath string) []px.TypedName
		Instantiator() Instantiator
		Indexed() bool
		SetIndexed()
	}

	smartPath struct {
		relativePath string
		loader       px.ModuleLoader
		namespaces   []px.Namespace
		extension    string

		// Paths are not supposed to contain module name
		moduleNameRelative bool
		matchMany          bool
		instantiator       Instantiator
		indexed            bool
	}
)

func NewSmartPath(relativePath, extension string,
	loader px.ModuleLoader, namespaces []px.Namespace, moduleNameRelative,
	matchMany bool, instantiator Instantiator) SmartPath {
	return &smartPath{relativePath: relativePath, extension: extension,
		loader: loader, namespaces: namespaces, moduleNameRelative: moduleNameRelative,
		matchMany: matchMany, instantiator: instantiator, indexed: false}
}

func (p *smartPath) Indexed() bool {
	return p.indexed
}

func (p *smartPath) SetIndexed() {
	p.indexed = true
}

func (p *smartPath) Loader() px.Loader {
	return p.loader
}

func (p *smartPath) EffectivePath(name px.TypedName) string {
	nameParts := name.Parts()
	if p.moduleNameRelative {
		if len(nameParts) < 2 || nameParts[0] != p.loader.ModuleName() {
			return ``
		}
		nameParts = nameParts[1:]
	}

	parts := make([]string, 0, len(nameParts)+2)
	parts = append(parts, p.loader.Path()) // system, environment, or module root
	if p.relativePath != `` {
		parts = append(parts, p.relativePath)
	}
	parts = append(parts, nameParts...)
	return filepath.Join(parts...) + p.extension
}

func (p *smartPath) GenericPath() string {
	parts := make([]string, 0)
	parts = append(parts, p.loader.Path()) // system, environment, or module root
	if p.relativePath != `` {
		parts = append(parts, p.relativePath)
	}
	return filepath.Join(parts...)
}

func (p *smartPath) Namespaces() []px.Namespace {
	return p.namespaces
}

func (p *smartPath) Extension() string {
	return p.extension
}

func (p *smartPath) RelativePath() string {
	return p.relativePath
}

func (p *smartPath) IsMatchMany() bool {
	return p.matchMany
}

func (p *smartPath) PreferredOrigin(origins []string) string {
	if len(origins) == 1 {
		return origins[0]
	}
	if p.namespaces[0] == px.NsTask {
		// Prefer .json file if present
		for _, origin := range origins {
			if strings.HasSuffix(origin, `.json`) {
				return origin
			}
		}
	}
	return origins[0]
}

var dropExtension = regexp.MustCompile(`\.[^\\/]*\z`)

func (p *smartPath) TypedNames(nameAuthority px.URI, relativePath string) []px.TypedName {
	parts := strings.Split(relativePath, `/`)
	l := len(parts) - 1
	s := parts[l]
	if p.extension == `` {
		s = dropExtension.ReplaceAllLiteralString(s, ``)
	} else {
		s = s[:len(s)-len(p.extension)]
	}
	parts[l] = s

	if p.moduleNameRelative && !(len(parts) == 1 && (s == `init` || s == `init_typeset`)) {
		parts = append([]string{p.loader.ModuleName()}, parts...)
	}
	ts := make([]px.TypedName, len(p.namespaces))
	for i, n := range p.namespaces {
		ts[i] = px.NewTypedName2(n, strings.Join(parts, `::`), nameAuthority)
	}
	return ts
}

func (p *smartPath) Instantiator() Instantiator {
	return p.instantiator
}
