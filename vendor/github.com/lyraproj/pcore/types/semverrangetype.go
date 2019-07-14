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
	SemVerRangeType struct{}

	SemVerRange struct {
		rng semver.VersionRange
	}
)

var semVerRangeTypeDefault = &SemVerRangeType{}

var SemVerRangeMetaType px.ObjectType

func init() {
	SemVerRangeMetaType = newObjectType(`Pcore::SemVerRangeType`, `Pcore::AnyType {}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return DefaultSemVerRangeType()
		})

	newGoConstructor2(`SemVerRange`,
		func(t px.LocalTypes) {
			t.Type(`SemVerRangeString`, `String[1]`)
			t.Type(`SemVerRangeHash`, `Struct[min=>Variant[Default,SemVer],Optional[max]=>Variant[Default,SemVer],Optional[exclude_max]=>Boolean]`)
		},

		func(d px.Dispatch) {
			d.Param(`SemVerRangeString`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				v, err := semver.ParseVersionRange(args[0].String())
				if err != nil {
					panic(illegalArgument(`SemVerRange`, 0, err.Error()))
				}
				return WrapSemVerRange(v)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Variant[Default,SemVer]`)
			d.Param(`Variant[Default,SemVer]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				var start semver.Version
				if _, ok := args[0].(*DefaultValue); ok {
					start = semver.Min
				} else {
					start = args[0].(*SemVer).Version()
				}
				var end semver.Version
				if _, ok := args[1].(*DefaultValue); ok {
					end = semver.Max
				} else {
					end = args[1].(*SemVer).Version()
				}
				excludeEnd := false
				if len(args) > 2 {
					excludeEnd = args[2].(booleanValue).Bool()
				}
				return WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},

		func(d px.Dispatch) {
			d.Param(`SemVerRangeHash`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				hash := args[0].(*Hash)
				start := hash.Get5(`min`, nil).(*SemVer).Version()

				var end semver.Version
				ev := hash.Get5(`max`, nil)
				if ev == nil {
					end = semver.Max
				} else {
					end = ev.(*SemVer).Version()
				}

				excludeEnd := false
				ev = hash.Get5(`excludeMax`, nil)
				if ev != nil {
					excludeEnd = ev.(booleanValue).Bool()
				}
				return WrapSemVerRange(semver.FromVersions(start, false, end, excludeEnd))
			})
		},
	)
}

func DefaultSemVerRangeType() *SemVerRangeType {
	return semVerRangeTypeDefault
}

func (t *SemVerRangeType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *SemVerRangeType) Equals(o interface{}, g px.Guard) bool {
	_, ok := o.(*SemVerRangeType)
	return ok
}

func (t *SemVerRangeType) MetaType() px.ObjectType {
	return SemVerRangeMetaType
}

func (t *SemVerRangeType) Name() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) CanSerializeAsString() bool {
	return true
}

func (t *SemVerRangeType) SerializationString() string {
	return t.String()
}

func (t *SemVerRangeType) String() string {
	return `SemVerRange`
}

func (t *SemVerRangeType) IsAssignable(o px.Type, g px.Guard) bool {
	_, ok := o.(*SemVerRangeType)
	return ok
}

func (t *SemVerRangeType) IsInstance(o px.Value, g px.Guard) bool {
	_, ok := o.(*SemVerRange)
	return ok
}

func (t *SemVerRangeType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(semver.MatchAll), true
}

func (t *SemVerRangeType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *SemVerRangeType) PType() px.Type {
	return &TypeType{t}
}

func WrapSemVerRange(val semver.VersionRange) *SemVerRange {
	return &SemVerRange{val}
}

func (bv *SemVerRange) VersionRange() semver.VersionRange {
	return bv.rng
}

func (bv *SemVerRange) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*SemVerRange); ok {
		return bv.rng.Equals(ov.rng)
	}
	return false
}

func (bv *SemVerRange) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf(bv.rng)
}

func (bv *SemVerRange) ReflectTo(c px.Context, dest reflect.Value) {
	rv := bv.Reflect(c)
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

func (bv *SemVerRange) CanSerializeAsString() bool {
	return true
}

func (bv *SemVerRange) SerializationString() string {
	return bv.String()
}

func (bv *SemVerRange) String() string {
	return px.ToString2(bv, None)
}

func (bv *SemVerRange) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), bv.PType())
	vr := bv.rng
	switch f.FormatChar() {
	case 'p':
		utils.WriteString(b, `SemVerRange(`)
		if f.IsAlt() {
			utils.PuppetQuote(b, vr.NormalizedString())
		} else {
			utils.PuppetQuote(b, vr.String())
		}
		utils.WriteString(b, `)`)
	case 's':
		if f.IsAlt() {
			vr.ToNormalizedString(b)
		} else {
			vr.ToString(b)
		}
	default:
		panic(s.UnsupportedFormat(bv.PType(), `ps`, f))
	}
}

func (bv *SemVerRange) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HkVersionRange)
	bv.rng.ToString(b)
}

func (bv *SemVerRange) PType() px.Type {
	return DefaultSemVerRangeType()
}
