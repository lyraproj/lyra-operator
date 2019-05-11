package types

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
)

var uriTypeDefault = &UriType{undef}

var URIMetaType px.ObjectType

var members = map[string]uriMemberFunc{
	`scheme`: func(uri *url.URL) px.Value {
		if uri.Scheme != `` {
			return stringValue(strings.ToLower(uri.Scheme))
		}
		return undef
	},
	`userinfo`: func(uri *url.URL) px.Value {
		if uri.User != nil {
			return stringValue(uri.User.String())
		}
		return undef
	},
	`host`: func(uri *url.URL) px.Value {
		if uri.Host != `` {
			h := uri.Host
			colon := strings.IndexByte(h, ':')
			if colon >= 0 {
				h = h[:colon]
			}
			return stringValue(strings.ToLower(h))
		}
		return undef
	},
	`port`: func(uri *url.URL) px.Value {
		port := uri.Port()
		if port != `` {
			if pn, err := strconv.Atoi(port); err == nil {
				return integerValue(int64(pn))
			}
		}
		return undef
	},
	`path`: func(uri *url.URL) px.Value {
		if uri.Path != `` {
			return stringValue(uri.Path)
		}
		return undef
	},
	`query`: func(uri *url.URL) px.Value {
		if uri.RawQuery != `` {
			return stringValue(uri.RawQuery)
		}
		return undef
	},
	`fragment`: func(uri *url.URL) px.Value {
		if uri.Fragment != `` {
			return stringValue(uri.Fragment)
		}
		return undef
	},
	`opaque`: func(uri *url.URL) px.Value {
		if uri.Opaque != `` {
			return stringValue(uri.Opaque)
		}
		return undef
	},
}

func init() {
	registerResolvableType(
		newNamedType(`Pcore::URIStringParam`, `Variant[String[1],Regexp,Type[Pattern],Type[Enum],Type[NotUndef],Type[Undef]]`).(px.ResolvableType))
	registerResolvableType(
		newNamedType(`Pcore::URIIntParam`, `Variant[Integer[0],Type[NotUndef],Type[Undef]]`).(px.ResolvableType))

	URIMetaType = newObjectType(`Pcore::URIType`,
		`Pcore::AnyType{
  attributes => {
    parameters => {
      type => Variant[Undef, String[1], URI, Struct[
        Optional['scheme'] => URIStringParam,
        Optional['userinfo'] => URIStringParam,
        Optional['host'] => URIStringParam,
        Optional['port'] => URIIntParam,
        Optional['path'] => URIStringParam,
        Optional['query'] => URIStringParam,
        Optional['fragment'] => URIStringParam,
        Optional['opaque'] => URIStringParam,
      ]],
      value => undef
    }
  }
}`, func(ctx px.Context, args []px.Value) px.Value {
			return newUriType2(args...)
		})

	newGoConstructor(`URI`,
		func(d px.Dispatch) {
			d.Param(`String[1]`)
			d.OptionalParam(`Boolean`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				strict := false
				str := args[0].String()
				if len(args) > 1 {
					strict = args[1].(booleanValue).Bool()
				}
				u, err := ParseURI2(str, strict)
				if err != nil {
					panic(px.Error(px.InvalidUri, issue.H{`str`: str, `detail`: err.Error()}))
				}
				return WrapURI(u)
			})
		},

		func(d px.Dispatch) {
			d.Param(`Hash[String[1],ScalarData]`)
			d.Function(func(c px.Context, args []px.Value) px.Value {
				return WrapURI(URIFromHash(args[0].(*Hash)))
			})
		})
}

type (
	UriType struct {
		parameters interface{} // string, URL, or hash
	}

	UriValue struct {
		UriType
	}

	uriMemberFunc func(*url.URL) px.Value

	uriMember struct {
		memberFunc uriMemberFunc
	}
)

func (um *uriMember) Call(c px.Context, receiver px.Value, block px.Lambda, args []px.Value) px.Value {
	return um.memberFunc(receiver.(*UriValue).URL())
}

func DefaultUriType() *UriType {
	return uriTypeDefault
}

func NewUriType(uri *url.URL) *UriType {
	return &UriType{uri}
}

func newUriType3(parameters *Hash) *UriType {
	if parameters.IsEmpty() {
		return uriTypeDefault
	}
	return &UriType{parameters}
}

func newUriType2(args ...px.Value) *UriType {
	switch len(args) {
	case 0:
		return DefaultUriType()
	case 1:
		if str, ok := args[0].(stringValue); ok {
			return NewUriType(ParseURI(string(str)))
		}
		if uri, ok := args[0].(*UriValue); ok {
			return NewUriType(uri.URL())
		}
		if hash, ok := args[0].(*Hash); ok {
			return newUriType3(hash)
		}
		panic(illegalArgumentType(`URI[]`, 0, `Variant[URI, Hash]`, args[0]))
	default:
		panic(illegalArgumentCount(`URI[]`, `0 or 1`, len(args)))
	}
}

// ParseURI parses a string into a uri.URL and panics with an issue code if the parse fails
func ParseURI(str string) *url.URL {
	uri, err := url.Parse(str)
	if err != nil {
		panic(px.Error(px.InvalidUri, issue.H{`str`: str, `detail`: err.Error()}))
	}
	return uri
}

func ParseURI2(str string, strict bool) (*url.URL, error) {
	if strict {
		return url.ParseRequestURI(str)
	}
	return url.Parse(str)
}

func URIFromHash(hash *Hash) *url.URL {
	uri := &url.URL{}
	if scheme, ok := hash.Get4(`scheme`); ok {
		uri.Scheme = scheme.String()
	}
	if user, ok := hash.Get4(`userinfo`); ok {
		us := user.String()
		colon := strings.IndexByte(us, ':')
		if colon >= 0 {
			uri.User = url.UserPassword(us[:colon], us[colon+1:])
		} else {
			uri.User = url.User(us)
		}
	}
	if host, ok := hash.Get4(`host`); ok {
		uri.Host = host.String()
	}
	if port, ok := hash.Get4(`port`); ok {
		uri.Host = fmt.Sprintf(`%s:%d`, uri.Host, port.(integerValue).Int())
	}
	if path, ok := hash.Get4(`path`); ok {
		uri.Path = path.String()
	}
	if query, ok := hash.Get4(`query`); ok {
		uri.RawQuery = query.String()
	}
	if fragment, ok := hash.Get4(`fragment`); ok {
		uri.Fragment = fragment.String()
	}
	if opaque, ok := hash.Get4(`opaque`); ok {
		uri.Opaque = opaque.String()
	}
	return uri
}

func (t *UriType) Accept(v px.Visitor, g px.Guard) {
	v(t)
}

func (t *UriType) Default() px.Type {
	return uriTypeDefault
}

func (t *UriType) Equals(other interface{}, g px.Guard) bool {
	if ot, ok := other.(*UriType); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return undef.Equals(ot.parameters, g)
		case *Hash:
			return t.parameters.(*Hash).Equals(ot.paramsAsHash(), g)
		default:
			if undef.Equals(ot.parameters, g) {
				return false
			}
			if u, ok := ot.parameters.(*url.URL); ok {
				return px.Equals(t.parameters, u, g)
			}
			return t.paramsAsHash().Equals(ot.paramsAsHash(), g)
		}
	}
	return false
}

func (t *UriType) Get(key string) (value px.Value, ok bool) {
	if key == `parameters` {
		switch t.parameters.(type) {
		case *UndefValue:
			return undef, true
		case *Hash:
			return t.parameters.(*Hash), true
		default:
			return urlToHash(t.parameters.(*url.URL)), true
		}
	}
	return nil, false
}

func (t *UriType) IsAssignable(other px.Type, g px.Guard) bool {
	if ot, ok := other.(*UriType); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return true
		default:
			oParams := ot.paramsAsHash()
			return t.paramsAsHash().AllPairs(func(k, b px.Value) bool {
				if a, ok := oParams.Get(k); ok {
					if at, ok := a.(px.Type); ok {
						bt, ok := b.(px.Type)
						return ok && isAssignable(bt, at)
					}
					return px.PuppetMatch(a, b)
				}
				return false
			})
		}
	}
	return false
}

func (t *UriType) IsInstance(other px.Value, g px.Guard) bool {
	if ov, ok := other.(*UriValue); ok {
		switch t.parameters.(type) {
		case *UndefValue:
			return true
		default:
			ovUri := ov.URL()
			return t.paramsAsHash().AllPairs(func(k, v px.Value) bool {
				return px.PuppetMatch(v, getURLField(ovUri, k.String()))
			})
		}
	}
	return false
}

func (t *UriType) Member(name string) (px.CallableMember, bool) {
	if member, ok := members[name]; ok {
		return &uriMember{member}, true
	}
	return nil, false
}

func (t *UriType) MetaType() px.ObjectType {
	return URIMetaType
}

func (t *UriType) Name() string {
	return `URI`
}

func (t *UriType) Parameters() []px.Value {
	switch t.parameters.(type) {
	case *UndefValue:
		return px.EmptyValues
	case *Hash:
		return []px.Value{t.parameters.(*Hash)}
	default:
		return []px.Value{urlToHash(t.parameters.(*url.URL))}
	}
}

func (t *UriType) CanSerializeAsString() bool {
	return true
}

func (t *UriType) SerializationString() string {
	return t.String()
}

func (t *UriType) String() string {
	return px.ToString2(t, None)
}

func (t *UriType) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	TypeToString(t, b, s, g)
}

func (t *UriType) PType() px.Type {
	return &TypeType{t}
}

func (t *UriType) paramsAsHash() *Hash {
	switch t.parameters.(type) {
	case *UndefValue:
		return emptyMap
	case *Hash:
		return t.parameters.(*Hash)
	default:
		return urlToHash(t.parameters.(*url.URL))
	}
}

func urlToHash(uri *url.URL) *Hash {
	entries := make([]*HashEntry, 0, 8)
	if uri.Scheme != `` {
		entries = append(entries, WrapHashEntry2(`scheme`, stringValue(strings.ToLower(uri.Scheme))))
	}
	if uri.User != nil {
		entries = append(entries, WrapHashEntry2(`userinfo`, stringValue(uri.User.String())))
	}
	if uri.Host != `` {
		h := uri.Host
		colon := strings.IndexByte(h, ':')
		if colon >= 0 {
			entries = append(entries, WrapHashEntry2(`host`, stringValue(strings.ToLower(h[:colon]))))
			if p, err := strconv.Atoi(uri.Port()); err == nil {
				entries = append(entries, WrapHashEntry2(`port`, integerValue(int64(p))))
			}
		} else {
			entries = append(entries, WrapHashEntry2(`host`, stringValue(strings.ToLower(h))))
		}
	}
	if uri.Path != `` {
		entries = append(entries, WrapHashEntry2(`path`, stringValue(uri.Path)))
	}
	if uri.RawQuery != `` {
		entries = append(entries, WrapHashEntry2(`query`, stringValue(uri.RawQuery)))
	}
	if uri.Fragment != `` {
		entries = append(entries, WrapHashEntry2(`fragment`, stringValue(uri.Fragment)))
	}
	if uri.Opaque != `` {
		entries = append(entries, WrapHashEntry2(`opaque`, stringValue(uri.Opaque)))
	}
	return WrapHash(entries)
}

func getURLField(uri *url.URL, key string) px.Value {
	if member, ok := members[key]; ok {
		return member(uri)
	}
	return undef
}

func WrapURI(uri *url.URL) *UriValue {
	return &UriValue{UriType{uri}}
}

func WrapURI2(str string) *UriValue {
	return WrapURI(ParseURI(str))
}

func (u *UriValue) Equals(other interface{}, guard px.Guard) bool {
	if ou, ok := other.(*UriValue); ok {
		return *u.URL() == *ou.URL()
	}
	return false
}

func (u *UriValue) Get(key string) (px.Value, bool) {
	if member, ok := members[key]; ok {
		return member(u.URL()), true
	}
	return undef, false
}

func (u *UriValue) CanSerializeAsString() bool {
	return true
}

func (u *UriValue) SerializationString() string {
	return u.String()
}

func (u *UriValue) String() string {
	return px.ToString(u)
}

func (u *UriValue) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), u.PType())
	val := u.URL().String()
	switch f.FormatChar() {
	case 's':
		f.ApplyStringFlags(b, val, f.IsAlt())
	case 'p':
		utils.WriteString(b, `URI(`)
		utils.PuppetQuote(b, val)
		utils.WriteByte(b, ')')
	default:
		panic(s.UnsupportedFormat(u.PType(), `sp`, f))
	}
}

func (u *UriValue) ToKey(b *bytes.Buffer) {
	b.WriteByte(1)
	b.WriteByte(HkUri)
	b.Write([]byte(u.URL().String()))
}

func (u *UriValue) PType() px.Type {
	return &u.UriType
}

func (u *UriValue) URL() *url.URL {
	return u.parameters.(*url.URL)
}
