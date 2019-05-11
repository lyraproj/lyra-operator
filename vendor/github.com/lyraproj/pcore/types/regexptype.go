package types

import (
	"bytes"
	"io"
	"reflect"
	"regexp"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

type (
	RegexpType struct {
		pattern *regexp.Regexp
	}

	// Regexp represents RegexpType as a value
	Regexp RegexpType
)

var regexpTypeDefault = &RegexpType{pattern: regexp.MustCompile(``)}

var RegexpMetaType px.ObjectType

func init() {
	RegexpMetaType = newObjectType(`Pcore::RegexpType`,
		`Pcore::ScalarType {
	attributes => {
		pattern => {
      type => Variant[Undef,String,Regexp],
      value => undef
    }
	}
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newRegexpType2(args...)
		})

	newGoConstructor(`Regexp`,
		func(d px.Dispatch) {
			d.Param(`Variant[String,Regexp]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				arg := args[0]
				if s, ok := arg.(*Regexp); ok {
					return s
				}
				return WrapRegexp(arg.String())
			})
		},
	)
}

func DefaultRegexpType() *RegexpType {
	return regexpTypeDefault
}

func NewRegexpType(patternString string) *RegexpType {
	if patternString == `` {
		return DefaultRegexpType()
	}
	pattern, err := regexp.Compile(patternString)
	if err != nil {
		panic(px.Error(px.InvalidRegexp, issue.H{`pattern`: patternString, `detail`: err.Error()}))
	}
	return &RegexpType{pattern: pattern}
}

func NewRegexpTypeR(pattern *regexp.Regexp) *RegexpType {
	if pattern.String() == `` {
		return DefaultRegexpType()
	}
	return &RegexpType{pattern: pattern}
}

func newRegexpType2(args ...px.Value) *RegexpType {
	switch len(args) {
	case 0:
		return regexpTypeDefault
	case 1:
		rx := args[0]
		if str, ok := rx.(stringValue); ok {
			return NewRegexpType(string(str))
		}
		if rt, ok := rx.(*Regexp); ok {
			return rt.PType().(*RegexpType)
		}
		panic(illegalArgumentType(`Regexp[]`, 0, `Variant[Regexp,String]`, args[0]))
	default:
		panic(illegalArgumentCount(`Regexp[]`, `0 - 1`, len(args)))
	}
}

func (t *RegexpType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *RegexpType) Default() px.Type {
	return regexpTypeDefault
}

func (t *RegexpType) Equals(o interface{}, g px.Guard) bool {
	ot, ok := o.(*RegexpType)
	return ok && t.pattern.String() == ot.pattern.String()
}

func (t *RegexpType) Get(key string) (value px.Value, ok bool) {
	switch key {
	case `pattern`:
		if t.String() == `` {
			return undef, true
		}
		return stringValue(t.pattern.String()), true
	}
	return nil, false
}

func (t *RegexpType) IsAssignable(o px.Type, g px.Guard) bool {
	rx, ok := o.(*RegexpType)
	return ok && (t.pattern.String() == `` || t.pattern.String() == rx.PatternString())
}

func (t *RegexpType) IsInstance(o px.Value, g px.Guard) bool {
	rx, ok := o.(*Regexp)
	return ok && (t.pattern.String() == `` || t.pattern.String() == rx.PatternString())
}

func (t *RegexpType) MetaType() px.ObjectType {
	return RegexpMetaType
}

func (t *RegexpType) Name() string {
	return `Regexp`
}

func (t *RegexpType) Parameters() []px.Value {
	if t.pattern.String() == `` {
		return px.EmptyValues
	}
	return []px.Value{WrapRegexp2(t.pattern)}
}

func (t *RegexpType) ReflectType(c px.Context) (reflect.Type, bool) {
	return reflect.TypeOf(regexpTypeDefault.pattern), true
}

func (t *RegexpType) PatternString() string {
	return t.pattern.String()
}

func (t *RegexpType) Regexp() *regexp.Regexp {
	return t.pattern
}

func (t *RegexpType) CanSerializeAsString() bool {
	return true
}

func (t *RegexpType) SerializationString() string {
	return t.String()
}

func (t *RegexpType) String() string {
	return px.ToString2(t, None)
}

func (t *RegexpType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *RegexpType) PType() px.Type {
	return &TypeType{t}
}

func MapToRegexps(regexpTypes []*RegexpType) []*regexp.Regexp {
	top := len(regexpTypes)
	result := make([]*regexp.Regexp, top)
	for idx := 0; idx < top; idx++ {
		result[idx] = regexpTypes[idx].Regexp()
	}
	return result
}

func UniqueRegexps(regexpTypes []*RegexpType) []*RegexpType {
	top := len(regexpTypes)
	if top < 2 {
		return regexpTypes
	}

	result := make([]*RegexpType, 0, top)
	exists := make(map[string]bool, top)
	for _, regexpType := range regexpTypes {
		key := regexpType.String()
		if !exists[key] {
			exists[key] = true
			result = append(result, regexpType)
		}
	}
	return result
}

func WrapRegexp(str string) *Regexp {
	pattern, err := regexp.Compile(str)
	if err != nil {
		panic(px.Error(px.InvalidRegexp, issue.H{`pattern`: str, `detail`: err.Error()}))
	}
	return &Regexp{pattern}
}

func WrapRegexp2(pattern *regexp.Regexp) *Regexp {
	return &Regexp{pattern}
}

func (r *Regexp) Equals(o interface{}, g px.Guard) bool {
	if ov, ok := o.(*Regexp); ok {
		return r.pattern.String() == ov.pattern.String()
	}
	return false
}

func (r *Regexp) Match(s string) []string {
	return r.pattern.FindStringSubmatch(s)
}

func (r *Regexp) Regexp() *regexp.Regexp {
	return r.pattern
}

func (r *Regexp) PatternString() string {
	return r.pattern.String()
}

func (r *Regexp) Reflect(c px.Context) reflect.Value {
	return reflect.ValueOf(r.pattern)
}

func (r *Regexp) ReflectTo(c px.Context, dest reflect.Value) {
	rv := r.Reflect(c)
	if rv.Kind() == reflect.Ptr && dest.Kind() != reflect.Ptr {
		rv = rv.Elem()
	}
	if !rv.Type().AssignableTo(dest.Type()) {
		panic(px.Error(px.AttemptToSetWrongKind, issue.H{`expected`: rv.Type().String(), `actual`: dest.Type().String()}))
	}
	dest.Set(rv)
}

func (r *Regexp) String() string {
	return px.ToString2(r, None)
}

func (r *Regexp) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HkRegexp)
	b.Write([]byte(r.pattern.String()))
}

func (r *Regexp) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	utils.RegexpQuote(b, r.pattern.String())
}

func (r *Regexp) PType() px.Type {
	rt := RegexpType(*r)
	return &rt
}
