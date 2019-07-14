package types

import (
	"bytes"
	"io"
	"reflect"

	"github.com/lyraproj/pcore/px"
)

type taggedType struct {
	typ              reflect.Type
	puppetTags       map[string]string
	annotations      px.OrderedMap
	tags             map[string]map[string]string
	parsedPuppetTags map[string]px.OrderedMap
}

var TagsAnnotationType px.ObjectType

func init() {
	px.NewTaggedType = func(typ reflect.Type, puppetTags map[string]string) px.AnnotatedType {
		tt := &taggedType{typ: typ, puppetTags: puppetTags, annotations: emptyMap}
		tt.initTags()
		return tt
	}

	px.NewAnnotatedType = func(typ reflect.Type, puppetTags map[string]string, annotations px.OrderedMap) px.AnnotatedType {
		tt := &taggedType{typ: typ, puppetTags: puppetTags, annotations: annotations}
		tt.initTags()
		return tt
	}

	TagsAnnotationType = newGoObjectType(`TagsAnnotation`, reflect.TypeOf((*px.TagsAnnotation)(nil)).Elem(), `Annotation{
    attributes => {
			# Arbitrary data used by custom implementations
      tags => Hash[String,String]
    }
	}`,
		func(ctx px.Context, args []px.Value) px.Value {
			return NewTagsAnnotation(args[0].(px.OrderedMap))
		},
		func(ctx px.Context, args []px.Value) px.Value {
			h := args[0].(*Hash)
			return NewTagsAnnotation(h.Get5(`tags`, px.EmptyMap).(px.OrderedMap))
		},
	)
}

type tagsAnnotation struct {
	tags px.OrderedMap
}

func NewTagsAnnotation(tags px.OrderedMap) px.TagsAnnotation {
	return &tagsAnnotation{tags}
}

func (c *tagsAnnotation) Equals(value interface{}, guard px.Guard) bool {
	if oc, ok := value.(*tagsAnnotation); ok {
		return c.tags.Equals(oc.tags, guard)
	}
	return false
}

func (c *tagsAnnotation) PType() px.Type {
	return TagsAnnotationType
}

func (c *tagsAnnotation) String() string {
	return px.ToString(c)
}

func (c *tagsAnnotation) ToString(bld io.Writer, format px.FormatContext, g px.RDetect) {
	ObjectToString(c, format, bld, g)
}

func (c *tagsAnnotation) Get(key string) (value px.Value, ok bool) {
	if key == `tags` {
		return c.tags, true
	}
	return nil, false
}

func (c *tagsAnnotation) InitHash() px.OrderedMap {
	return px.SingletonMap(`tags`, c.tags)
}

func (c *tagsAnnotation) Tag(key string) string {
	if t, ok := c.tags.Get4(key); ok {
		return t.String()
	}
	return ``
}

func (c *tagsAnnotation) Tags() px.OrderedMap {
	return c.tags
}

func (r *tagsAnnotation) Validate(c px.Context, annotatedType px.Annotatable) {
}

func (tg *taggedType) Annotations() px.OrderedMap {
	return tg.annotations
}

func (tg *taggedType) Type() reflect.Type {
	return tg.typ
}

func (tg *taggedType) Tags() map[string]px.OrderedMap {
	return tg.parsedPuppetTags
}

func (tg *taggedType) OtherTags() map[string]map[string]string {
	return tg.tags
}

func (tg *taggedType) initTags() {
	fs := Fields(tg.typ)
	nf := len(fs)
	tags := make(map[string]map[string]string, 7)
	puppet := make(map[string]string)
	if nf > 0 {
		for i, f := range fs {
			if i == 0 && f.Anonymous {
				// Parent
				continue
			}
			if f.PkgPath != `` {
				// Unexported
				continue
			}
			ft := ParseTags(string(f.Tag))
			if p, ok := ft[`puppet`]; ok {
				puppet[f.Name] = p
				delete(ft, `puppet`)
			}
			if len(ft) > 0 {
				tags[f.Name] = ft
			}
		}
	}
	if tg.puppetTags != nil && len(tg.puppetTags) > 0 {
		for k, v := range tg.puppetTags {
			puppet[k] = v
		}
	}
	pt := make(map[string]px.OrderedMap, len(puppet))
	for k, v := range puppet {
		if h, ok := ParseTagHash(v); ok {
			pt[k] = h
		}
	}
	tg.parsedPuppetTags = pt
	tg.tags = tags
}

func ParseTags(tag string) map[string]string {
	result := make(map[string]string)
	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		tagLen := len(tag)
		if tagLen == 0 {
			break
		}

		var c rune
		for i, c = range tag {
			if c < ' ' || c == ':' || c == '"' || c == 0x7f {
				break
			}
		}
		if i == 0 || i+1 >= tagLen || c != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+2:] // Skip ':' and leading '"'
		esc := false
		tb := bytes.NewBufferString(``)
		for i, c = range tag {
			if esc {
				tb.WriteRune(c)
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				break
			} else {
				tb.WriteRune(c)
			}
		}
		if esc || c != '"' {
			break
		}
		result[name] = tb.String()
		tag = tag[i+1:]
	}
	return result
}
