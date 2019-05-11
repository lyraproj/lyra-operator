package pximpl

import (
	"context"
	"fmt"
	"sync"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
)

type (
	pxContext struct {
		context.Context
		loader       px.Loader
		logger       px.Logger
		stack        []issue.Location
		implRegistry px.ImplementationRegistry
		vars         map[string]interface{}
	}

	systemLocation struct{}
)

func (systemLocation) File() string {
	return ``
}

func (systemLocation) Line() int {
	return 0
}

func (systemLocation) Pos() int {
	return 0
}

var resolvableFunctions = make([]px.ResolvableFunction, 0, 16)
var resolvableFunctionsLock sync.Mutex

func init() {
	px.RegisterGoFunction = func(function px.ResolvableFunction) {
		resolvableFunctionsLock.Lock()
		resolvableFunctions = append(resolvableFunctions, function)
		resolvableFunctionsLock.Unlock()
	}

	px.ResolveResolvables = resolveResolvables
	px.ResolveTypes = resolveTypes
}

func NewContext(loader px.Loader, logger px.Logger) px.Context {
	return WithParent(context.Background(), loader, logger, newImplementationRegistry())
}

func WithParent(parent context.Context, loader px.Loader, logger px.Logger, ir px.ImplementationRegistry) px.Context {
	var c *pxContext
	ir = newParentedImplementationRegistry(ir)
	if cp, ok := parent.(pxContext); ok {
		c = cp.clone()
		c.Context = parent
		c.loader = loader
		c.logger = logger
	} else {
		c = &pxContext{Context: parent, loader: loader, logger: logger, stack: make([]issue.Location, 0, 8), implRegistry: ir}
	}
	return c
}

func (c *pxContext) DefiningLoader() px.DefiningLoader {
	l := c.loader
	for {
		if dl, ok := l.(px.DefiningLoader); ok {
			return dl
		}
		if pl, ok := l.(px.ParentedLoader); ok {
			l = pl.Parent()
			continue
		}
		panic(`No defining loader found in context`)
	}
}

func (c *pxContext) Delete(key string) {
	if c.vars != nil {
		delete(c.vars, key)
	}
}

func (c *pxContext) DoWithLoader(loader px.Loader, doer px.Doer) {
	saveLoader := c.loader
	defer func() {
		c.loader = saveLoader
	}()
	c.loader = loader
	doer()
}

func (c *pxContext) Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported {
	if location == nil {
		location = c.StackTop()
	}
	return issue.NewReported(issueCode, issue.SeverityError, args, location)
}

func (c *pxContext) Fork() px.Context {
	s := make([]issue.Location, len(c.stack))
	copy(s, c.stack)
	clone := c.clone()
	clone.loader = px.NewParentedLoader(clone.loader)
	clone.implRegistry = newParentedImplementationRegistry(clone.implRegistry)
	clone.stack = s

	if c.vars != nil {
		cv := make(map[string]interface{}, len(c.vars))
		for k, v := range c.vars {
			cv[k] = v
		}
		clone.vars = cv
	}
	return clone
}

func (c *pxContext) Fail(message string) issue.Reported {
	return c.Error(nil, px.Failure, issue.H{`message`: message})
}

func (c *pxContext) Get(key string) (interface{}, bool) {
	if c.vars != nil {
		if v, ok := c.vars[key]; ok {
			return v, true
		}
	}
	return nil, false
}

func (c *pxContext) ImplementationRegistry() px.ImplementationRegistry {
	return c.implRegistry
}

func (c *pxContext) Loader() px.Loader {
	return c.loader
}

func (c *pxContext) Logger() px.Logger {
	return c.logger
}

func (c *pxContext) ParseTypeValue(typeString px.Value) px.Type {
	if sv, ok := typeString.(px.StringValue); ok {
		return c.ParseType(sv.String())
	}
	panic(px.Error(px.IllegalArgumentType, issue.H{`function`: `ParseTypeValue`, `index`: 0, `expected`: `String`, `actual`: px.DetailedValueType(typeString).String()}))
}

func (c *pxContext) ParseType(str string) px.Type {
	t, err := types.Parse(str)
	if err != nil {
		panic(err)
	}
	if pt, ok := t.(px.ResolvableType); ok {
		return pt.Resolve(c)
	}
	panic(fmt.Errorf(`expression "%s" does no resolve to a Type`, str))
}

func (c *pxContext) Reflector() px.Reflector {
	return types.NewReflector(c)
}

func resolveResolvables(c px.Context) {
	l := c.Loader().(px.DefiningLoader)
	ts := types.PopDeclaredTypes()
	for _, rt := range ts {
		l.SetEntry(px.NewTypedName(px.NsType, rt.Name()), px.NewLoaderEntry(rt, nil))
	}

	for _, mp := range types.PopDeclaredMappings() {
		c.ImplementationRegistry().RegisterType(mp.T, mp.R)
	}

	resolveTypes(c, ts...)

	ctors := types.PopDeclaredConstructors()
	for _, ct := range ctors {
		rf := px.BuildFunction(ct.Name, ct.LocalTypes, ct.Creators)
		l.SetEntry(px.NewTypedName(px.NsConstructor, rf.Name()), px.NewLoaderEntry(rf.Resolve(c), nil))
	}

	fs := popDeclaredGoFunctions()
	for _, rf := range fs {
		l.SetEntry(px.NewTypedName(px.NsFunction, rf.Name()), px.NewLoaderEntry(rf.Resolve(c), nil))
	}
}

func (c *pxContext) Scope() px.Keyed {
	return px.EmptyMap
}

func (c *pxContext) Set(key string, value interface{}) {
	if c.vars == nil {
		c.vars = map[string]interface{}{key: value}
	} else {
		c.vars[key] = value
	}
}

func (c *pxContext) SetLoader(loader px.Loader) {
	c.loader = loader
}

func (c *pxContext) Stack() []issue.Location {
	return c.stack
}

func (c *pxContext) StackPop() {
	c.stack = c.stack[:len(c.stack)-1]
}

func (c *pxContext) StackPush(location issue.Location) {
	c.stack = append(c.stack, location)
}

func (c *pxContext) StackTop() issue.Location {
	s := len(c.stack)
	if s == 0 {
		return &systemLocation{}
	}
	return c.stack[s-1]
}

// clone a new context from this context which is an exact copy except for the parent
// of the clone which is set to the original. It is used internally by Fork
func (c *pxContext) clone() *pxContext {
	clone := &pxContext{}
	*clone = *c
	clone.Context = c
	return clone
}

func resolveTypes(c px.Context, types ...px.ResolvableType) {
	l := c.DefiningLoader()
	typeSets := make([]px.TypeSet, 0)
	allAnnotated := make([]px.Annotatable, 0, len(types))
	for _, rt := range types {
		switch t := rt.Resolve(c).(type) {
		case px.TypeSet:
			typeSets = append(typeSets, t)
		case px.ObjectType:
			if ctor := t.Constructor(c); ctor != nil {
				l.SetEntry(px.NewTypedName(px.NsConstructor, t.Name()), px.NewLoaderEntry(ctor, nil))
			}
			allAnnotated = append(allAnnotated, t)
		case px.Annotatable:
			allAnnotated = append(allAnnotated, t)
		}
	}

	for _, ts := range typeSets {
		allAnnotated = resolveTypeSet(c, l, ts, allAnnotated)
	}

	// Validate type annotations
	for _, a := range allAnnotated {
		a.Annotations(c).EachValue(func(v px.Value) {
			v.(px.Annotation).Validate(c, a)
		})
	}
}

func resolveTypeSet(c px.Context, l px.DefiningLoader, ts px.TypeSet, allAnnotated []px.Annotatable) []px.Annotatable {
	ts.Types().EachValue(func(tv px.Value) {
		t := tv.(px.Type)
		if tsc, ok := t.(px.TypeSet); ok {
			allAnnotated = resolveTypeSet(c, l, tsc, allAnnotated)
		}
		// Types already known to the loader might have been added to a TypeSet. When that
		// happens, we don't want them added again.
		tn := px.NewTypedName(px.NsType, t.Name())
		le := l.LoadEntry(c, tn)
		if le == nil || le.Value() == nil {
			if a, ok := t.(px.Annotatable); ok {
				allAnnotated = append(allAnnotated, a)
			}
			l.SetEntry(tn, px.NewLoaderEntry(t, nil))
			if ot, ok := t.(px.ObjectType); ok {
				if ctor := ot.Constructor(c); ctor != nil {
					l.SetEntry(px.NewTypedName(px.NsConstructor, t.Name()), px.NewLoaderEntry(ctor, nil))
				}
			}
		}
	})
	return allAnnotated
}

func popDeclaredGoFunctions() (fs []px.ResolvableFunction) {
	resolvableFunctionsLock.Lock()
	fs = resolvableFunctions
	if len(fs) > 0 {
		resolvableFunctions = make([]px.ResolvableFunction, 0, 16)
	}
	resolvableFunctionsLock.Unlock()
	return
}
