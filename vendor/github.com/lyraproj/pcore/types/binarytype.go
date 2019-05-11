package types

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"unicode/utf8"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
)

var binaryTypeDefault = &BinaryType{}

var BinaryMetaType px.ObjectType

func init() {
	BinaryMetaType = newObjectType(`Pcore::BinaryType`, `Pcore::AnyType{}`, func(ctx px.Context, args []px.Value) px.Value {
		return DefaultBinaryType()
	})

	newGoConstructor2(`Binary`,
		func(t px.LocalTypes) {
			t.Type(`ByteInteger`, `Integer[0,255]`)
			t.Type(`Encoding`, `Enum['%b', '%u', '%B', '%s', '%r']`)
			t.Type(`StringHash`, `Struct[value => String, format => Optional[Encoding]]`)
			t.Type(`ArrayHash`, `Struct[value => Array[ByteInteger]]`)
		},

		func(d px.Dispatch) {
			d.Param(`String`)
			d.OptionalParam(`Encoding`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				str := args[0].String()
				f := `%B`
				if len(args) > 1 {
					f = args[1].String()
				}
				return BinaryFromString(str, f)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Array[ByteInteger]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return BinaryFromArray(args[0].(px.List))
			})
		},

		func(d px.Dispatch) {
			d.Param(`StringHash`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				hv := args[0].(px.OrderedMap)
				return BinaryFromString(hv.Get5(`value`, px.Undef).String(), hv.Get5(`format`, px.Undef).String())
			})
		},

		func(d px.Dispatch) {
			d.Param(`ArrayHash`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return BinaryFromArray(args[0].(px.List))
			})
		},
	)
}

type (
	BinaryType struct{}

	// Binary keeps only the value because the type is known and not parameterized
	Binary struct {
		bytes []byte
	}
)

func DefaultBinaryType() *BinaryType {
	return binaryTypeDefault
}

func (t *BinaryType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *BinaryType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*BinaryType)
	return ok
}

func (t *BinaryType) IsAssignable(o px.Type, g px.Guard) bool {
	_, ok := o.(*BinaryType)
	return ok
}

func (t *BinaryType) IsInstance(o px.Value, g px.Guard) bool {
	_, ok := o.(*Binary)
	return ok
}

func (t *BinaryType) MetaType() px.ObjectType {
	return BinaryMetaType
}

func (t *BinaryType) Name() string {
	return `Binary`
}

func (t *BinaryType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf([]byte{}), true
}

func (t *BinaryType) CanSerializeAsString() bool {
	return true
}

func (t *BinaryType) SerializationString() string {
	return t.String()
}

func (t *BinaryType) String() string {
	return `Binary`
}

func (t *BinaryType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *BinaryType) PType() px.Type {
	return &TypeType{t}
}

func WrapBinary(val []byte) *Binary {
	return &Binary{val}
}

// BinaryFromFile opens file appointed by the given path for reading and returns
// its contents as a Binary. The function will panic with an issue.Reported unless
// the operation succeeds.
func BinaryFromFile(path string) *Binary {
	if bf, ok := BinaryFromFile2(path); ok {
		return bf
	}
	panic(px.Error(px.FileNotFound, issue.H{`path`: path}))
}

// BinaryFromFile2 opens file appointed by the given path for reading and returns
// its contents as a Binary together with a boolean indicating if the file was
// found or not.
//
// The function will only return false if the given file does not exist. It will panic
// with an issue.Reported on all other errors.
func BinaryFromFile2(path string) (*Binary, bool) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		stat, statErr := os.Stat(path)
		if statErr != nil {
			if os.IsNotExist(statErr) {
				return nil, false
			}
			if os.IsPermission(statErr) {
				panic(px.Error(px.FileReadDenied, issue.H{`path`: path}))
			}
		} else {
			if stat.IsDir() {
				panic(px.Error(px.IsDirectory, issue.H{`path`: path}))
			}
		}
		panic(px.Error(px.Failure, issue.H{`message`: err.Error()}))
	}
	return WrapBinary(bs), true
}

func BinaryFromString(str string, f string) *Binary {
	var bs []byte
	var err error

	switch f {
	case `%b`:
		bs, err = base64.StdEncoding.DecodeString(str)
	case `%u`:
		bs, err = base64.URLEncoding.DecodeString(str)
	case `%B`:
		bs, err = base64.StdEncoding.Strict().DecodeString(str)
	case `%s`:
		if !utf8.ValidString(str) {
			panic(illegalArgument(`BinaryFromString`, 0, `The given string is not valid utf8. Cannot create a Binary UTF-8 representation`))
		}
		bs = []byte(str)
	case `%r`:
		bs = []byte(str)
	default:
		panic(illegalArgument(`BinaryFromString`, 1, `unsupported format specifier`))
	}
	if err == nil {
		return WrapBinary(bs)
	}
	panic(illegalArgument(`BinaryFromString`, 0, err.Error()))
}

func BinaryFromArray(array px.List) *Binary {
	top := array.Len()
	result := make([]byte, top)
	for idx := 0; idx < top; idx++ {
		if v, ok := toInt(array.At(idx)); ok && 0 <= v && v <= 255 {
			result[idx] = byte(v)
			continue
		}
		panic(illegalArgument(`BinaryFromString`, 0, `The given array is not all integers between 0 and 255`))
	}
	return WrapBinary(result)
}

func (bv *Binary) AsArray() px.List {
	vs := make([]px.Value, len(bv.bytes))
	for i, b := range bv.bytes {
		vs[i] = integerValue(int64(b))
	}
	return WrapValues(vs)
}

func (bv *Binary) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*Binary); ok {
		return bytes.Equal(bv.bytes, ov.bytes)
	}
	return false
}

func (bv *Binary) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf(bv.bytes)
}

func (bv *Binary) ReflectTo(c px.Context, value reflect.Value) {
	switch value.Type().Elem().Kind() {
	case reflect.Int8, reflect.Uint8:
		value.SetBytes(bv.bytes)
	case reflect.Interface:
		value.Set(reflect.ValueOf(bv.bytes))
	default:
		panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: `[]byte`, `actual`: fmt.Sprintf(`[]%s`, value.Kind())}))
	}
}

func (bv *Binary) CanSerializeAsString() bool {
	return true
}

func (bv *Binary) SerializationString() string {
	return base64.StdEncoding.Strict().EncodeToString(bv.bytes)
}

func (bv *Binary) String() string {
	return px.ToString2(bv, None)
}

func (bv *Binary) ToKey(b *bytes.Buffer) {
	b.WriteByte(0)
	b.WriteByte(HkBinary)
	b.Write(bv.bytes)
}

func (bv *Binary) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), bv.PType())
	var str string
	switch f.FormatChar() {
	case 's':
		if !utf8.Valid(bv.bytes) {
			panic(px.Error(px.Failure, issue.H{`message`: `binary data is not valid UTF-8`}))
		}
		str = string(bv.bytes)
	case 'p':
		str = `Binary('` + base64.StdEncoding.EncodeToString(bv.bytes) + `')`
	case 'b':
		str = base64.StdEncoding.EncodeToString(bv.bytes) + "\n"
	case 'B':
		str = base64.StdEncoding.Strict().EncodeToString(bv.bytes)
	case 'u':
		str = base64.URLEncoding.EncodeToString(bv.bytes)
	case 't':
		str = `Binary`
	case 'T':
		str = `BINARY`
	default:
		panic(s.UnsupportedFormat(bv.PType(), `bButTsp`, f))
	}
	f.ApplyStringFlags(b, str, f.IsAlt())
}

func (bv *Binary) PType() px.Type {
	return DefaultBinaryType()
}

func (bv *Binary) Bytes() []byte {
	return bv.bytes
}
