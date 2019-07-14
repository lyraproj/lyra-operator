package types

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

var coreTypes map[string]px.Type

func Resolve(c px.Context, tn string) px.Type {
	pt := coreTypes[tn]
	if pt != nil {
		return pt
	}
	return loadType(c, tn)
}

func ResolveWithParams(c px.Context, name string, args []px.Value) px.Type {
	t := Resolve(c, name)
	if oo, ok := t.(px.ObjectType); ok && oo.IsParameterized() {
		return NewObjectTypeExtension(c, oo, args)
	}
	if pt, ok := t.(px.ParameterizedType); ok {
		mt := pt.MetaType().(*objectType)
		if mt.creators != nil {
			if posCtor := mt.creators[0]; posCtor != nil {
				return posCtor(c, args).(px.Type)
			}
		}
	}
	panic(px.Error(px.NotParameterizedType, issue.H{`type`: name}))
}

func loadType(c px.Context, name string) px.Type {
	if c == nil {
		return nil
	}
	tn := newTypedName2(px.NsType, name, c.Loader().NameAuthority())
	found, ok := px.Load(c, tn)
	if ok {
		return found.(px.Type)
	}
	return NewTypeReferenceType(name)
}
