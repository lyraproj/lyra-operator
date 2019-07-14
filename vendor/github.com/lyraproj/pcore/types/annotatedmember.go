package types

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/hash"
	"github.com/lyraproj/pcore/px"
)

type annotatedMember struct {
	annotatable
	name      string
	container *objectType
	typ       px.Type
	override  bool
	final     bool
}

func (a *annotatedMember) initialize(c px.Context, memberType, name string, container *objectType, initHash *Hash) {
	a.annotatable.initialize(initHash)
	a.name = name
	a.container = container
	typ := initHash.Get5(keyType, nil)
	if tn, ok := typ.(stringValue); ok {
		a.typ = container.parseAttributeType(c, memberType, name, tn)
	} else {
		// Unchecked because type is guaranteed by earlier type assertion on the hash
		a.typ = typ.(px.Type)
	}
	a.override = boolArg(initHash, keyOverride, false)
	a.final = boolArg(initHash, keyFinal, false)
}

func (a *annotatedMember) Accept(v px.Visitor, g px.Guard) {
	a.typ.Accept(v, g)
	visitAnnotations(a.annotations, v, g)
}

func (a *annotatedMember) Call(c px.Context, receiver px.Value, block px.Lambda, args []px.Value) px.Value {
	// TODO:
	panic("implement me")
}

func (a *annotatedMember) Name() string {
	return a.name
}

func (a *annotatedMember) Container() px.ObjectType {
	return a.container
}

func (a *annotatedMember) Type() px.Type {
	return a.typ
}

func (a *annotatedMember) Override() bool {
	return a.override
}

func (a *annotatedMember) initHash() *hash.StringHash {
	h := a.annotatable.initHash()
	h.Put(keyType, a.typ)
	if a.final {
		h.Put(keyFinal, BooleanTrue)
	}
	if a.override {
		h.Put(keyOverride, BooleanTrue)
	}
	return h
}

func (a *annotatedMember) Final() bool {
	return a.final
}

// Checks if the this _member_ overrides an inherited member, and if so, that this member is declared with
// override = true and that the inherited member accepts to be overridden by this member.
func assertOverride(a px.AnnotatedMember, parentMembers *hash.StringHash) {
	parentMember, _ := parentMembers.Get(a.Name(), nil).(px.AnnotatedMember)
	if parentMember == nil {
		if a.Override() {
			panic(px.Error(px.OverriddenNotFound, issue.H{`label`: a.Label(), `feature_type`: a.FeatureType()}))
		}
	} else {
		assertCanBeOverridden(parentMember, a)
	}
}

func assertCanBeOverridden(a px.AnnotatedMember, member px.AnnotatedMember) {
	if a.FeatureType() != member.FeatureType() {
		panic(px.Error(px.OverrideMemberMismatch, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if a.Final() {
		aa, ok := a.(px.Attribute)
		if !(ok && aa.Kind() == constant && member.(px.Attribute).Kind() == constant) {
			panic(px.Error(px.OverrideOfFinal, issue.H{`member`: member.Label(), `label`: a.Label()}))
		}
	}
	if !member.Override() {
		panic(px.Error(px.OverrideIsMissing, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
	if !px.IsAssignable(a.Type(), member.Type()) {
		panic(px.Error(px.OverrideTypeMismatch, issue.H{`member`: member.Label(), `label`: a.Label()}))
	}
}

// Visit the keys of an annotations map. All keys are known to be types
func visitAnnotations(a *Hash, v px.Visitor, g px.Guard) {
	if a != nil {
		a.EachKey(func(key px.Value) {
			key.(px.Type).Accept(v, g)
		})
	}
}
