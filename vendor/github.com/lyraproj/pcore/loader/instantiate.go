package loader

import (
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

// InstantiatePuppetType reads the contents a puppet manifest file and parses it using
// the types.Parse() function.
func InstantiatePuppetType(ctx px.Context, loader ContentProvidingLoader, tn px.TypedName, sources []string) {
	content := string(loader.GetContent(ctx, sources[0]))
	dt, err := types.Parse(content)
	if err != nil {
		panic(err)
	}
	if nt, ok := dt.(px.Type); ok {
		if !strings.EqualFold(tn.Name(), nt.Name()) {
			panic(px.Error(px.WrongDefinition, issue.H{`source`: sources[0], `type`: px.NsType, `expected`: tn.Name(), `actual`: nt.Name()}))
		}
		px.AddTypes(ctx, nt)
	} else {
		px.AddTypes(ctx, types.NamedType(tn.Authority(), tn.Name(), dt))
	}
}
