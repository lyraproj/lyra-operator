package types

import (
	"io"

	"fmt"
	"reflect"
	"strconv"

	"github.com/lyraproj/pcore/px"
)

type NumericType struct{}

var numericTypeDefault = &NumericType{}

var NumericMetaType px.ObjectType

func init() {
	NumericMetaType = newObjectType(`Pcore::NumericType`, `Pcore::ScalarDataType {}`, func(ctx px.Context, args []px.Value) px.Value {
		return DefaultNumericType()
	})

	newGoConstructor2(`Numeric`,
		func(t px.LocalTypes) {
			t.Type(`Convertible`, `Variant[Numeric, Boolean, Pattern[/`+FloatPattern+`/], Timespan, Timestamp]`)
			t.Type(`NamedArgs`, `Struct[from => Convertible, Optional[abs] => Boolean]`)
		},

		func(d px.Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return numberFromNamedArgs(args, true)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Convertible`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return numberFromPositionalArgs(args, true)
			})
		},
	)
}

func numberFromPositionalArgs(args []px.Value, tryInt bool) px.Number {
	n := fromConvertible(args[0], tryInt)
	if len(args) > 1 && args[1].(booleanValue).Bool() {
		if i, ok := n.(integerValue); ok {
			n = integerValue(i.Abs())
		} else {
			n = floatValue(n.(floatValue).Abs())
		}
	}
	return n
}

func numberFromNamedArgs(args []px.Value, tryInt bool) px.Number {
	h := args[0].(*Hash)
	n := fromConvertible(h.Get5(`from`, px.Undef), tryInt)
	a := h.Get5(`abs`, nil)
	if a != nil && a.(booleanValue).Bool() {
		if i, ok := n.(integerValue); ok {
			n = integerValue(i.Abs())
		} else {
			n = floatValue(n.(floatValue).Abs())
		}
	}
	return n
}

func DefaultNumericType() *NumericType {
	return numericTypeDefault
}

func (t *NumericType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *NumericType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*NumericType)
	return ok
}

func (t *NumericType) IsAssignable(o px.Type, g px.Guard) bool {
	switch o.(type) {
	case *IntegerType, *FloatType:
		return true
	default:
		return false
	}
}

func (t *NumericType) IsInstance(o px.Value, g px.Guard) bool {
	switch o.PType().(type) {
	case *FloatType, *IntegerType:
		return true
	default:
		return false
	}
}

func (t *NumericType) MetaType() px.ObjectType {
	return NumericMetaType
}

func (t *NumericType) Name() string {
	return `Numeric`
}

func (t *NumericType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(float64(0.0)), true
}

func (t *NumericType) CanSerializeAsString() bool {
	return true
}

func (t *NumericType) SerializationString() string {
	return t.String()
}

func (t *NumericType) String() string {
	return px.ToString2(t, None)
}

func (t *NumericType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *NumericType) PType() px.Type {
	return &TypeType{t}
}

func fromConvertible(c px.Value, allowInt bool) px.Number {
	switch c := c.(type) {
	case integerValue:
		if allowInt {
			return c
		}
		return floatValue(c.Float())
	case *Timestamp:
		return floatValue(c.Float())
	case Timespan:
		return floatValue(c.Float())
	case booleanValue:
		if allowInt {
			return integerValue(c.Int())
		}
		return floatValue(c.Float())
	case px.Number:
		return c
	case stringValue:
		s := c.String()
		if allowInt {
			if i, err := strconv.ParseInt(s, 0, 64); err == nil {
				return integerValue(i)
			}
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return floatValue(f)
		}
		if allowInt {
			if len(s) > 2 && s[0] == '0' && (s[1] == 'b' || s[1] == 'B') {
				if i, err := strconv.ParseInt(s[2:], 2, 64); err == nil {
					return integerValue(i)
				}
			}
		}
	}
	panic(illegalArguments(`Numeric`, fmt.Sprintf(`Value of type %s cannot be converted to an Number`, c.PType().String())))
}
