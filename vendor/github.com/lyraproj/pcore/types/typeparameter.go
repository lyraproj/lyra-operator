package types

import (
	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
)

type typeParameter struct {
	attribute
}

var TypeTypeParameter = NewStructType([]*StructElement{
	newStructElement2(keyType, DefaultTypeType()),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations),
})

func (t *typeParameter) initHash() *hash.StringHash {
	h := t.attribute.initHash()
	h.Put(keyType, h.Get(keyType, nil).(*TypeType).PType())
	if v, ok := h.Get3(keyValue); ok && v.(px.Value).Equals(undef, nil) {
		h.Delete(keyValue)
	}
	return h
}

func (t *typeParameter) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*typeParameter); ok {
		return t.attribute.Equals(&ot.attribute, g)
	}
	return false
}

func (t *typeParameter) InitHash() px.OrderedMap {
	return WrapStringPValue(t.initHash())
}

func (t *typeParameter) FeatureType() string {
	return `type_parameter`
}
