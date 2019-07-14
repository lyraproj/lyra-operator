package types

import (
	"io"
	"math"

	"github.com/lyraproj/pcore/px"
)

type CollectionType struct {
	size *IntegerType
}

var CollectionMetaType px.ObjectType

func init() {
	CollectionMetaType = newObjectType(`Pcore::CollectionType`, `Pcore::AnyType {
  attributes => {
    'size_type' => { type => Type[Integer], value => Integer[0] }
  }
}`, func(ctx px.Context, args []px.Value) px.Value {
		return newCollectionType2(args...)
	})
}

func DefaultCollectionType() *CollectionType {
	return collectionTypeDefault
}

func NewCollectionType(size *IntegerType) *CollectionType {
	if size == nil || *size == *IntegerTypePositive {
		return DefaultCollectionType()
	}
	return &CollectionType{size}
}

func newCollectionType2(args ...px.Value) *CollectionType {
	switch len(args) {
	case 0:
		return DefaultCollectionType()
	case 1:
		arg := args[0]
		size, ok := arg.(*IntegerType)
		if !ok {
			sz, ok := toInt(arg)
			if !ok {
				if _, ok := arg.(*DefaultValue); !ok {
					panic(illegalArgumentType(`Collection[]`, 0, `Variant[Integer, Default, Type[Integer]]`, arg))
				}
				sz = 0
			}
			size = NewIntegerType(sz, math.MaxInt64)
		}
		return NewCollectionType(size)
	case 2:
		arg := args[0]
		min, ok := toInt(arg)
		if !ok {
			if _, ok := arg.(*DefaultValue); !ok {
				panic(illegalArgumentType(`Collection[]`, 0, `Variant[Integer, Default]`, arg))
			}
			min = 0
		}
		arg = args[1]
		max, ok := toInt(arg)
		if !ok {
			if _, ok := arg.(*DefaultValue); !ok {
				panic(illegalArgumentType(`Collection[]`, 1, `Variant[Integer, Default]`, arg))
			}
			max = math.MaxInt64
		}
		return NewCollectionType(NewIntegerType(min, max))
	default:
		panic(illegalArgumentCount(`Collection[]`, `0 - 2`, len(args)))
	}
}

func (t *CollectionType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	t.size.Accept(v, g)
}

func (t *CollectionType) Default() px.Type {
	return collectionTypeDefault
}

func (t *CollectionType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*CollectionType); ok {
		return t.size.Equals(ot.size, g)
	}
	return false
}

func (t *CollectionType) Generic() px.Type {
	return collectionTypeDefault
}

func (t *CollectionType) Get(key string) (px.Value, bool) {
	switch key {
	case `size_type`:
		if t.size == nil {
			return px.Undef, true
		}
		return t.size, true
	default:
		return nil, false
	}
}

func (t *CollectionType) IsAssignable(o px.Type, g px.Guard) bool {
	var osz *IntegerType
	switch o := o.(type) {
	case *CollectionType:
		osz = o.size
	case *ArrayType:
		osz = o.size
	case *HashType:
		osz = o.size
	case *TupleType:
		osz = o.givenOrActualSize
	case *StructType:
		n := int64(len(o.elements))
		osz = NewIntegerType(n, n)
	default:
		return false
	}
	return t.size.IsAssignable(osz, g)
}

func (t *CollectionType) IsInstance(o px.Value, g px.Guard) bool {
	return t.IsAssignable(o.PType(), g)
}

func (t *CollectionType) MetaType() px.ObjectType {
	return CollectionMetaType
}

func (t *CollectionType) Name() string {
	return `Collection`
}

func (t *CollectionType) Parameters() []px.Value {
	if *t.size == *IntegerTypePositive {
		return px.EmptyValues
	}
	return t.size.SizeParameters()
}

func (t *CollectionType) CanSerializeAsString() bool {
	return true
}

func (t *CollectionType) SerializationString() string {
	return t.String()
}

func (t *CollectionType) Size() *IntegerType {
	return t.size
}

func (t *CollectionType) String() string {
	return px.ToString2(t, None)
}

func (t *CollectionType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *CollectionType) PType() px.Type {
	return &TypeType{t}
}

var collectionTypeDefault = &CollectionType{IntegerTypePositive}
