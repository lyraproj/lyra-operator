package types

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"sync/atomic"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/utils"
	"github.com/lyraproj/semver/semver"
)

const (
	KeyNameAuthority = `name_authority`
	KeyReferences    = `references`
	KeyTypes         = `types`
	KeyVersion       = `version`
	KeyVersionRange  = `version_range`
)

var TypeSetMetaType px.ObjectType

var TypeSimpleTypeName = NewPatternType([]*RegexpType{NewRegexpType(`\A[A-Z]\w*\z`)})
var TypeQualifiedReference = NewPatternType([]*RegexpType{NewRegexpType(`\A[A-Z][\w]*(?:::[A-Z][\w]*)*\z`)})

var typeStringOrVersion = NewVariantType(stringTypeNotEmpty, DefaultSemVerType())
var typeStringOrRange = NewVariantType(stringTypeNotEmpty, DefaultSemVerRangeType())
var typeStringOrUri = NewVariantType(stringTypeNotEmpty, DefaultUriType())

var typeTypeReferenceInit = NewStructType([]*StructElement{
	newStructElement2(keyName, TypeQualifiedReference),
	newStructElement2(KeyVersionRange, typeStringOrRange),
	NewStructElement(newOptionalType3(KeyNameAuthority), typeStringOrUri),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations)})

var typeTypesetInit = NewStructType([]*StructElement{
	NewStructElement(newOptionalType3(px.KeyPcoreUri), typeStringOrUri),
	newStructElement2(px.KeyPcoreVersion, typeStringOrVersion),
	NewStructElement(newOptionalType3(KeyNameAuthority), typeStringOrUri),
	NewStructElement(newOptionalType3(keyName), TypeTypeName),
	NewStructElement(newOptionalType3(KeyVersion), typeStringOrVersion),
	NewStructElement(newOptionalType3(KeyTypes),
		NewHashType(TypeSimpleTypeName,
			NewVariantType(DefaultTypeType(), TypeObjectInitHash), NewIntegerType(1, math.MaxInt64))),
	NewStructElement(newOptionalType3(KeyReferences),
		NewHashType(TypeSimpleTypeName, typeTypeReferenceInit, NewIntegerType(1, math.MaxInt64))),
	NewStructElement(newOptionalType3(keyAnnotations), typeAnnotations)})

func init() {
	oneArgCtor := func(ctx px.Context, args []px.Value) px.Value {
		return newTypeSetType2(args[0].(*Hash), ctx.Loader())
	}
	TypeSetMetaType = MakeObjectType(`Pcore::TypeSet`, AnyMetaType,
		WrapStringToValueMap(map[string]px.Value{
			`attributes`: singletonMap(`_pcore_init_hash`, typeTypesetInit)}), true,
		// Hash constructor is equal to the positional arguments constructor
		oneArgCtor, oneArgCtor)
}

type (
	typeSetReference struct {
		annotatable
		name          string
		owner         *typeSet
		nameAuthority px.URI
		versionRange  semver.VersionRange
		typeSet       *typeSet
	}

	typeSet struct {
		annotatable
		hashKey       px.HashKey
		dcToCcMap     map[string]string
		name          string
		nameAuthority px.URI
		pcoreURI      px.URI
		pcoreVersion  semver.Version
		version       semver.Version
		typedName     px.TypedName
		types         *Hash
		references    map[string]*typeSetReference
		loader        px.Loader
		deferredInit  px.OrderedMap
	}
)

func newTypeSetReference(t *typeSet, ref *Hash) *typeSetReference {
	r := &typeSetReference{
		owner:         t,
		nameAuthority: uriArg(ref, KeyNameAuthority, t.nameAuthority),
		name:          stringArg(ref, keyName, ``),
		versionRange:  versionRangeArg(ref, KeyVersionRange, nil),
	}
	r.annotatable.initialize(ref)
	return r
}

func (r *typeSetReference) initHash() *hash.StringHash {
	h := r.annotatable.initHash()
	if r.nameAuthority != r.owner.nameAuthority {
		h.Put(KeyNameAuthority, stringValue(string(r.nameAuthority)))
	}
	h.Put(keyName, stringValue(r.name))
	h.Put(KeyVersionRange, WrapSemVerRange(r.versionRange))
	return h
}

func (r *typeSetReference) Equals(other interface{}, g px.Guard) bool {
	if or, ok := other.(*typeSetReference); ok {
		return r.name == or.name && r.nameAuthority == or.nameAuthority && r.versionRange == or.versionRange && r.typeSet.Equals(or.typeSet, g)
	}
	return false
}

func (r *typeSetReference) resolve(c px.Context) {
	tn := px.NewTypedName2(px.NsType, r.name, r.nameAuthority)
	loadedEntry := c.Loader().LoadEntry(c, tn)
	if loadedEntry != nil {
		if ts, ok := loadedEntry.Value().(*typeSet); ok {
			ts = ts.Resolve(c).(*typeSet)
			if r.versionRange.Includes(ts.version) {
				r.typeSet = ts
				return
			}
			panic(px.Error(px.TypesetReferenceMismatch, issue.H{`name`: r.owner.name, `ref_name`: r.name, `version_range`: r.versionRange, `actual`: ts.version}))
		}
	}
	var v interface{}
	if loadedEntry != nil {
		v = loadedEntry.Value()
	}
	if v == nil {
		panic(px.Error(px.TypesetReferenceUnresolved, issue.H{`name`: r.owner.name, `ref_name`: r.name}))
	}
	var typeName string
	if vt, ok := v.(px.Type); ok {
		typeName = vt.Name()
	} else if vv, ok := v.(px.Value); ok {
		typeName = vv.PType().Name()
	} else {
		typeName = fmt.Sprintf("%T", v)
	}
	panic(px.Error(px.TypesetReferenceBadType, issue.H{`name`: r.owner.name, `ref_name`: r.name, `type_name`: typeName}))
}

var typeSetTypeDefault = &typeSet{
	name:          `TypeSet`,
	nameAuthority: px.RuntimeNameAuthority,
	pcoreURI:      px.PcoreUri,
	pcoreVersion:  px.PcoreVersion,
	version:       semver.Zero,
}

var typeSetId = int64(0)

func AllocTypeSetType() *typeSet {
	return &typeSet{
		annotatable: annotatable{annotations: emptyMap},
		dcToCcMap:   make(map[string]string, 17),
		hashKey:     px.HashKey(fmt.Sprintf("\x00tTypeSet%d", atomic.AddInt64(&typeSetId, 1))),
		types:       emptyMap,
		references:  map[string]*typeSetReference{},
	}
}

func (t *typeSet) Initialize(c px.Context, args []px.Value) {
	if len(args) == 1 {
		if h, ok := args[0].(px.OrderedMap); ok {
			t.InitFromHash(c, h)
			return
		}
	}
	panic(px.Error(px.Failure, issue.H{`message`: `internal error when creating an TypeSet data type`}))
}

func NewTypeSet(na px.URI, name string, initHash px.OrderedMap) px.TypeSet {
	obj := AllocTypeSetType()
	obj.nameAuthority = na
	if name == `` {
		if initHash.IsEmpty() {
			return DefaultTypeSetType().(*typeSet)
		}
		name = initHash.Get5(keyName, emptyString).String()
	}
	obj.name = name
	obj.deferredInit = initHash
	return obj
}

func newTypeSetType2(initHash px.OrderedMap, loader px.Loader) px.TypeSet {
	obj := NewTypeSet(loader.NameAuthority(), ``, initHash).(*typeSet)
	obj.loader = loader
	return obj
}

func DefaultTypeSetType() px.TypeSet {
	return typeSetTypeDefault
}

func (t *typeSet) Annotations() *Hash {
	return t.annotations
}

func (t *typeSet) Accept(v px.Visitor, g px.Guard) {
	v(t)
	// TODO: Visit typeset members
}

func (t *typeSet) Default() px.Type {
	return typeSetTypeDefault
}

func (t *typeSet) Equals(other interface{}, guard px.Guard) bool {
	if ot, ok := other.(*typeSet); ok {
		return t.name == ot.name && t.nameAuthority == ot.nameAuthority && t.pcoreURI == ot.pcoreURI && t.pcoreVersion.Equals(ot.pcoreVersion) && t.version.Equals(ot.version)
	}
	return false
}

func (t *typeSet) Generic() px.Type {
	return DefaultTypeSetType()
}

func (t *typeSet) InitFromHash(c px.Context, initHash px.OrderedMap) {
	px.AssertInstance(`typeset initializer`, typeTypesetInit, initHash)
	t.name = stringArg(initHash, keyName, t.name)
	t.nameAuthority = uriArg(initHash, KeyNameAuthority, t.nameAuthority)

	t.pcoreVersion = versionArg(initHash, px.KeyPcoreVersion, nil)
	if !px.ParsablePcoreVersions.Includes(t.pcoreVersion) {
		panic(px.Error(px.UnhandledPcoreVersion,
			issue.H{`name`: t.name, `expected_range`: px.ParsablePcoreVersions, `pcore_version`: t.pcoreVersion}))
	}
	t.pcoreURI = uriArg(initHash, px.KeyPcoreUri, ``)
	t.version = versionArg(initHash, KeyVersion, nil)
	t.types = hashArg(initHash, KeyTypes)
	t.types.EachKey(func(kv px.Value) {
		key := kv.String()
		t.dcToCcMap[strings.ToLower(key)] = key
	})

	refs := hashArg(initHash, KeyReferences)
	if !refs.IsEmpty() {
		refMap := make(map[string]*typeSetReference, 7)
		rootMap := make(map[px.URI]map[string][]semver.VersionRange, 7)
		refs.EachPair(func(k, v px.Value) {
			refAlias := k.String()

			if t.types.IncludesKey(k) {
				panic(px.Error(px.TypesetAliasCollides,
					issue.H{`name`: t.name, `ref_alias`: refAlias}))
			}

			if _, ok := refMap[refAlias]; ok {
				panic(px.Error(px.TypesetReferenceDuplicate,
					issue.H{`name`: t.name, `ref_alias`: refAlias}))
			}

			ref := newTypeSetReference(t, v.(*Hash))
			refName := ref.name
			refNA := ref.nameAuthority
			naRoots, found := rootMap[refNA]
			if !found {
				naRoots = make(map[string][]semver.VersionRange, 3)
				rootMap[refNA] = naRoots
			}

			if ranges, found := naRoots[refName]; found {
				for _, rng := range ranges {
					if rng.Intersection(ref.versionRange) != nil {
						panic(px.Error(px.TypesetReferenceOverlap,
							issue.H{`name`: t.name, `ref_na`: refNA, `ref_name`: refName}))
					}
				}
				naRoots[refName] = append(ranges, ref.versionRange)
			} else {
				naRoots[refName] = []semver.VersionRange{ref.versionRange}
			}

			refMap[refAlias] = ref
			t.dcToCcMap[strings.ToLower(refAlias)] = refAlias
		})
		t.references = refMap
	}
	t.annotatable.initialize(initHash.(*Hash))
}

func (t *typeSet) Get(key string) (value px.Value, ok bool) {
	switch key {
	case px.KeyPcoreUri:
		if t.pcoreURI == `` {
			return undef, true
		}
		return WrapURI2(string(t.pcoreURI)), true
	case px.KeyPcoreVersion:
		return WrapSemVer(t.pcoreVersion), true
	case KeyNameAuthority:
		if t.nameAuthority == `` {
			return undef, true
		}
		return WrapURI2(string(t.nameAuthority)), true
	case keyName:
		return stringValue(t.name), true
	case KeyVersion:
		if t.version == nil {
			return undef, true
		}
		return WrapSemVer(t.version), true
	case KeyTypes:
		return t.types, true
	case KeyReferences:
		return t.referencesHash(), true
	}
	return nil, false
}

func (t *typeSet) GetType(typedName px.TypedName) (px.Type, bool) {
	if !(typedName.Namespace() == px.NsType && typedName.Authority() == t.nameAuthority) {
		return nil, false
	}

	segments := typedName.Parts()
	first := segments[0]
	if len(segments) == 1 {
		if found, ok := t.GetType2(first); ok {
			return found, true
		}
	}

	if len(t.references) == 0 {
		return nil, false
	}

	tsRef, ok := t.references[first]
	if !ok {
		tsRef, ok = t.references[t.dcToCcMap[first]]
		if !ok {
			return nil, false
		}
	}

	typeSet := tsRef.typeSet
	switch len(segments) {
	case 1:
		return typeSet, true
	case 2:
		return typeSet.GetType2(segments[1])
	default:
		return typeSet.GetType(typedName.Child())
	}
}

func (t *typeSet) GetType2(name string) (px.Type, bool) {
	v := t.types.Get6(name, func() px.Value {
		return t.types.Get5(t.dcToCcMap[name], nil)
	})
	if found, ok := v.(px.Type); ok {
		return found, true
	}
	return nil, false
}

func (t *typeSet) InitHash() px.OrderedMap {
	return WrapStringPValue(t.initHash())
}

func (t *typeSet) IsInstance(o px.Value, g px.Guard) bool {
	return t.IsAssignable(o.PType(), g)
}

func (t *typeSet) IsAssignable(other px.Type, g px.Guard) bool {
	if ot, ok := other.(*typeSet); ok {
		return t.Equals(typeSetTypeDefault, g) || t.Equals(ot, g)
	}
	return false
}

func (t *typeSet) MetaType() px.ObjectType {
	return TypeSetMetaType
}

func (t *typeSet) Name() string {
	return t.name
}

func (t *typeSet) NameAuthority() px.URI {
	return t.nameAuthority
}

func (t *typeSet) TypedName() px.TypedName {
	return t.typedName
}

func (t *typeSet) Parameters() []px.Value {
	if t.Equals(typeSetTypeDefault, nil) {
		return px.EmptyValues
	}
	return []px.Value{t.InitHash()}
}

func (t *typeSet) Resolve(c px.Context) px.Type {
	ihe := t.deferredInit
	if ihe == nil {
		return t
	}

	t.deferredInit = nil
	initHash := t.resolveDeferred(c, ihe)
	t.loader = c.Loader()
	t.InitFromHash(c, initHash)
	t.typedName = px.NewTypedName2(px.NsType, t.name, t.nameAuthority)

	for _, ref := range t.references {
		ref.resolve(c)
	}
	c.DoWithLoader(px.NewTypeSetLoader(c.Loader(), t), func() {
		if t.nameAuthority == `` {
			t.nameAuthority = t.resolveNameAuthority(initHash, c, nil)
		}
		t.types = t.types.MapValues(func(tp px.Value) px.Value {
			if rtp, ok := tp.(px.ResolvableType); ok {
				return rtp.Resolve(c)
			}
			return tp
		}).(*Hash)
	})
	return t
}

func (t *typeSet) Types() px.OrderedMap {
	return t.types
}

func (t *typeSet) Version() semver.Version {
	return t.version
}

func (t *typeSet) String() string {
	return px.ToString2(t, Expanded)
}

func (t *typeSet) ToString(b io.Writer, s px.FormatContext, g px.RDetect) {
	f := px.GetFormat(s.FormatMap(), t.PType())
	switch f.FormatChar() {
	case 's', 'p':
		quoted := f.IsAlt() && f.FormatChar() == 's'
		if quoted || f.HasStringFlags() {
			bld := bytes.NewBufferString(``)
			t.basicTypeToString(bld, f, s, g)
			f.ApplyStringFlags(b, bld.String(), quoted)
		} else {
			t.basicTypeToString(b, f, s, g)
		}
	default:
		panic(s.UnsupportedFormat(t.PType(), `sp`, f))
	}
}

func (t *typeSet) PType() px.Type {
	return &TypeType{t}
}

func (t *typeSet) basicTypeToString(b io.Writer, f px.Format, s px.FormatContext, g px.RDetect) {
	if t.Equals(DefaultTypeSetType(), nil) {
		utils.WriteString(b, `TypeSet`)
		return
	}

	if ex, ok := s.Property(`expanded`); !(ok && ex == `true`) {
		name := t.Name()
		if ts, ok := s.Property(`typeSet`); ok {
			name = stripTypeSetName(ts, name)
		}
		utils.WriteString(b, name)
		return
	}
	s = s.WithProperties(map[string]string{`typeSet`: t.Name(), `typeSetParent`: `true`})

	utils.WriteString(b, `TypeSet[{`)
	indent1 := s.Indentation()
	indent2 := indent1.Increase(f.IsAlt())
	indent3 := indent2.Increase(f.IsAlt())
	padding1 := ``
	padding2 := ``
	padding3 := ``
	if f.IsAlt() {
		padding1 = indent1.Padding()
		padding2 = indent2.Padding()
		padding3 = indent3.Padding()
	}

	cf := f.ContainerFormats()
	if cf == nil {
		cf = DefaultContainerFormats
	}

	ctx2 := px.NewFormatContext2(indent2, s.FormatMap(), s.Properties())
	cti2 := px.NewFormatContext2(indent2, cf, s.Properties())
	ctx3 := px.NewFormatContext2(indent3, s.FormatMap(), s.Properties())

	first2 := true
	t.initHash().EachPair(func(key string, vi interface{}) {
		if first2 {
			first2 = false
		} else {
			utils.WriteString(b, `,`)
			if !f.IsAlt() {
				utils.WriteString(b, ` `)
			}
		}
		value := vi.(px.Value)
		if f.IsAlt() {
			utils.WriteString(b, "\n")
			utils.WriteString(b, padding2)
		}
		utils.WriteString(b, key)
		utils.WriteString(b, ` => `)
		switch key {
		case `pcore_uri`, `pcore_version`, `name_authority`, `version`:
			utils.PuppetQuote(b, value.String())
		case `types`, `references`:
			// The keys should not be quoted in this hash
			utils.WriteString(b, `{`)
			first3 := true
			value.(*Hash).EachPair(func(typeName, typ px.Value) {
				if first3 {
					first3 = false
				} else {
					utils.WriteString(b, `,`)
					if !f.IsAlt() {
						utils.WriteString(b, ` `)
					}
				}
				if f.IsAlt() {
					utils.WriteString(b, "\n")
					utils.WriteString(b, padding3)
				}
				utils.WriteString(b, typeName.String())
				utils.WriteString(b, ` => `)
				typ.ToString(b, ctx3, g)
			})
			if f.IsAlt() {
				utils.WriteString(b, "\n")
				utils.WriteString(b, padding2)
			}
			utils.WriteString(b, "}")
		default:
			cx := cti2
			if isContainer(value, s) {
				cx = ctx2
			}
			value.ToString(b, cx, g)
		}
	})
	if f.IsAlt() {
		utils.WriteString(b, "\n")
		utils.WriteString(b, padding1)
	}
	utils.WriteString(b, "}]")
}

func (t *typeSet) initHash() *hash.StringHash {
	h := t.annotatable.initHash()
	if t.pcoreURI != `` {
		h.Put(px.KeyPcoreUri, WrapURI2(string(t.pcoreURI)))
	}
	h.Put(px.KeyPcoreVersion, WrapSemVer(t.pcoreVersion))
	if t.nameAuthority != `` {
		h.Put(KeyNameAuthority, WrapURI2(string(t.nameAuthority)))
	}
	h.Put(keyName, stringValue(t.name))
	if t.version != nil {
		h.Put(KeyVersion, WrapSemVer(t.version))
	}
	if !t.types.IsEmpty() {
		h.Put(KeyTypes, t.types)
	}
	if len(t.references) > 0 {
		h.Put(KeyReferences, t.referencesHash())
	}
	return h
}

func (t *typeSet) referencesHash() *Hash {
	if len(t.references) == 0 {
		return emptyMap
	}
	entries := make([]*HashEntry, len(t.references))
	idx := 0
	for key, tr := range t.references {
		entries[idx] = WrapHashEntry2(key, WrapStringPValue(tr.initHash()))
		idx++
	}
	return WrapHash(entries)
}

func (t *typeSet) resolveDeferred(c px.Context, lh px.OrderedMap) *Hash {
	entries := make([]*HashEntry, 0)
	types := hash.NewStringHash(16)

	var typesHash *Hash

	lh.Each(func(v px.Value) {
		le := v.(px.MapEntry)
		key := le.Key().String()
		if key == KeyTypes || key == KeyReferences {
			if key == KeyTypes {
				typesHash = emptyMap
			}

			// Avoid resolving qualified references into types
			if vh, ok := le.Value().(px.OrderedMap); ok {
				xes := make([]*HashEntry, 0)
				vh.Each(func(v px.Value) {
					he := v.(px.MapEntry)
					var name string
					if qr, ok := he.Key().(*DeferredType); ok && len(qr.Parameters()) == 0 {
						name = qr.Name()
					} else if tr, ok := he.Key().(*TypeReferenceType); ok {
						name = tr.typeString
					} else {
						name = resolveValue(c, he.Key()).String()
					}
					if key == KeyTypes {
						// Defer type resolution until all types are known
						types.Put(name, he.Value())
					} else {
						xes = append(xes, WrapHashEntry2(name, resolveValue(c, he.Value())))
					}
				})
				if key == KeyReferences {
					entries = append(entries, WrapHashEntry2(key, WrapHash(xes)))
				}
			} else {
				// Probably a bogus entry, will cause type error further on
				entries = append(entries, resolveEntry(c, le).(*HashEntry))
				if key == KeyTypes {
					typesHash = nil
				}
			}
		} else {
			entries = append(entries, resolveEntry(c, le).(*HashEntry))
		}
	})

	result := WrapHash(entries)
	nameAuth := t.resolveNameAuthority(result, c, nil)
	if !types.IsEmpty() {
		es := make([]*HashEntry, 0, types.Len())
		types.EachPair(func(typeName string, value interface{}) {
			fullName := fmt.Sprintf(`%s::%s`, t.name, typeName)
			var tp px.Type
			if tv, ok := value.(px.Type); ok {
				tp = tv
			} else {
				tp = NamedType(nameAuth, fullName, value.(px.Value))
			}
			es = append(es, WrapHashEntry2(typeName, tp))
		})
		typesHash = WrapHash(es)
	}
	if typesHash != nil {
		result = WrapHash(append(entries, WrapHashEntry2(KeyTypes, typesHash)))
	}
	return result
}

func (t *typeSet) resolveNameAuthority(hash px.OrderedMap, c px.Context, location issue.Location) px.URI {
	nameAuth := t.nameAuthority
	if nameAuth == `` {
		nameAuth = uriArg(hash, KeyNameAuthority, ``)
		if nameAuth == `` {
			nameAuth = c.Loader().NameAuthority()
		}
	}
	if nameAuth == `` {
		n := t.name
		if n == `` {
			n = stringArg(hash, keyName, ``)
		}
		var err error
		if location != nil {
			err = px.Error2(location, px.TypesetMissingNameAuthority, issue.H{`name`: n})
		} else {
			err = px.Error(px.TypesetMissingNameAuthority, issue.H{`name`: n})
		}
		panic(err)
	}
	return nameAuth
}
