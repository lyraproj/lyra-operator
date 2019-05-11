package pximpl

import (
	"io"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type parameter struct {
	name     string
	typ      px.Type
	value    px.Value
	captures bool
}

func init() {
	px.NewParameter = newParameter
}

func newParameter(name string, typ px.Type, value px.Value, capturesRest bool) px.Parameter {
	return &parameter{name, typ, value, capturesRest}
}

func (p *parameter) HasValue() bool {
	return p.value != nil
}

func (p *parameter) Name() string {
	return p.name
}

func (p *parameter) Value() px.Value {
	return p.value
}

func (p *parameter) Type() px.Type {
	return p.typ
}

func (p *parameter) CapturesRest() bool {
	return p.captures
}

func (p *parameter) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `name`:
		return types.WrapString(p.name), true
	case `type`:
		return p.typ, true
	case `value`:
		return p.Value(), true
	case `has_value`:
		return types.WrapBoolean(p.value != nil), true
	case `captures_rest`:
		return types.WrapBoolean(p.captures), true
	}
	return nil, false
}

func (p *parameter) InitHash() px.OrderedMap {
	es := make([]*types.HashEntry, 0, 3)
	es = append(es, types.WrapHashEntry2(`name`, types.WrapString(p.name)))
	es = append(es, types.WrapHashEntry2(`type`, p.typ))
	if p.value != nil {
		es = append(es, types.WrapHashEntry2(`value`, p.value))
	}
	if p.value == px.Undef {
		es = append(es, types.WrapHashEntry2(`has_value`, types.BooleanTrue))
	}
	if p.captures {
		es = append(es, types.WrapHashEntry2(`captures_rest`, types.BooleanTrue))
	}
	return types.WrapHash(es)
}

var ParameterMetaType px.Type

func (p *parameter) Equals(other interface{}, guard px.Guard) bool {
	return p == other
}

func (p *parameter) String() string {
	return px.ToString(p)
}

func (p *parameter) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	types.ObjectToString(p, format, bld, g)
}

func (p *parameter) PType() px.Type {
	return ParameterMetaType
}

func init() {
	ParameterMetaType = px.NewObjectType(`Parameter`, `{
    attributes => {
      'name' => String,
      'type' => Type,
      'has_value' => { type => Boolean, value => false },
      'value' => { type => Variant[Deferred,Data], value => undef },
      'captures_rest' => { type => Boolean, value => false },
    }
  }`, func(ctx px.Context, args []px.Value) px.Value {
		n := args[0].String()
		t := args[1].(px.Type)
		h := false
		if len(args) > 2 {
			h = args[2].(px.Boolean).Bool()
		}
		var v px.Value
		if len(args) > 3 {
			v = args[3]
		}
		c := false
		if len(args) > 4 {
			c = args[4].(px.Boolean).Bool()
		}
		if h && v == nil {
			v = px.Undef
		}
		return newParameter(n, t, v, c)
	}, func(ctx px.Context, args []px.Value) px.Value {
		h := args[0].(*types.Hash)
		n := h.Get5(`name`, px.EmptyString).String()
		t := h.Get5(`type`, types.DefaultDataType()).(px.Type)
		var v px.Value
		if x, ok := h.Get4(`value`); ok {
			v = x
		}
		hv := false
		if x, ok := h.Get4(`has_value`); ok {
			hv = x.(px.Boolean).Bool()
		}
		c := false
		if x, ok := h.Get4(`captures_rest`); ok {
			c = x.(px.Boolean).Bool()
		}
		if hv && v == nil {
			v = px.Undef
		}
		return newParameter(n, t, v, c)
	})
}
