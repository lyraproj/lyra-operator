package types

import (
	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
)

var annotationTypeDefault = &objectType{
	annotatable:         annotatable{annotations: emptyMap},
	hashKey:             px.HashKey("\x00tAnnotation"),
	name:                `Annotation`,
	parameters:          hash.EmptyStringHash,
	attributes:          hash.EmptyStringHash,
	functions:           hash.EmptyStringHash,
	equalityIncludeType: true,
	equality:            nil}

func DefaultAnnotationType() px.Type {
	return annotationTypeDefault
}

var typeAnnotations = NewHashType(NewTypeType(annotationTypeDefault), DefaultHashType(), nil)

type annotatable struct {
	annotations         *Hash
	resolvedAnnotations *Hash
}

func (a *annotatable) Annotations(c px.Context) px.OrderedMap {
	if a.resolvedAnnotations == nil {
		ah := a.annotations
		if ah.IsEmpty() {
			a.resolvedAnnotations = emptyMap
		} else {
			as := make([]*HashEntry, 0, ah.Len())
			ah.EachPair(func(k, v px.Value) {
				at := k.(px.ObjectType)
				as = append(as, WrapHashEntry(k, px.New(c, at, v)))
			})
			a.resolvedAnnotations = WrapHash(as)
		}
	}
	return a.resolvedAnnotations
}

func (a *annotatable) initialize(initHash *Hash) {
	a.annotations = hashArg(initHash, keyAnnotations)
}

func (a *annotatable) initHash() *hash.StringHash {
	h := hash.NewStringHash(5)
	if a.annotations.Len() > 0 {
		h.Put(keyAnnotations, a.annotations)
	}
	return h
}
