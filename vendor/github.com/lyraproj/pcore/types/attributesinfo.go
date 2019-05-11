package types

import "github.com/lyraproj/pcore/px"

type attributesInfo struct {
	nameToPos                map[string]int
	attributes               []px.Attribute
	equalityAttributeIndexes []int
	requiredCount            int
}

func newAttributesInfo(attributes []px.Attribute, requiredCount int, equality []string) *attributesInfo {
	nameToPos := make(map[string]int, len(attributes))
	posToName := make(map[int]string, len(attributes))
	for ix, at := range attributes {
		nameToPos[at.Name()] = ix
		posToName[ix] = at.Name()
	}

	ei := make([]int, len(equality))
	for ix, e := range equality {
		ei[ix] = nameToPos[e]
	}

	return &attributesInfo{attributes: attributes, nameToPos: nameToPos, equalityAttributeIndexes: ei, requiredCount: requiredCount}
}

func (ai *attributesInfo) NameToPos() map[string]int {
	return ai.nameToPos
}

func (ai *attributesInfo) Attributes() []px.Attribute {
	return ai.attributes
}

func (ai *attributesInfo) EqualityAttributeIndex() []int {
	return ai.equalityAttributeIndexes
}

func (ai *attributesInfo) RequiredCount() int {
	return ai.requiredCount
}

func (ai *attributesInfo) PositionalFromHash(hash px.OrderedMap) []px.Value {
	nameToPos := ai.NameToPos()
	va := make([]px.Value, len(nameToPos))

	hash.EachPair(func(k px.Value, v px.Value) {
		if ix, ok := nameToPos[k.String()]; ok {
			va[ix] = v
		}
	})
	attrs := ai.Attributes()
	fillValueSlice(va, attrs)
	for i := len(va) - 1; i >= ai.RequiredCount(); i-- {
		if !attrs[i].Default(va[i]) {
			break
		}
		va = va[:i]
	}
	return va
}
