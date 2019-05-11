package types

import (
	"io"
	"reflect"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

var deferredMetaType px.ObjectType

func init() {
	deferredMetaType = newGoObjectType(`DeferredType`, reflect.TypeOf(&DeferredType{}), `{}`)
}

type DeferredType struct {
	tn       string
	params   []px.Value
	resolved px.Type
}

func NewDeferredType(name string, params ...px.Value) *DeferredType {
	return &DeferredType{tn: name, params: params}
}

func (dt *DeferredType) String() string {
	return px.ToString(dt)
}

func (dt *DeferredType) Equals(other interface{}, guard px.Guard) bool {
	if ot, ok := other.(*DeferredType); ok {
		return dt.tn == ot.tn
	}
	return false
}

func (dt *DeferredType) ToString(bld io.Writer, s px.FormatContext, g px.RDetect) {
	utils.WriteString(bld, `DeferredType(`)
	utils.WriteString(bld, dt.tn)
	if dt.params != nil {
		utils.WriteString(bld, `, `)
		WrapValues(dt.params).ToString(bld, s.Subsequent(), g)
	}
	utils.WriteByte(bld, ')')
}

func (dt *DeferredType) PType() px.Type {
	return deferredMetaType
}

func (dt *DeferredType) Name() string {
	return dt.tn
}

func (dt *DeferredType) Resolve(c px.Context) px.Type {
	if dt.resolved == nil {
		if dt.params != nil {
			if dt.Name() == `TypeSet` && len(dt.params) == 1 {
				if ih, ok := dt.params[0].(px.OrderedMap); ok {
					dt.resolved = newTypeSetType2(ih, c.Loader())
				}
			} else {
				ar := resolveValue(c, WrapValues(dt.params)).(*Array)
				dt.resolved = ResolveWithParams(c, dt.tn, ar.AppendTo(make([]px.Value, 0, ar.Len())))
			}
		} else {
			dt.resolved = Resolve(c, dt.tn)
		}
	}
	return dt.resolved
}

func (dt *DeferredType) Parameters() []px.Value {
	return dt.params
}

func resolveValue(c px.Context, v px.Value) (rv px.Value) {
	switch v := v.(type) {
	case *DeferredType:
		rv = v.Resolve(c)
	case Deferred:
		rv = v.Resolve(c, emptyMap)
	case *Array:
		rv = v.Map(func(e px.Value) px.Value { return resolveValue(c, e) })
	case *HashEntry:
		rv = resolveEntry(c, v)
	case *Hash:
		rv = v.MapEntries(func(he px.MapEntry) px.MapEntry { return resolveEntry(c, he) })
	default:
		rv = v
	}
	return
}

func resolveEntry(c px.Context, he px.MapEntry) px.MapEntry {
	return WrapHashEntry(resolveValue(c, he.Key()), resolveValue(c, he.Value()))
}
