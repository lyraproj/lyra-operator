package types

import (
	"io"
	"reflect"
	"strings"

	"github.com/lyraproj/pcore/px"
)

var BooleanFalse = booleanValue(false)
var BooleanTrue = booleanValue(true)

type (
	BooleanType struct {
		value int // -1 == unset, 0 == false, 1 == true
	}

	// booleanValue represents bool as a pcore.Value
	booleanValue bool
)

var booleanTypeDefault = &BooleanType{-1}

var BooleanMetaType px.ObjectType

func init() {
	BooleanMetaType = newObjectType(`Pcore::BooleanType`, `Pcore::ScalarDataType {
  attributes => {
    value => { type => Optional[Boolean], value => undef }
  }
}`, func(ctx px.Context, args []px.Value) px.Value {
		return newBooleanType2(args...)
	})

	newGoConstructor(`Boolean`,
		func(d px.Dispatch) {
			d.Param(`Variant[Integer, Float, Boolean, Enum['false','true','yes','no','y','n',true]]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				switch arg := args[0].(type) {
				case integerValue:
					if arg == 0 {
						return BooleanFalse
					}
					return BooleanTrue
				case floatValue:
					if arg == 0.0 {
						return BooleanFalse
					}
					return BooleanTrue
				case booleanValue:
					return arg
				default:
					switch strings.ToLower(arg.String()) {
					case `false`, `no`, `n`:
						return BooleanFalse
					default:
						return BooleanTrue
					}
				}
			})
		},
	)
}

func DefaultBooleanType() *BooleanType {
	return booleanTypeDefault
}

func NewBooleanType(value bool) *BooleanType {
	n := 0
	if value {
		n = 1
	}
	return &BooleanType{n}
}

func newBooleanType2(args ...px.Value) *BooleanType {
	switch len(args) {
	case 0:
		return DefaultBooleanType()
	case 1:
		if bv, ok := args[0].(booleanValue); ok {
			return NewBooleanType(bool(bv))
		}
		panic(illegalArgumentType(`Boolean[]`, 0, `Boolean`, args[0]))
	default:
		panic(illegalArgumentCount(`Boolean[]`, `0 or 1`, len(args)))
	}
}

func (t *BooleanType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *BooleanType) Default() px.Type {
	return booleanTypeDefault
}

func (t *BooleanType) Generic() px.Type {
	return booleanTypeDefault
}

func (t *BooleanType) Equals(o interface{}, g px.Guard) bool {
	if bo, ok := o.(*BooleanType); ok {
		return t.value == bo.value
	}
	return false
}

func (t *BooleanType) Get(key string) (px.Value, bool) {
	switch key {
	case `value`:
		switch t.value {
		case 0:
			return BooleanFalse, true
		case 1:
			return BooleanTrue, true
		default:
			return px.Undef, true
		}
	default:
		return nil, false
	}
}

func (t *BooleanType) MetaType() px.ObjectType {
	return BooleanMetaType
}

func (t *BooleanType) Name() string {
	return `Boolean`
}

func (t *BooleanType) String() string {
	switch t.value {
	case 0:
		return `Boolean[false]`
	case 1:
		return `Boolean[true]`
	default:
		return `Boolean`
	}
}

func (t *BooleanType) IsAssignable(o px.Type, g px.Guard) bool {
	if bo, ok := o.(*BooleanType); ok {
		return t.value == -1 || t.value == bo.value
	}
	return false
}

func (t *BooleanType) IsInstance(o px.Value, g px.Guard) bool {
	if bo, ok := o.(booleanValue); ok {
		return t.value == -1 || bool(bo) == (t.value == 1)
	}
	return false
}

func (t *BooleanType) Parameters() []px.Value {
	if t.value == -1 {
		return px.EmptyValues
	}
	return []px.Value{booleanValue(t.value == 1)}
}

func (t *BooleanType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(true), true
}

func (t *BooleanType) CanSerializeAsString() bool {
	return true
}

func (t *BooleanType) SerializationString() string {
	return t.String()
}

func (t *BooleanType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *BooleanType) PType() px.Type {
	return &TypeType{t}
}

func WrapBoolean(val bool) px.Boolean {
	if val {
		return BooleanTrue
	}
	return BooleanFalse
}

func (bv booleanValue) Bool() bool {
	return bool(bv)
}

func (bv booleanValue) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(booleanValue); ok {
		return bv == ov
	}
	return false
}

func (bv booleanValue) Float() float64 {
	if bv {
		return float64(1.0)
	}
	return float64(0.0)
}

func (bv booleanValue) Int() int64 {
	if bv {
		return int64(1)
	}
	return int64(0)
}

func (bv booleanValue) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf(bool(bv))
}

var theTrue = true
var theFalse = false
var theTruePtr = &theTrue
var theFalsePtr = &theFalse

var reflectTrue = reflect.ValueOf(theTrue)
var reflectFalse = reflect.ValueOf(theFalse)
var reflectTruePtr = reflect.ValueOf(theTruePtr)
var reflectFalsePtr = reflect.ValueOf(theFalsePtr)

func (bv booleanValue) ReflectTo(c px.Context, value reflect.Value) {
	if value.Kind() == reflect.Interface {
		if bv {
			value.Set(reflectTrue)
		} else {
			value.Set(reflectFalse)
		}
	} else if value.Kind() == reflect.Ptr {
		if bv {
			value.Set(reflectTruePtr)
		} else {
			value.Set(reflectFalsePtr)
		}
	} else {
		value.SetBool(bool(bv))
	}
}

func (bv booleanValue) CanSerializeAsString() bool {
	return true
}

func (bv booleanValue) SerializationString() string {
	return bv.String()
}

func (bv booleanValue) String() string {
	if bv {
		return `true`
	}
	return `false`
}

func (bv booleanValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), bv.PType())
	switch f.FormatChar() {
	case 't':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `true`, `false`), false)
	case 'T':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `True`, `False`), false)
	case 'y':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `yes`, `no`), false)
	case 'Y':
		f.ApplyStringFlags(b, bv.stringVal(f.IsAlt(), `Yes`, `No`), false)
	case 'd', 'x', 'X', 'o', 'b', 'B':
		integerValue(bv.Int()).ToString(b, px.NewFormatContext(DefaultIntegerType(), f, s.Indentation()), g)
	case 'e', 'E', 'f', 'g', 'G', 'a', 'A':
		floatValue(bv.Float()).ToString(b, px.NewFormatContext(DefaultFloatType(), f, s.Indentation()), g)
	case 's', 'p':
		f.ApplyStringFlags(b, bv.stringVal(false, `true`, `false`), false)
	default:
		panic(s.UnsupportedFormat(bv.PType(), `tTyYdxXobBeEfgGaAsp`, f))
	}
}

func (bv booleanValue) stringVal(alt bool, yes string, no string) string {
	str := no
	if bv {
		str = yes
	}
	if alt {
		str = str[:1]
	}
	return str
}

var hkTrue = px.HashKey([]byte{1, HkBoolean, 1})
var hkFalse = px.HashKey([]byte{1, HkBoolean, 0})

func (bv booleanValue) ToKey() px.HashKey {
	if bv {
		return hkTrue
	}
	return hkFalse
}

func (bv booleanValue) PType() px.Type {
	if bv {
		return &BooleanType{1}
	}
	return &BooleanType{0}
}
