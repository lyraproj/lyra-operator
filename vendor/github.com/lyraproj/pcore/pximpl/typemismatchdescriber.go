package pximpl

import (
	"bytes"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/pcore/utils"
)

type (
	pathElement struct {
		key      string
		pathType pathType
	}

	pathType string

	mismatchClass string

	mismatch interface {
		canonicalPath() []*pathElement
		path() []*pathElement
		pathString() string
		setPath(path []*pathElement)
		text() string
		class() mismatchClass
		equals(other mismatch) bool
	}

	expectedActualMismatch interface {
		mismatch
		actual() px.Type
		expected() px.Type
		setExpected(expected px.Type)
	}

	sizeMismatch interface {
		expectedActualMismatch
		from() int64
		to() int64
	}

	sizeMismatchFunc func(path []*pathElement, expected *types.IntegerType, actual *types.IntegerType) mismatch

	basicMismatch struct {
		p []*pathElement
	}

	unexpectedBlock struct{ basicMismatch }

	missingRequiredBlock struct{ basicMismatch }

	keyMismatch struct {
		basicMismatch
		key string
	}

	missingKey              struct{ keyMismatch }
	extraneousKey           struct{ keyMismatch }
	unresolvedTypeReference struct{ keyMismatch }

	basicEAMismatch struct {
		basicMismatch
		actualType   px.Type
		expectedType px.Type
	}

	typeMismatch      struct{ basicEAMismatch }
	patternMismatch   struct{ typeMismatch }
	basicSizeMismatch struct{ basicEAMismatch }
	countMismatch     struct{ basicSizeMismatch }
)

var NoMismatch []mismatch

const (
	subject    = pathType(``)
	entry      = pathType(`entry`)
	entryKey   = pathType(`key of entry`)
	_parameter = pathType(`parameter`)
	_return    = pathType(`return`)
	block      = pathType(`block`)
	index      = pathType(`index`)
	variant    = pathType(`variant`)
	signature  = pathType(`signature`)

	countMismatchClass           = mismatchClass(`countMismatch`)
	missingKeyClass              = mismatchClass(`missingKey`)
	missingRequiredBlockClass    = mismatchClass(`missingRequiredBlock`)
	extraneousKeyClass           = mismatchClass(`extraneousKey`)
	patternMismatchClass         = mismatchClass(`patternMismatch`)
	sizeMismatchClass            = mismatchClass(`sizeMismatch`)
	typeMismatchClass            = mismatchClass(`typeMismatch`)
	unexpectedBlockClass         = mismatchClass(`unexpectedBlock`)
	unresolvedTypeReferenceClass = mismatchClass(`unresolvedTypeReference`)
)

func (p pathType) String(key string) string {
	if p == subject || p == signature {
		return key
	}
	if p == block && key == `block` {
		return key
	}
	if p == _parameter && utils.IsDecimalInteger(key) {
		return fmt.Sprintf("parameter %s", key)
	}
	return fmt.Sprintf("%s '%s'", string(p), key)
}

func (pe *pathElement) String() string {
	return pe.pathType.String(pe.key)
}

func copyMismatch(m mismatch) mismatch {
	orig := reflect.Indirect(reflect.ValueOf(m))
	c := reflect.New(orig.Type())
	c.Elem().Set(orig)
	return c.Interface().(mismatch)
}

func withPath(m mismatch, path []*pathElement) mismatch {
	m = copyMismatch(m)
	m.setPath(path)
	return m
}

func chopPath(m mismatch, index int) mismatch {
	p := m.path()
	if index >= len(p) {
		return m
	}
	cp := make([]*pathElement, 0)
	for i, pe := range p {
		if i != index {
			cp = append(cp, pe)
		}
	}
	return withPath(m, cp)
}

func mergeMismatch(m mismatch, o mismatch, path []*pathElement) mismatch {
	m = withPath(m, path)
	switch m.(type) {
	case *typeMismatch:
		et := m.(*typeMismatch)
		if ot, ok := o.(*typeMismatch); ok {
			if ev, ok := et.expectedType.(*types.VariantType); ok {
				if ov, ok := ot.expectedType.(*types.VariantType); ok {
					ts := make([]px.Type, 0, len(ev.Types())+len(ov.Types()))
					ts = append(ts, ev.Types()...)
					ts = append(ts, ov.Types()...)
					et.setExpected(types.NewVariantType(types.UniqueTypes(ts)...))
				} else {
					et.setExpected(types.NewVariantType(types.UniqueTypes(types.CopyAppend(ev.Types(), ot.expectedType))...))
				}
			} else {
				if ov, ok := ot.expectedType.(*types.VariantType); ok {
					ts := make([]px.Type, 0, len(ov.Types())+1)
					ts = append(ts, et.expectedType)
					ts = append(ts, ov.Types()...)
					et.setExpected(types.NewVariantType(types.UniqueTypes(ts)...))
				} else {
					if !et.expectedType.Equals(ot.expectedType, nil) {
						et.setExpected(types.NewVariantType(et.expectedType, ot.expectedType))
					}
				}
			}
		}
	case sizeMismatch:
		esm := m.(sizeMismatch)
		if osm, ok := o.(sizeMismatch); ok {
			min := esm.from()
			if min > osm.from() {
				min = osm.from()
			}
			max := esm.to()
			if max < osm.to() {
				max = osm.to()
			}
			esm.setExpected(types.NewIntegerType(min, max))
		}
	case expectedActualMismatch:
		eam := m.(expectedActualMismatch)
		if oam, ok := o.(expectedActualMismatch); ok {
			eam.setExpected(oam.expected())
		}
	}
	return m
}

func joinPath(path []*pathElement) string {
	s := make([]string, len(path))
	for i, p := range path {
		s[i] = p.String()
	}
	return strings.Join(s, ` `)
}

func formatMismatch(m mismatch) string {
	p := m.path()
	variant := ``
	position := ``
	if len(p) > 0 {
		f := p[0]
		if f.pathType == signature {
			variant = fmt.Sprintf(` %s`, f.String())
			p = p[1:]
		}
		if len(p) > 0 {
			position = fmt.Sprintf(` %s`, joinPath(p))
		}
	}
	return message(m, variant, position)
}

func message(m mismatch, variant string, position string) string {
	if variant == `` && position == `` {
		return m.text()
	}
	return fmt.Sprintf("%s%s %s", variant, position, m.text())
}

func (m *basicMismatch) canonicalPath() []*pathElement {
	result := make([]*pathElement, 0)
	for _, p := range m.p {
		if p.pathType != variant && p.pathType != signature {
			result = append(result, p)
		}
	}
	return result
}

func (m *basicMismatch) class() mismatchClass {
	return ``
}

func (m *basicMismatch) path() []*pathElement {
	return m.p
}

func (m *basicMismatch) setPath(path []*pathElement) {
	m.p = path
}

func (m *basicMismatch) pathString() string {
	return joinPath(m.p)
}

func (m *basicMismatch) text() string {
	return ``
}

func (m *basicMismatch) equals(other mismatch) bool {
	return m.class() == other.class() && pathEquals(m.p, other.path())
}

func newUnexpectedBlock(path []*pathElement) mismatch {
	return &unexpectedBlock{basicMismatch{p: path}}
}

func (*unexpectedBlock) class() mismatchClass {
	return unexpectedBlockClass
}

func (*unexpectedBlock) text() string {
	return `does not expect a block`
}

func newMissingRequiredBlock(path []*pathElement) mismatch {
	return &missingRequiredBlock{basicMismatch{p: path}}
}

func (*missingRequiredBlock) class() mismatchClass {
	return missingRequiredBlockClass
}

func (*missingRequiredBlock) text() string {
	return `expects a block`
}

func (m *keyMismatch) equals(other mismatch) bool {
	if om, ok := other.(*keyMismatch); ok && pathEquals(m.p, other.path()) {
		return m.key == om.key
	}
	return false
}

func newMissingKey(path []*pathElement, key string) mismatch {
	return &missingKey{keyMismatch{basicMismatch{p: path}, key}}
}

func (*missingKey) class() mismatchClass {
	return missingKeyClass
}

func (m *missingKey) text() string {
	return fmt.Sprintf(`expects a value for key '%s'`, m.key)
}

func newExtraneousKey(path []*pathElement, key string) mismatch {
	return &extraneousKey{keyMismatch{basicMismatch{p: path}, key}}
}

func (*extraneousKey) class() mismatchClass {
	return extraneousKeyClass
}

func (m *extraneousKey) text() string {
	return fmt.Sprintf(`unrecognized key '%s'`, m.key)
}

func newUnresolvedTypeReference(path []*pathElement, key string) mismatch {
	return &unresolvedTypeReference{keyMismatch{basicMismatch{p: path}, key}}
}

func (*unresolvedTypeReference) class() mismatchClass {
	return unresolvedTypeReferenceClass
}

func (m *unresolvedTypeReference) text() string {
	return fmt.Sprintf(`references an unresolved type '%s'`, m.key)
}

func (ea *basicEAMismatch) equals(other mismatch) bool {
	if om, ok := other.(*basicEAMismatch); ok && pathEquals(ea.p, other.path()) {
		return ea.expectedType == om.expectedType && ea.actualType == om.actualType
	}
	return false
}

func (ea *basicEAMismatch) expected() px.Type {
	return ea.expectedType
}

func (ea *basicEAMismatch) actual() px.Type {
	return ea.actualType
}

func (ea *basicEAMismatch) setExpected(expected px.Type) {
	ea.expectedType = expected
}

func newTypeMismatch(path []*pathElement, expected px.Type, actual px.Type) mismatch {
	return &typeMismatch{basicEAMismatch{basicMismatch{p: path}, actual, expected}}
}

func (*typeMismatch) class() mismatchClass {
	return typeMismatchClass
}

func (tm *typeMismatch) text() string {
	e := tm.expectedType
	a := tm.actualType
	multi := false
	optional := false
	if opt, ok := e.(*types.OptionalType); ok {
		e = opt.ContainedType()
		optional = true
	}
	var as, es string
	if vt, ok := e.(*types.VariantType); ok {
		el := vt.Types()
		els := make([]string, len(el))
		if reportDetailed(el, a) {
			as = detailedToActualToS(el, a)
			for i, e := range el {
				els[i] = shortName(e)
			}
		} else {
			for i, e := range el {
				els[i] = shortName(e)
			}
			as = shortName(a)
		}
		if optional {
			els = append([]string{`Undef`}, els...)
		}
		switch len(els) {
		case 1:
			es = els[0]
		case 2:
			es = fmt.Sprintf(`%s or %s`, els[0], els[1])
			multi = true
		default:
			es = fmt.Sprintf(`%s, or %s`, strings.Join(els[0:len(els)-1], `, `), els[len(els)-1])
			multi = true
		}
	} else {
		el := []px.Type{e}
		if reportDetailed(el, a) {
			as = detailedToActualToS(el, a)
			es = px.ToString2(e, types.Expanded)
		} else {
			as = shortName(a)
			es = shortName(e)
		}
	}

	if multi {
		return fmt.Sprintf(`expects a value of type %s, got %s`, es, as)
	}
	return fmt.Sprintf(`expects %s %s value, got %s`, issue.Article(es), es, as)
}

func shortName(t px.Type) string {
	if tc, ok := t.(px.TypeWithContainedType); ok && !(tc.ContainedType() == nil || tc.ContainedType() == types.DefaultAnyType()) {
		return fmt.Sprintf("%s[%s]", t.Name(), tc.ContainedType().Name())
	}
	if t.Name() == `` {
		return `Object`
	}
	return t.Name()
}

func detailedToActualToS(es []px.Type, a px.Type) string {
	es = allResolved(es)
	if alwaysFullyDetailed(es, a) || anyAssignable(es, px.Generalize(a)) {
		if as, ok := a.(px.StringType); ok && as.Value() != nil {
			b := bytes.NewBufferString(``)
			utils.PuppetQuote(b, *as.Value())
			return b.String()
		}
		return px.ToString2(a, types.Expanded)
	}
	return a.Name()
}

func anyAssignable(es []px.Type, a px.Type) bool {
	for _, e := range es {
		if px.IsAssignable(e, a) {
			return true
		}
	}
	return false
}

func alwaysFullyDetailed(es []px.Type, a px.Type) bool {
	for _, e := range es {
		if px.Generalize(e).Equals(px.Generalize(a), nil) {
			return true
		}
		if _, ok := e.(*types.TypeAliasType); ok {
			return true
		}
		if _, ok := a.(*types.TypeAliasType); ok {
			return true
		}
		if specialization(e, a) {
			return true
		}
	}
	return false
}

func specialization(e px.Type, a px.Type) (result bool) {
	switch e.(type) {
	case *types.InitType:
		result = true
	case px.StringType:
		as, ok := a.(px.StringType)
		result = ok && as.Value() != nil
	case *types.StructType:
		_, result = a.(*types.HashType)
	case *types.TupleType:
		_, result = a.(*types.ArrayType)
	case px.ObjectType:
		_, result = a.(*types.StructType)
	default:
		result = false
	}
	return
}

func allResolved(es []px.Type) []px.Type {
	rs := make([]px.Type, len(es))
	for i, e := range es {
		if ea, ok := e.(*types.TypeAliasType); ok {
			e = ea.ResolvedType()
		}
		rs[i] = e
	}
	return rs
}

func reportDetailed(e []px.Type, a px.Type) bool {
	return alwaysFullyDetailed(e, a) || assignableToDefault(e, a)
}

func assignableToDefault(es []px.Type, a px.Type) bool {
	for _, e := range es {
		if ea, ok := e.(*types.TypeAliasType); ok {
			e = ea.ResolvedType()
		}
		if px.IsAssignable(px.DefaultFor(e), a) {
			return true
		}
	}
	return false
}

func newPatternMismatch(path []*pathElement, expected px.Type, actual px.Type) mismatch {
	return &patternMismatch{
		typeMismatch{
			basicEAMismatch{
				basicMismatch{p: path}, actual, expected}}}
}

func (*patternMismatch) class() mismatchClass {
	return patternMismatchClass
}

func (m *patternMismatch) text() string {
	e := m.expectedType
	valuePfx := ``
	if oe, ok := e.(*types.OptionalType); ok {
		e = oe.ContainedType()
		valuePfx = `an undef value or `
	}
	return fmt.Sprintf(`expects %sa match for %s, got %s`,
		valuePfx, px.ToString2(e, types.Expanded), m.actualString())
}

func (m *patternMismatch) actualString() string {
	a := m.actualType
	if as, ok := a.(px.StringType); ok && as.Value() != nil {
		return fmt.Sprintf(`'%s'`, *as.Value())
	}
	return shortName(a)
}

func newSizeMismatch(path []*pathElement, expected *types.IntegerType, actual *types.IntegerType) mismatch {
	return &basicSizeMismatch{
		basicEAMismatch{
			basicMismatch{p: path}, actual, expected}}
}

func (*basicSizeMismatch) class() mismatchClass {
	return sizeMismatchClass
}

func (m *basicSizeMismatch) from() int64 {
	return m.expectedType.(*types.IntegerType).Min()
}

func (m *basicSizeMismatch) to() int64 {
	return m.expectedType.(*types.IntegerType).Max()
}

func (m *basicSizeMismatch) text() string {
	return fmt.Sprintf(`expects size to be %s, got %s`,
		rangeToS(m.expectedType.(*types.IntegerType), `0`),
		rangeToS(m.actualType.(*types.IntegerType), `0`))
}

func rangeToS(rng *types.IntegerType, zeroString string) string {
	if rng.Min() == rng.Max() {
		if rng.Min() == 0 {
			return zeroString
		}
		return strconv.FormatInt(rng.Min(), 10)
	} else if rng.Min() == 0 {
		if rng.Max() == math.MaxInt64 {
			return `unbounded`
		}
		return fmt.Sprintf(`at most %d`, rng.Max())
	} else if rng.Max() == math.MaxInt64 {
		return fmt.Sprintf(`at least %d`, rng.Min())
	} else {
		return fmt.Sprintf("between %d and %d", rng.Min(), rng.Max())
	}
}

func newCountMismatch(path []*pathElement, expected *types.IntegerType, actual *types.IntegerType) mismatch {
	return &countMismatch{basicSizeMismatch{
		basicEAMismatch{
			basicMismatch{p: path}, actual, expected}}}
}

func (*countMismatch) class() mismatchClass {
	return countMismatchClass
}

func (tm *countMismatch) text() string {
	ei := tm.expectedType.(*types.IntegerType)
	suffix := `s`
	if ei.Min() == 1 && (ei.Max() == 1 || ei.Max() == math.MaxInt64) || ei.Min() == 0 && ei.Max() == 1 {
		suffix = ``
	}

	return fmt.Sprintf(`expects %s argument%s, got %s`,
		rangeToS(ei, `no`), suffix,
		rangeToS(tm.actualType.(*types.IntegerType), `none`))
}

func describeOptionalType(expected *types.OptionalType, original, actual px.Type, path []*pathElement) []mismatch {
	if _, ok := actual.(*types.UndefType); ok {
		return NoMismatch
	}
	if _, ok := original.(*types.TypeAliasType); !ok {
		// If the original expectation is an alias, it must now track the optional type instead
		original = expected
	}
	return internalDescribe(expected.ContainedType(), original, actual, path)
}

func describeEnumType(expected *types.EnumType, original, actual px.Type, path []*pathElement) []mismatch {
	if px.IsAssignable(expected, actual) {
		return []mismatch{}
	}
	return []mismatch{newPatternMismatch(path, original, actual)}
}

func describeInitType(expected *types.InitType, actual px.Type, path []*pathElement) []mismatch {
	if px.IsAssignable(expected, actual) {
		return []mismatch{}
	}

	ds := make([]mismatch, 0, 4)
	ix := 0
	at := types.NewTupleType([]px.Type{actual}, nil)
	expected.EachSignature(func(sg px.Signature) {
		ds = append(ds, describeSignatureArguments(sg, at, append(path, &pathElement{strconv.Itoa(ix), signature}))...)
	})
	return ds
}

func describePatternType(expected *types.PatternType, original, actual px.Type, path []*pathElement) []mismatch {
	if px.IsAssignable(expected, actual) {
		return NoMismatch
	}
	return []mismatch{newPatternMismatch(path, original, actual)}
}

func describeTypeAliasType(expected *types.TypeAliasType, actual px.Type, path []*pathElement) []mismatch {
	return internalDescribe(px.Normalize(expected.ResolvedType()), expected, actual, path)
}

func describeArrayType(expected *types.ArrayType, original, actual px.Type, path []*pathElement) []mismatch {
	descriptions := make([]mismatch, 0, 4)
	et := expected.ElementType()
	if ta, ok := actual.(*types.TupleType); ok {
		if px.IsAssignable(expected.Size(), ta.Size()) {
			for ax, at := range ta.Types() {
				if !px.IsAssignable(et, at) {
					descriptions = append(descriptions, internalDescribe(et, et, at, pathWith(path, &pathElement{strconv.Itoa(ax), index}))...)
				}
			}
		} else {
			descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), ta.Size()))
		}
	} else if aa, ok := actual.(*types.ArrayType); ok {
		if !px.IsAssignable(expected, aa) {
			if px.IsAssignable(expected.Size(), aa.Size()) {
				descriptions = append(descriptions, newTypeMismatch(path, original, types.NewArrayType(aa.ElementType(), nil)))
			} else {
				descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), aa.Size()))
			}
		}
	} else {
		descriptions = append(descriptions, newTypeMismatch(path, original, actual))
	}
	return descriptions
}

func describeHashType(expected *types.HashType, original, actual px.Type, path []*pathElement) []mismatch {
	descriptions := make([]mismatch, 0, 4)
	kt := expected.KeyType()
	vt := expected.ValueType()
	if sa, ok := actual.(*types.StructType); ok {
		if px.IsAssignable(expected.Size(), sa.Size()) {
			for _, al := range sa.Elements() {
				descriptions = append(descriptions, internalDescribe(kt, kt, al.Key(), pathWith(path, &pathElement{al.Name(), entryKey}))...)
				descriptions = append(descriptions, internalDescribe(vt, vt, al.Value(), pathWith(path, &pathElement{al.Name(), entry}))...)
			}
		} else {
			descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), sa.Size()))
		}
	} else if ha, ok := actual.(*types.HashType); ok {
		if !px.IsAssignable(expected, ha) {
			if px.IsAssignable(expected.Size(), ha.Size()) {
				descriptions = append(descriptions, newTypeMismatch(path, original, types.NewHashType(ha.KeyType(), ha.ValueType(), nil)))
			} else {
				descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), ha.Size()))
			}
		}
	} else {
		descriptions = append(descriptions, newTypeMismatch(path, original, actual))
	}
	return descriptions
}

func describeStructType(expected *types.StructType, original, actual px.Type, path []*pathElement) []mismatch {
	descriptions := make([]mismatch, 0, 4)
	if sa, ok := actual.(*types.StructType); ok {
		h2 := sa.HashedMembersCloned()
		for _, e1 := range expected.Elements() {
			key := e1.Name()
			e2, ok := h2[key]
			if ok {
				delete(h2, key)
				ek := e1.ActualKeyType()
				descriptions = append(descriptions, internalDescribe(ek, ek, e2.ActualKeyType(), pathWith(path, &pathElement{key, entryKey}))...)
				descriptions = append(descriptions, internalDescribe(e1.Value(), e1.Value(), e2.Value(), pathWith(path, &pathElement{key, entry}))...)
			} else {
				if !e1.Optional() {
					descriptions = append(descriptions, newMissingKey(path, e1.Name()))
				}
			}
		}
		for key := range h2 {
			descriptions = append(descriptions, newExtraneousKey(path, key))
		}
	} else if ha, ok := actual.(*types.HashType); ok {
		if !px.IsAssignable(expected, ha) {
			if px.IsAssignable(expected.Size(), ha.Size()) {
				descriptions = append(descriptions, newTypeMismatch(path, original, types.NewHashType(ha.KeyType(), ha.ValueType(), nil)))
			} else {
				descriptions = append(descriptions, newSizeMismatch(path, expected.Size(), ha.Size()))
			}
		}
	} else {
		descriptions = append(descriptions, newTypeMismatch(path, original, actual))
	}
	return descriptions
}

func describeTupleType(expected *types.TupleType, original, actual px.Type, path []*pathElement) []mismatch {
	return describeTuple(expected, original, actual, path, newCountMismatch)
}

func describeArgumentTuple(expected *types.TupleType, actual px.Type, path []*pathElement) []mismatch {
	return describeTuple(expected, expected, actual, path, newCountMismatch)
}

func describeTuple(expected *types.TupleType, original, actual px.Type, path []*pathElement, sm sizeMismatchFunc) []mismatch {
	if aa, ok := actual.(*types.ArrayType); ok {
		if len(expected.Types()) == 0 || px.IsAssignable(expected, aa) {
			return NoMismatch
		}
		t2Entry := aa.ElementType()
		if t2Entry == types.DefaultAnyType() {
			// Array of anything can not be assigned (unless tuple is tuple of anything) - this case
			// was handled at the top of this method.
			return []mismatch{newTypeMismatch(path, original, actual)}
		}

		if !px.IsAssignable(expected.Size(), aa.Size()) {
			return []mismatch{sm(path, expected.Size(), aa.Size())}
		}

		descriptions := make([]mismatch, 0, 4)
		for ex, et := range expected.Types() {
			descriptions = append(descriptions, internalDescribe(et, et, aa.ElementType(),
				pathWith(path, &pathElement{strconv.Itoa(ex), index}))...)
		}
		return descriptions
	}

	if at, ok := actual.(*types.TupleType); ok {
		if expected.Equals(actual, nil) || px.IsAssignable(expected, at) {
			return NoMismatch
		}

		if !px.IsAssignable(expected.Size(), at.Size()) {
			return []mismatch{sm(path, expected.Size(), at.Size())}
		}

		exl := len(expected.Types())
		if exl == 0 {
			return NoMismatch
		}

		descriptions := make([]mismatch, 0, 4)
		for ax, at := range at.Types() {
			if ax >= exl {
				ex := exl - 1
				ext := expected.Types()[ex]
				descriptions = append(descriptions, internalDescribe(ext, ext, at,
					pathWith(path, &pathElement{strconv.Itoa(ax), index}))...)
			}
		}
		return descriptions
	}

	return []mismatch{newTypeMismatch(path, original, actual)}
}

func pathEquals(a, b []*pathElement) bool {
	n := len(a)
	if n != len(b) {
		return false
	}
	for i := 0; i < n; i++ {
		if *(a[i]) != *(b[i]) {
			return false
		}
	}
	return true
}

func pathWith(path []*pathElement, elem *pathElement) []*pathElement {
	top := len(path)
	pc := make([]*pathElement, top+1)
	copy(pc, path)
	pc[top] = elem
	return pc
}

func describeCallableType(expected *types.CallableType, original, actual px.Type, path []*pathElement) []mismatch {
	if ca, ok := actual.(*types.CallableType); ok {
		ep := expected.ParametersType()
		paramErrors := NoMismatch
		if ep != nil {
			ap := ca.ParametersType()
			paramErrors = describeArgumentTuple(ep.(*types.TupleType), types.NilAs(types.DefaultTupleType(), ap), path)
		}
		if len(paramErrors) == 0 {
			er := expected.ReturnType()
			ar := types.NilAs(types.DefaultAnyType(), ca.ReturnType())
			if er == nil || px.IsAssignable(er, ar) {
				eb := expected.BlockType()
				ab := ca.BlockType()
				if eb == nil || px.IsAssignable(eb, types.NilAs(types.DefaultUndefType(), ab)) {
					return NoMismatch
				}
				if ab == nil {
					return []mismatch{newMissingRequiredBlock(path)}
				}
				return []mismatch{newTypeMismatch(pathWith(path, &pathElement{``, block}), eb, ab)}
			}
			return []mismatch{newTypeMismatch(pathWith(path, &pathElement{``, _return}), er, ar)}
		}
		return paramErrors
	}
	return []mismatch{newTypeMismatch(path, original, actual)}
}

func describeAnyType(expected px.Type, original, actual px.Type, path []*pathElement) []mismatch {
	if px.IsAssignable(expected, actual) {
		return NoMismatch
	}
	return []mismatch{newTypeMismatch(path, original, actual)}
}

func describe(expected px.Type, actual px.Type, path []*pathElement) []mismatch {
	var unresolved *types.TypeReferenceType
	expected.Accept(func(t px.Type) {
		if unresolved == nil {
			if ur, ok := t.(*types.TypeReferenceType); ok {
				unresolved = ur
			}
		}
	}, nil)

	if unresolved != nil {
		return []mismatch{newUnresolvedTypeReference(path, unresolved.TypeString())}
	}
	return internalDescribe(px.Normalize(expected), expected, actual, path)
}

func internalDescribe(expected px.Type, original, actual px.Type, path []*pathElement) []mismatch {
	switch expected := expected.(type) {
	case *types.VariantType:
		return describeVariantType(expected, original, actual, path)
	case *types.StructType:
		return describeStructType(expected, original, actual, path)
	case *types.HashType:
		return describeHashType(expected, original, actual, path)
	case *types.TupleType:
		return describeTupleType(expected, original, actual, path)
	case *types.ArrayType:
		return describeArrayType(expected, original, actual, path)
	case *types.CallableType:
		return describeCallableType(expected, original, actual, path)
	case *types.OptionalType:
		return describeOptionalType(expected, original, actual, path)
	case *types.PatternType:
		return describePatternType(expected, original, actual, path)
	case *types.EnumType:
		return describeEnumType(expected, original, actual, path)
	case *types.InitType:
		return describeInitType(expected, actual, path)
	case *types.TypeAliasType:
		return describeTypeAliasType(expected, actual, path)
	default:
		return describeAnyType(expected, original, actual, path)
	}
}

func describeVariantType(expected *types.VariantType, original, actual px.Type, path []*pathElement) []mismatch {
	vs := make([]mismatch, 0, len(expected.Types()))
	ts := expected.Types()
	if _, ok := original.(*types.OptionalType); ok {
		ts = types.CopyAppend(ts, types.DefaultUndefType())
	}

	for ex, vt := range ts {
		if px.IsAssignable(vt, actual) {
			return NoMismatch
		}
		d := internalDescribe(vt, vt, actual, pathWith(path, &pathElement{strconv.Itoa(ex), variant}))
		vs = append(vs, d...)
	}

	ds := mergeDescriptions(len(path), sizeMismatchClass, vs)
	if _, ok := original.(*types.TypeAliasType); ok && len(ds) == 1 {
		// All variants failed in this alias so we report it as a mismatch on the alias
		// rather than reporting individual failures of the variants
		ds = []mismatch{newTypeMismatch(path, original, actual)}
	}
	return ds
}

func mergeDescriptions(varyingPathPosition int, sm mismatchClass, descriptions []mismatch) []mismatch {
	n := len(descriptions)
	if n == 0 {
		return NoMismatch
	}

	for _, mClass := range []mismatchClass{sm, missingRequiredBlockClass, unexpectedBlockClass, typeMismatchClass} {
		mismatches := make([]mismatch, 0, 4)
		for _, desc := range descriptions {
			if desc.class() == mClass {
				mismatches = append(mismatches, desc)
			}
		}
		if len(mismatches) == n {
			// If they all have the same canonical path, then we can compact this into one
			prev := mismatches[0]
			for idx := 1; idx < n; idx++ {
				curr := mismatches[idx]
				if pathEquals(prev.canonicalPath(), curr.canonicalPath()) {
					prev = mergeMismatch(prev, curr, prev.path())
				} else {
					prev = nil
					break
				}
			}
			if prev != nil {
				// Report the generic mismatch and skip the rest
				descriptions = []mismatch{prev}
				break
			}
		}
	}
	descriptions = unique(descriptions)
	if len(descriptions) == 1 {
		descriptions = []mismatch{chopPath(descriptions[0], varyingPathPosition)}
	}
	return descriptions
}

func unique(v []mismatch) []mismatch {
	u := make([]mismatch, 0, len(v))
next:
	for _, m := range v {
		for _, x := range u {
			if m == x {
				break next
			}
		}
		u = append(u, m)
	}
	return u
}

func init() {
	px.DescribeSignatures = describeSignatures

	px.DescribeMismatch = func(name string, expected, actual px.Type) string {
		result := describe(expected, actual, []*pathElement{{fmt.Sprintf("function %s:", name), subject}})
		switch len(result) {
		case 0:
			return ``
		case 1:
			return formatMismatch(result[0])
		default:
			rs := make([]string, len(result))
			for i, r := range result {
				rs[i] = formatMismatch(r)
			}
			return strings.Join(rs, "\n")
		}
	}
}

func describeSignatures(signatures []px.Signature, argsTuple px.Type, block px.Lambda) string {
	errorArrays := make([][]mismatch, len(signatures))
	allSet := true

	ne := 0
	for ix, sg := range signatures {
		ae := describeSignatureArguments(sg, argsTuple, []*pathElement{{strconv.Itoa(ix), signature}})
		errorArrays[ix] = ae
		if len(ae) == 0 {
			allSet = false
		} else {
			ne++
		}
	}

	// Skip block checks if all signatures have argument errors
	if !allSet {
		blockArrays := make([][]mismatch, len(signatures))
		bcCount := 0
		for ix, sg := range signatures {
			ae := describeSignatureBlock(sg, block, []*pathElement{{strconv.Itoa(ix), signature}})
			blockArrays[ix] = ae
			if len(ae) > 0 {
				bcCount++
			}
		}
		if bcCount == len(blockArrays) {
			// Skip argument errors when all alternatives have block errors
			errorArrays = blockArrays
		} else if bcCount > 0 {
			// Merge errors giving argument errors precedence over block errors
			for ix, ea := range errorArrays {
				if len(ea) == 0 {
					errorArrays[ix] = blockArrays[ix]
				}
			}
		}
	}
	if len(errorArrays) == 0 {
		return ``
	}

	if ne > 1 {
		// If the argsTuple is of size one and the argument is a Struct, then skip the positional
		// signature since that output just decreases readability.
		isStruct := false
		switch args := argsTuple.(type) {
		case *types.TupleType:
			if len(args.Types()) == 1 {
				_, isStruct = args.Types()[0].(*types.StructType)
			}
		case *types.ArrayType:
			aSize := args.Size()
			if aSize.Max() == 1 {
				_, isStruct = args.ElementType().(*types.StructType)
			}
		}

		if isStruct {
			structArg := -1
			for ix, sg := range signatures {
				ae := errorArrays[ix]
				if len(ae) > 0 {
					paramsTuple := sg.ParametersType().(*types.TupleType)
					tps := paramsTuple.Types()
					if len(tps) >= 1 && paramsTuple.Size().Min() <= 1 {
						if _, ok := tps[0].(*types.StructType); ok {
							if structArg >= 0 {
								// Multiple struct args. Break out
								structArg = -1
								break
							}
							structArg = ix
						}
					}
				}
			}
			if structArg >= 0 {
				// Strip other errors
				errorArrays = errorArrays[structArg : structArg+1]
				signatures = signatures[structArg : structArg+1]
			}
		}
	}

	errors := make([]mismatch, 0)
	for _, ea := range errorArrays {
		errors = append(errors, ea...)
	}

	errors = mergeDescriptions(0, countMismatchClass, errors)
	if len(errors) == 1 {
		return formatMismatch(errors[0])
	}

	var result []string
	if len(signatures) == 1 {
		result = []string{fmt.Sprintf(`expects (%s)`, signatureString(signatures[0]))}
		for _, e := range errorArrays[0] {
			result = append(result, fmt.Sprintf(`  rejected:%s`, formatMismatch(chopPath(e, 0))))
		}
	} else {
		result = []string{`expects one of:`}
		for ix, sg := range signatures {
			result = append(result, fmt.Sprintf(`  (%s)`, signatureString(sg)))
			for _, e := range errorArrays[ix] {
				result = append(result, fmt.Sprintf(`    rejected:%s`, formatMismatch(chopPath(e, 0))))
			}
		}
	}
	return strings.Join(result, "\n")
}

func describeSignatureArguments(signature px.Signature, args px.Type, path []*pathElement) []mismatch {
	paramsTuple := signature.ParametersType().(*types.TupleType)
	eSize := paramsTuple.Size()

	var aSize *types.IntegerType
	var aTypes []px.Type
	switch args := args.(type) {
	case *types.TupleType:
		aSize = args.Size()
		aTypes = args.Types()
	case *types.ArrayType:
		aSize = args.Size()
		n := int(aSize.Min())
		aTypes = make([]px.Type, n)
		for i := 0; i < n; i++ {
			aTypes[i] = args.ElementType()
		}
	}
	if px.IsAssignable(eSize, aSize) {
		eTypes := paramsTuple.Types()
		eLast := len(eTypes) - 1
		eNames := signature.ParameterNames()
		for ax, aType := range aTypes {
			ex := ax
			if ex > eLast {
				ex = eLast
			}
			eType := eTypes[ex]
			if !px.IsAssignable(eType, aType) {
				descriptions := describe(eType, aType, pathWith(path, &pathElement{eNames[ex], _parameter}))
				if len(descriptions) > 0 {
					return descriptions
				}
			}
		}
		return NoMismatch
	}
	return []mismatch{newCountMismatch(path, eSize, aSize)}
}

func describeSignatureBlock(signature px.Signature, aBlock px.Lambda, path []*pathElement) []mismatch {
	eBlock := signature.BlockType()
	if aBlock == nil {
		if eBlock == nil || px.IsAssignable(eBlock, types.DefaultUndefType()) {
			return NoMismatch
		}
		return []mismatch{newMissingRequiredBlock(path)}
	}

	if eBlock == nil {
		return []mismatch{newUnexpectedBlock(path)}
	}
	return describe(eBlock, aBlock.Signature(), pathWith(path, &pathElement{signature.BlockName(), block}))
}

func signatureString(signature px.Signature) string {
	tuple := signature.ParametersType().(*types.TupleType)
	size := tuple.Size()
	if size.Min() == 0 && size.Max() == 0 {
		return ``
	}

	names := signature.ParameterNames()
	ts := tuple.Types()
	limit := len(ts)
	results := make([]string, 0, limit)
	for ix, t := range ts {
		indicator := ``
		if size.Max() == math.MaxInt64 && ix == limit-1 {
			// Last is a repeated_param.
			indicator = `*`
			if size.Min() == int64(len(names)) {
				indicator = `+`
			}
		} else if optional(ix, size.Max()) {
			indicator = `?`
			if ot, ok := t.(*types.OptionalType); ok {
				t = ot.ContainedType()
			}
		}
		results = append(results, fmt.Sprintf(`%s %s%s`, t.String(), names[ix], indicator))
	}

	block := signature.BlockType()
	if block != nil {
		if ob, ok := block.(*types.OptionalType); ok {
			block = ob.ContainedType()
		}
		results = append(results, fmt.Sprintf(`%s %s`, block.String(), signature.BlockName()))
	}
	return strings.Join(results, `, `)
}

func optional(index int, requiredCount int64) bool {
	count := int64(index + 1)
	return count > requiredCount
}
