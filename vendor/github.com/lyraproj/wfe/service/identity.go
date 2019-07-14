package service

import (
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/wfe/api"
)

type identity struct {
	id        string
	invokable serviceapi.Invokable
}

func (i *identity) associate(c px.Context, internalID, externalID px.Value) {
	i.invokable.Invoke(c, i.id, `associate`, internalID, externalID)
}

func (i *identity) bumpEra(c px.Context) {
	i.invokable.Invoke(c, i.id, `bumpEra`)
}

func (i *identity) garbage(c px.Context, prefix string) px.List {
	result := i.invokable.Invoke(c, i.id, `garbage`, types.WrapString(prefix))
	if l, ok := result.(px.List); ok {
		return l
	}
	return nil
}

func (i *identity) getExternal(c px.Context, internalId px.Value, required bool) px.Value {
	result := i.invokable.Invoke(c, i.id, `getExternal`, internalId)
	if id, ok := result.(px.StringValue); ok && id.String() != `` {
		return id
	}
	if required {
		panic(px.Error(api.UnableToDetermineExternalId, issue.H{`id`: internalId}))
	}
	return nil
}

func (i *identity) sweep(c px.Context, prefix string) {
	i.invokable.Invoke(c, i.id, `sweep`, types.WrapString(prefix))
}

func (i *identity) purgeExternal(c px.Context, externalID px.Value) {
	i.invokable.Invoke(c, i.id, `purgeExternal`, externalID)
}

func (i *identity) removeExternal(c px.Context, externalID px.Value) {
	i.invokable.Invoke(c, i.id, `removeExternal`, externalID)
}

/*
func (i *identity) search(c px.Context, prefix string) px.List {
	result := i.invokable.Invoke(c, i.id, `search`, types.WrapString(prefix))
	if l, ok := result.(px.List); ok {
		return l
	}
	return nil
}

func (i *identity) exists(c px.Context, internalId px.Value) bool {
	result := i.invokable.Invoke(c, i.id, `getExternal`, internalId).(px.List)
	return result.At(1).(px.Boolean).Bool()
}

func (i *identity) getInternal(c px.Context, externalID px.Value) (px.Value, bool) {
	result := i.invokable.Invoke(c, i.id, `getInternal`, externalID)
	if id, ok := result.(px.StringValue); ok && id.String() != `` {
		return id, ok
	}
	return nil, false
}

func (i *identity) purgeInternal(c px.Context, internalID px.Value) {
	i.invokable.Invoke(c, i.id, `purgeInternal`, internalID)
}

func (i *identity) removeInternal(c px.Context, internalID px.Value) {
	i.invokable.Invoke(c, i.id, `removeInternal`, internalID)
}
*/

var IdentityId = px.NewTypedName(px.NsDefinition, "Identity::Service")

func getIdentity(c px.Context) *identity {
	d := GetDefinition(c, IdentityId)
	return &identity{d.Identifier().Name(), GetService(c, d.ServiceId())}
}
