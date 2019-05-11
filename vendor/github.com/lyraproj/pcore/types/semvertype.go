package types

import (
	"bytes"
	"io"

	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/semver/semver"
)

type (
	SemVerType struct {
		vRange semver.VersionRange
	}

	SemVer SemVerType
)

var semVerTypeDefault = &SemVerType{semver.MatchAll}

var SemVerMetaType px.ObjectType

func init() {
	SemVerMetaType = newObjectType(`Pcore::SemVerType`, `Pcore::ScalarType {
	attributes => {
		ranges => {
      type => Array[Variant[SemVerRange,String[1]]],
      value => []
    }
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
		return newSemVerType2(args...)
	})

	newGoConstructor2(`SemVer`,
		func(t px.LocalTypes) {
			t.Type(`PositiveInteger`, `Integer[0,default]`)
			t.Type(`SemVerQualifier`, `Pattern[/\A[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*\z/]`)
			t.Type(`SemVerString`, `String[1]`)
			t.Type(`SemVerHash`, `Struct[major=>PositiveInteger,minor=>PositiveInteger,patch=>PositiveInteger,Optional[prerelease]=>SemVerQualifier,Optional[build]=>SemVerQualifier]`)
		},

		func(d px.Dispatch) {
			d.Param(`SemVerString`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				v, err := semver.ParseVersion(args[0].String())
				if err != nil {
					panic(illegalArgument(`SemVer`, 0, err.Error()))
				}
				return WrapSemVer(v)
			})
		},

		func(d px.Dispatch) {
			d.Param(`PositiveInteger`)
			d.Param(`PositiveInteger`)
			d.Param(`PositiveInteger`)
			d.OptionalParam(`SemVerQualifier`)
			d.OptionalParam(`SemVerQualifier`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				argc := len(args)
				major := args[0].(integerValue).Int()
				minor := args[1].(integerValue).Int()
				patch := args[2].(integerValue).Int()
				preRelease := ``
				build := ``
				if argc > 3 {
					preRelease = args[3].String()
					if argc > 4 {
						build = args[4].String()
					}
				}
				v, err := semver.NewVersion3(int(major), int(minor), int(patch), preRelease, build)
				if err != nil {
					panic(illegalArguments(`SemVer`, err.Error()))
				}
				return WrapSemVer(v)
			})
		},

		func(d px.Dispatch) {
			d.Param(`SemVerHash`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				hash := args[0].(*Hash)
				major := hash.Get5(`major`, ZERO).(integerValue).Int()
				minor := hash.Get5(`minor`, ZERO).(integerValue).Int()
				patch := hash.Get5(`patch`, ZERO).(integerValue).Int()
				preRelease := ``
				build := ``
				ev := hash.Get5(`prerelease`, nil)
				if ev != nil {
					preRelease = ev.String()
				}
				ev = hash.Get5(`build`, nil)
				if ev != nil {
					build = ev.String()
				}
				v, err := semver.NewVersion3(int(major), int(minor), int(patch), preRelease, build)
				if err != nil {
					panic(illegalArguments(`SemVer`, err.Error()))
				}
				return WrapSemVer(v)
			})
		},
	)
}

func DefaultSemVerType() *SemVerType {
	return semVerTypeDefault
}

func NewSemVerType(vr semver.VersionRange) *SemVerType {
	if vr.Equals(semver.MatchAll) {
		return DefaultSemVerType()
	}
	return &SemVerType{vr}
}

func newSemVerType2(limits ...px.Value) *SemVerType {
	return newSemVerType3(WrapValues(limits))
}

func newSemVerType3(limits px.List) *SemVerType {
	argc := limits.Len()
	if argc == 0 {
		return DefaultSemVerType()
	}

	if argc == 1 {
		if ranges, ok := limits.At(0).(px.List); ok {
			return newSemVerType3(ranges)
		}
	}

	var finalRange semver.VersionRange
	limits.EachWithIndex(func(arg px.Value, idx int) {
		var rng semver.VersionRange
		str, ok := arg.(stringValue)
		if ok {
			var err error
			rng, err = semver.ParseVersionRange(string(str))
			if err != nil {
				panic(illegalArgument(`SemVer[]`, idx, err.Error()))
			}
		} else {
			rv, ok := arg.(*SemVerRange)
			if !ok {
				panic(illegalArgumentType(`SemVer[]`, idx, `Variant[String,SemVerRange]`, arg))
			}
			rng = rv.VersionRange()
		}
		if finalRange == nil {
			finalRange = rng
		} else {
			finalRange = finalRange.Merge(rng)
		}
	})
	return NewSemVerType(finalRange)
}

func (t *SemVerType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *SemVerType) Default() px.Type {
	return semVerTypeDefault
}

func (t *SemVerType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*SemVerType)
	return ok
}

func (t *SemVerType) Get(key string) (px.Value, bool) {
	switch key {
	case `ranges`:
		return WrapValues(t.Parameters()), true
	default:
		return nil, false
	}
}

func (t *SemVerType) MetaType() px.ObjectType {
	return SemVerMetaType
}

func (t *SemVerType) Name() string {
	return `SemVer`
}

func (t *SemVerType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(semver.Max), true
}

func (t *SemVerType) CanSerializeAsString() bool {
	return true
}

func (t *SemVerType) SerializationString() string {
	return t.String()
}

func (t *SemVerType) String() string {
	return px.ToString2(t, None)
}

func (t *SemVerType) IsAssignable(o px.Type, g px.Guard) bool {
	if vt, ok := o.(*SemVerType); ok {
		return vt.vRange.IsAsRestrictiveAs(t.vRange)
	}
	return false
}

func (t *SemVerType) IsInstance(o px.Value, g px.Guard) bool {
	if v, ok := o.(*SemVer); ok {
		return t.vRange.Includes(v.Version())
	}
	return false
}

func (t *SemVerType) Parameters() []px.Value {
	if t.vRange.Equals(semver.MatchAll) {
		return px.EmptyValues
	}
	return []px.Value{stringValue(t.vRange.String())}
}

func (t *SemVerType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *SemVerType) PType() px.Type {
	return &TypeType{t}
}

func WrapSemVer(val semver.Version) *SemVer {
	return (*SemVer)(NewSemVerType(semver.ExactVersionRange(val)))
}

func (v *SemVer) Version() semver.Version {
	return v.vRange.StartVersion()
}

func (v *SemVer) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*SemVer); ok {
		return v.Version().Equals(ov.Version())
	}
	return false
}

func (v *SemVer) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf(v.Version())
}

func (v *SemVer) ReflectTo(c px.Context, dest reflect.Value) {
	rv := v.Reflect(c)
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

func (v *SemVer) CanSerializeAsString() bool {
	return true
}

func (v *SemVer) SerializationString() string {
	return v.String()
}

func (v *SemVer) String() string {
	if v.vRange == nil || v.vRange.StartVersion() == nil {
		return `0.0.0-`
	}
	return v.Version().String()
}

func (v *SemVer) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HkVersion)
	v.Version().ToString(b)
}

func (v *SemVer) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), v.PType())
	val := v.Version().String()
	switch f.FormatChar() {
	case 's':
		f.ApplyStringFlags(b, val, f.IsAlt())
	case 'p':
		utils.WriteString(b, `SemVer(`)
		utils.PuppetQuote(b, val)
		utils.WriteByte(b, ')')
	default:
		panic(s.UnsupportedFormat(v.PType(), `sp`, f))
	}
}

func (v *SemVer) PType() px.Type {
	return (*SemVerType)(v)
}
