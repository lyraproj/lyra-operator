package types

import (
	"io"

	"reflect"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type PatternType struct {
	regexps []*RegexpType
}

var PatternMetaType px.ObjectType

func init() {
	PatternMetaType = newObjectType(`Pcore::PatternType`,
		`Pcore::ScalarDataType {
	attributes => {
		patterns => Array[Regexp]
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newPatternType2(args...)
		})
}

func DefaultPatternType() *PatternType {
	return patternTypeDefault
}

func NewPatternType(regexps []*RegexpType) *PatternType {
	return &PatternType{regexps}
}

func newPatternType2(regexps ...px.Value) *PatternType {
	return newPatternType3(WrapValues(regexps))
}

func newPatternType3(regexps px.List) *PatternType {

	cnt := regexps.Len()
	switch cnt {
	case 0:
		return DefaultPatternType()
	case 1:
		if av, ok := regexps.At(0).(*Array); ok {
			return newPatternType3(av)
		}
	}

	rs := make([]*RegexpType, cnt)
	regexps.EachWithIndex(func(arg px.Value, idx int) {
		switch arg := arg.(type) {
		case *RegexpType:
			rs[idx] = arg
		case *Regexp:
			rs[idx] = arg.PType().(*RegexpType)
		case stringValue:
			rs[idx] = newRegexpType2(arg)
		default:
			panic(illegalArgumentType(`Pattern[]`, idx, `Type[Regexp], Regexp, or String`, arg))
		}
	})
	return NewPatternType(rs)
}

func (t *PatternType) Accept(v px.Visitor, g px.Guard) {
	v(t)
	for _, rx := range t.regexps {
		rx.Accept(v, g)
	}
}

func (t *PatternType) Default() px.Type {
	return patternTypeDefault
}

func (t *PatternType) Equals(o interface{}, g px.Guard) bool {
	if ot, ok := o.(*PatternType); ok {
		return len(t.regexps) == len(ot.regexps) && px.IncludesAll(t.regexps, ot.regexps, g)
	}
	return false
}

func (t *PatternType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `patterns`:
		return WrapValues(t.Parameters()), true
	}
	return nil, false
}

func (t *PatternType) IsAssignable(o px.Type, g px.Guard) bool {
	if _, ok := o.(*PatternType); ok {
		return len(t.regexps) == 0
	}

	if _, ok := o.(*stringType); ok {
		if len(t.regexps) == 0 {
			return true
		}
	}

	if vc, ok := o.(*vcStringType); ok {
		if len(t.regexps) == 0 {
			return true
		}
		str := vc.value
		return utils.MatchesString(MapToRegexps(t.regexps), str)
	}

	if et, ok := o.(*EnumType); ok {
		if len(t.regexps) == 0 {
			return true
		}
		enums := et.values
		return len(enums) > 0 && utils.MatchesAllStrings(MapToRegexps(t.regexps), enums)
	}
	return false
}

func (t *PatternType) IsInstance(o px.Value, g px.Guard) bool {
	str, ok := o.(stringValue)
	return ok && (len(t.regexps) == 0 || utils.MatchesString(MapToRegexps(t.regexps), string(str)))
}

func (t *PatternType) MetaType() px.ObjectType {
	return PatternMetaType
}

func (t *PatternType) Name() string {
	return `Pattern`
}

func (t *PatternType) Parameters() []px.Value {
	top := len(t.regexps)
	if top == 0 {
		return px.EmptyValues
	}
	rxs := make([]px.Value, top)
	for idx, rx := range t.regexps {
		rxs[idx] = WrapRegexp2(rx.pattern)
	}
	return rxs
}

func (t *PatternType) Patterns() *Array {
	rxs := make([]px.Value, len(t.regexps))
	for idx, rx := range t.regexps {
		rxs[idx] = rx
	}
	return WrapValues(rxs)
}

func (t *PatternType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(`x`), true
}

func (t *PatternType) CanSerializeAsString() bool {
	return true
}

func (t *PatternType) SerializationString() string {
	return t.String()
}

func (t *PatternType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *PatternType) String() string {
	return px.ToString2(t, None)
}

func (t *PatternType) PType() px.Type {
	return &TypeType{t}
}

var patternTypeDefault = &PatternType{[]*RegexpType{}}
