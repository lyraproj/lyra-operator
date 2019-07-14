package yaml

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	ym "gopkg.in/yaml.v2"
)

func Unmarshal(c px.Context, data []byte) px.Value {
	ms := make(ym.MapSlice, 0)
	err := ym.Unmarshal([]byte(data), &ms)
	if err != nil {
		var itm interface{}
		err2 := ym.Unmarshal([]byte(data), &itm)
		if err2 != nil {
			panic(px.Error(px.ParseError, issue.H{`language`: `YAML`, `detail`: err.Error()}))
		}
		return wrapValue(c, itm)
	}
	return wrapSlice(c, ms)
}

func wrapSlice(c px.Context, ms ym.MapSlice) px.Value {
	es := make([]*types.HashEntry, len(ms))
	for i, me := range ms {
		es[i] = types.WrapHashEntry(wrapValue(c, me.Key), wrapValue(c, me.Value))
	}
	return types.WrapHash(es)
}

func wrapValue(c px.Context, v interface{}) px.Value {
	switch v := v.(type) {
	case ym.MapSlice:
		return wrapSlice(c, v)
	case []interface{}:
		vs := make([]px.Value, len(v))
		for i, y := range v {
			vs[i] = wrapValue(c, y)
		}
		return types.WrapValues(vs)
	default:
		return px.Wrap(c, v)
	}
}
