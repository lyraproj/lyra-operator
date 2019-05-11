package pximpl

import (
	"reflect"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type implRegistry struct {
	reflectToObjectType map[string]px.Type
	objectTypeToReflect map[string]reflect.Type
}

type parentedImplRegistry struct {
	px.ImplementationRegistry
	implRegistry
}

func newImplementationRegistry() px.ImplementationRegistry {
	return &implRegistry{make(map[string]px.Type, 7), make(map[string]reflect.Type, 7)}
}

func newParentedImplementationRegistry(parent px.ImplementationRegistry) px.ImplementationRegistry {
	return &parentedImplRegistry{parent, implRegistry{make(map[string]px.Type, 7), make(map[string]reflect.Type, 7)}}
}

func (ir *implRegistry) RegisterType(t px.Type, r reflect.Type) {
	r = types.NormalizeType(r)
	r = assertUnregistered(ir, t, r)
	ir.addTypeMapping(t, r)
}

func (ir *implRegistry) TypeToReflected(t px.Type) (reflect.Type, bool) {
	rt, ok := ir.objectTypeToReflect[t.Name()]
	return rt, ok
}

func (ir *implRegistry) ReflectedNameToType(tn string) (px.Type, bool) {
	pt, ok := ir.reflectToObjectType[tn]
	return pt, ok
}

func (ir *implRegistry) ReflectedToType(t reflect.Type) (px.Type, bool) {
	return ir.ReflectedNameToType(types.NormalizeType(t).String())
}

func (ir *implRegistry) addTypeMapping(t px.Type, r reflect.Type) {
	ir.objectTypeToReflect[t.Name()] = r
	ir.reflectToObjectType[r.String()] = t
}

func (pr *parentedImplRegistry) RegisterType(t px.Type, r reflect.Type) {
	r = types.NormalizeType(r)
	r = assertUnregistered(pr, t, r)
	pr.addTypeMapping(t, r)
}

func (pr *parentedImplRegistry) TypeToReflected(t px.Type) (reflect.Type, bool) {
	rt, ok := pr.ImplementationRegistry.TypeToReflected(t)
	if !ok {
		rt, ok = pr.implRegistry.TypeToReflected(t)
	}
	return rt, ok
}

func (pr *parentedImplRegistry) ReflectedNameToType(tn string) (px.Type, bool) {
	pt, ok := pr.ImplementationRegistry.ReflectedNameToType(tn)
	if !ok {
		pt, ok = pr.implRegistry.ReflectedNameToType(tn)
	}
	return pt, ok
}

func (pr *parentedImplRegistry) ReflectedToType(t reflect.Type) (px.Type, bool) {
	return pr.ReflectedNameToType(types.NormalizeType(t).String())
}

func assertUnregistered(ir px.ImplementationRegistry, t px.Type, r reflect.Type) reflect.Type {
	if rt, ok := ir.TypeToReflected(t); ok {
		if r.String() != rt.String() {
			panic(px.Error(px.ImplAlreadyRegistered, issue.H{`type`: t}))
		}
	}
	if tn, ok := ir.ReflectedToType(r); ok {
		if tn != t {
			panic(px.Error(px.ImplAlreadyRegistered, issue.H{`type`: r.String()}))
		}
	}
	return r
}
