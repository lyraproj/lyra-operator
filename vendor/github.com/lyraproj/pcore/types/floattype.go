package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

type (
	FloatType struct {
		min float64
		max float64
	}

	// floatValue represents float64 as a pcore.Value
	floatValue float64
)

var floatTypeDefault = &FloatType{-math.MaxFloat64, math.MaxFloat64}
var floatType32 = &FloatType{-math.MaxFloat32, math.MaxFloat32}

var FloatMetaType px.ObjectType

func init() {
	FloatMetaType = newObjectType(`Pcore::FloatType`,
		`Pcore::NumericType {
  attributes => {
    from => { type => Optional[Float], value => undef },
    to => { type => Optional[Float], value => undef }
  }
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newFloatType2(args...)
		})

	newGoConstructor2(`Float`,
		func(t px.LocalTypes) {
			t.Type(`Convertible`, `Variant[Numeric, Boolean, Pattern[/`+FloatPattern+`/], Timespan, Timestamp]`)
			t.Type(`NamedArgs`, `Struct[{from => Convertible, Optional[abs] => Boolean}]`)
		},

		func(d px.Dispatch) {
			d.Param(`Convertible`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return numberFromPositionalArgs(args, false)
			})
		},

		func(d px.Dispatch) {
			d.Param(`NamedArgs`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return numberFromNamedArgs(args, false)
			})
		},
	)
}

func DefaultFloatType() *FloatType {
	return floatTypeDefault
}

func NewFloatType(min float64, max float64) *FloatType {
	if min == -math.MaxFloat64 && max == math.MaxFloat64 {
		return DefaultFloatType()
	}
	if min > max {
		panic(illegalArguments(`Float[]`, `min is not allowed to be greater than max`))
	}
	return &FloatType{min, max}
}

func newFloatType2(limits ...px.Value) *FloatType {
	argc := len(limits)
	if argc == 0 {
		return floatTypeDefault
	}
	min, ok := toFloat(limits[0])
	if !ok {
		if _, ok = limits[0].(*DefaultValue); !ok {
			panic(illegalArgumentType(`Float[]`, 0, `Float`, limits[0]))
		}
		min = -math.MaxFloat64
	}

	var max float64
	switch argc {
	case 1:
		max = math.MaxFloat64
	case 2:
		if max, ok = toFloat(limits[1]); !ok {
			if _, ok = limits[1].(*DefaultValue); !ok {
				panic(illegalArgumentType(`Float[]`, 1, `Float`, limits[1]))
			}
			max = math.MaxFloat64
		}
	default:
		panic(illegalArgumentCount(`Float`, `0 - 2`, len(limits)))
	}
	return NewFloatType(min, max)
}

func (t *FloatType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *FloatType) Default() px.Type {
	return floatTypeDefault
}

func (t *FloatType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*FloatType); ok {
		return t.min == ot.min && t.max == ot.max
	}
	return false
}

func (t *FloatType) Generic() px.Type {
	return floatTypeDefault
}

func (t *FloatType) Get(key string) (px.Value, bool) {
	switch key {
	case `from`:
		v := px.Undef
		if t.min != -math.MaxFloat64 {
			v = floatValue(t.min)
		}
		return v, true
	case `to`:
		v := px.Undef
		if t.max != math.MaxFloat64 {
			v = floatValue(t.max)
		}
		return v, true
	default:
		return nil, false
	}
}

func (t *FloatType) IsAssignable(o px.Type, g px.Guard) bool {
	if ft, ok := o.(*FloatType); ok {
		return t.min <= ft.min && t.max >= ft.max
	}
	return false
}

func (t *FloatType) IsInstance(o px.Value, g px.Guard) bool {
	if n, ok := toFloat(o); ok {
		return t.min <= n && n <= t.max
	}
	return false
}

func (t *FloatType) MetaType() px.ObjectType {
	return FloatMetaType
}

func (t *FloatType) Min() float64 {
	return t.min
}

func (t *FloatType) Max() float64 {
	return t.max
}

func (t *FloatType) Name() string {
	return `Float`
}

func (t *FloatType) Parameters() []px.Value {
	if t.min == -math.MaxFloat64 {
		if t.max == math.MaxFloat64 {
			return px.EmptyValues
		}
		return []px.Value{WrapDefault(), floatValue(t.max)}
	}
	if t.max == math.MaxFloat64 {
		return []px.Value{floatValue(t.min)}
	}
	return []px.Value{floatValue(t.min), floatValue(t.max)}
}

func (t *FloatType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(float64(0.0)), true
}

func (t *FloatType) CanSerializeAsString() bool {
	return true
}

func (t *FloatType) SerializationString() string {
	return t.String()
}

func (t *FloatType) String() string {
	return px.ToString2(t, None)
}

func (t *FloatType) IsUnbounded() bool {
	return t.min == -math.MaxFloat64 && t.max == math.MaxFloat64
}

func (t *FloatType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *FloatType) PType() px.Type {
	return &TypeType{t}
}

func WrapFloat(val float64) px.Float {
	return floatValue(val)
}

func (fv floatValue) Abs() float64 {
	f := float64(fv)
	if f < 0 {
		return -f
	}
	return f
}

func (fv floatValue) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(floatValue); ok {
		return fv == ov
	}
	return false
}

func (fv floatValue) Float() float64 {
	return float64(fv)
}

func (fv floatValue) Int() int64 {
	return int64(fv.Float())
}

func (fv floatValue) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf(fv.Float())
}

func (fv floatValue) ReflectTo(c px.Context, value reflect.Value) {
	switch value.Kind() {
	case reflect.Float64, reflect.Float32:
		value.SetFloat(float64(fv))
		return
	case reflect.Interface:
		value.Set(reflect.ValueOf(float64(fv)))
		return
	case reflect.Ptr:
		switch value.Type().Elem().Kind() {
		case reflect.Float64:
			f := float64(fv)
			value.Set(reflect.ValueOf(&f))
			return
		case reflect.Float32:
			f32 := float32(fv)
			value.Set(reflect.ValueOf(&f32))
			return
		}
	}
	panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: `Float`, `actual`: value.Kind().String()}))
}

func (fv floatValue) String() string {
	return fmt.Sprintf(`%v`, fv.Float())
}

func (fv floatValue) ToKey(b *bytes.Buffer) {
	n := math.Float64bits(float64(fv))
	b.WriteByte(1)
	b.WriteByte(HkFloat)
	b.WriteByte(byte(n >> 56))
	b.WriteByte(byte(n >> 48))
	b.WriteByte(byte(n >> 40))
	b.WriteByte(byte(n >> 32))
	b.WriteByte(byte(n >> 24))
	b.WriteByte(byte(n >> 16))
	b.WriteByte(byte(n >> 8))
	b.WriteByte(byte(n))
}

var defaultFormatP = newFormat(`%g`)
var defaultFormatS = newFormat(`%#g`)

func (fv floatValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), fv.PType())
	switch f.FormatChar() {
	case 'd', 'x', 'X', 'o', 'b', 'B':
		integerValue(fv.Int()).ToString(b, px.NewFormatContext(DefaultIntegerType(), f, s.Indentation()), g)
	case 'p':
		f.ApplyStringFlags(b, floatGFormat(defaultFormatP, float64(fv)), false)
	case 'e', 'E', 'f':
		_, err := fmt.Fprintf(b, f.OrigFormat(), float64(fv))
		if err != nil {
			panic(err)
		}
	case 'g', 'G':
		_, err := io.WriteString(b, floatGFormat(f, float64(fv)))
		if err != nil {
			panic(err)
		}
	case 's':
		f.ApplyStringFlags(b, floatGFormat(defaultFormatS, float64(fv)), f.IsAlt())
	case 'a', 'A':
		// TODO: Implement this or list as limitation?
		panic(s.UnsupportedFormat(fv.PType(), `dxXobBeEfgGaAsp`, f))
	default:
		panic(s.UnsupportedFormat(fv.PType(), `dxXobBeEfgGaAsp`, f))
	}
}

func floatGFormat(f px.Format, value float64) string {
	str := fmt.Sprintf(f.WithoutWidth().OrigFormat(), value)
	sc := byte('e')
	if f.FormatChar() == 'G' {
		sc = 'E'
	}
	if strings.IndexByte(str, sc) >= 0 {
		// Scientific notation in use.
		return str
	}

	// Go might strip both trailing zeroes and decimal point when using '%g'. The
	// decimal point and trailing zeroes are restored here
	totLen := len(str)
	prc := f.Precision()
	if prc < 0 && !f.IsAlt() {
		prc = 6
	}

	dotIndex := strings.IndexByte(str, '.')
	missing := 0
	if prc >= 0 {
		if dotIndex >= 0 {
			missing = prc - (totLen - 1)
		} else {
			missing = prc - totLen
			if missing == 0 {
				// Impossible to add a fraction part. Force scientific notation
				return fmt.Sprintf(f.ReplaceFormatChar(sc).OrigFormat(), value)
			}
		}
	}

	b := bytes.NewBufferString(``)

	padByte := byte(' ')
	if f.IsZeroPad() {
		padByte = '0'
	}
	pad := 0
	if f.Width() > 0 {
		pad = f.Width() - (totLen + missing + 1)
	}

	if !f.IsLeft() {
		for ; pad > 0; pad-- {
			b.WriteByte(padByte)
		}
	}

	b.WriteString(str)
	if dotIndex < 0 {
		b.WriteByte('.')
		if missing == 0 {
			b.WriteByte('0')
		}
	}
	for missing > 0 {
		b.WriteByte('0')
		missing--
	}

	if f.IsLeft() {
		for ; pad > 0; pad-- {
			b.WriteByte(padByte)
		}
	}
	return b.String()
}

func (fv floatValue) PType() px.Type {
	f := float64(fv)
	return &FloatType{f, f}
}
