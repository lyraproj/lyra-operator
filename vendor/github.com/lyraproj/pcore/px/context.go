package px

import (
	"context"
	"reflect"
	"runtime"

	"github.com/lyraproj/pcore/threadlocal"

	"github.com/lyraproj/issue/issue"
)

const PuppetContextKey = `puppet.context`

type VariableState int

const NotFound = VariableState(0)
const Global = VariableState(1)
const Local = VariableState(2)

// VariableStates is implemented by an evaluation scope that wishes to differentiate between
// local and global states. The difference is significant for modules like Hiera that relies
// on that global variables remain unchanged during evaluation.
type VariableStates interface {
	// State returns NotFound, Global, or Local for the given name.
	State(name string) VariableState
}

// An Context holds all state during evaluation. Since it contains the stack, each
// thread of execution must use a context of its own. It's expected that multiple
// contexts share common parents for scope and loaders.
//
type Context interface {
	context.Context

	// Delete deletes the given key from the context variable map
	Delete(key string)

	// DefiningLoader returns a Loader that can receive new definitions
	DefiningLoader() DefiningLoader

	// DoWithLoader assigns the given loader to the receiver and calls the doer. The original loader is
	// restored before this call returns.
	DoWithLoader(loader Loader, doer Doer)

	// Error creates a Reported with the given issue code, location, and arguments
	// Typical use is to panic with the returned value
	Error(location issue.Location, issueCode issue.Code, args issue.H) issue.Reported

	// Fail creates a Reported with the EVAL_FAILURE issue code, location from stack top,
	// and the given message
	// Typical use is to panic with the returned value
	Fail(message string) issue.Reported

	// Fork a new context from this context. The fork will have the same scope,
	// loaders, and logger as this context. The stack and the map of context variables will
	// be shallow copied
	Fork() Context

	// Get returns the context variable with the given key together with a bool to indicate
	// if the key was found
	Get(key string) (interface{}, bool)

	// ImplementationRegistry returns the registry that holds mappings between Type and reflect.Type
	ImplementationRegistry() ImplementationRegistry

	// Loader returns the loader of the receiver.
	Loader() Loader

	// Logger returns the logger of the receiver. This will be the same logger as the
	// logger of the evaluator.
	Logger() Logger

	// ParseTypeValue parses and evaluates the given Value into a Type. It will panic with
	// an issue.Reported unless the parsing was successful and the result is evaluates
	// to a Type
	ParseTypeValue(str Value) Type

	// ParseType parses and evaluates the given string into a Type. It will panic with
	// an issue.Reported unless the parsing was successful and the result is evaluates
	// to a Type
	ParseType(str string) Type

	// Reflector returns a Reflector capable of converting to and from reflected values
	// and types
	Reflector() Reflector

	// Scope returns an optional variable scope. This method is intended to be used when
	// the context is used as an evaluation context. The method returns an empty OrderedMap
	// by default.
	Scope() Keyed

	// Set adds or replaces the context variable for the given key with the given value
	Set(key string, value interface{})

	// Permanently change the loader of this context
	SetLoader(loader Loader)

	// Stack returns the full stack. The returned value must not be modified.
	Stack() []issue.Location

	// StackPop pops the last pushed location from the stack
	StackPop()

	// StackPush pushes a location onto the stack. The location is typically the
	// currently evaluated expression.
	StackPush(location issue.Location)

	// StackTop returns the top of the stack
	StackTop() issue.Location
}

// AddTypes Makes the given types known to the loader appointed by the Context
func AddTypes(c Context, types ...Type) {
	l := c.DefiningLoader()
	rts := make([]ResolvableType, 0, len(types))
	tss := make([]TypeSet, 0, 4)
	for _, t := range types {
		// A TypeSet should not be added until it is resolved since it uses its own
		// loader for the resolution.
		if ts, ok := t.(TypeSet); ok {
			tss = append(tss, ts)
		} else {
			l.SetEntry(NewTypedName(NsType, t.Name()), NewLoaderEntry(t, nil))
		}
		if rt, ok := t.(ResolvableType); ok {
			rts = append(rts, rt)
		}
	}
	ResolveTypes(c, rts...)
	for _, ts := range tss {
		l.SetEntry(NewTypedName(NsType, ts.Name()), NewLoaderEntry(ts, nil))
	}
}

// Call calls a function known to the loader of the Context with arguments and an optional block.
func Call(c Context, name string, args []Value, block Lambda) Value {
	tn := NewTypedName2(`function`, name, c.Loader().NameAuthority())
	if f, ok := Load(c, tn); ok {
		return f.(Function).Call(c, block, args...)
	}
	panic(issue.NewReported(UnknownFunction, issue.SeverityError, issue.H{`name`: tn.String()}, c.StackTop()))
}

// DoWithContext sets the given context to be the current context of the executing Go routine, calls
// the actor, and ensures that the old Context is restored once that call ends normally by panic.
func DoWithContext(ctx Context, actor func(Context)) {
	if saveCtx, ok := threadlocal.Get(PuppetContextKey); ok {
		defer func() {
			threadlocal.Set(PuppetContextKey, saveCtx)
		}()
	} else {
		threadlocal.Init()
	}
	threadlocal.Set(PuppetContextKey, ctx)
	actor(ctx)
}

// CurrentContext returns the current runtime context or panics if no such context has been assigned
func CurrentContext() Context {
	if ctx, ok := threadlocal.Get(PuppetContextKey); ok {
		return ctx.(Context)
	}
	panic(issue.NewReported(NoCurrentContext, issue.SeverityError, issue.NoArgs, 0))
}

// Fork calls the given function in a new go routine. The given context is forked and becomes
// the CurrentContext for that routine.
func Fork(c Context, doer ContextDoer) {
	go func() {
		defer threadlocal.Cleanup()
		threadlocal.Init()
		cf := c.Fork()
		threadlocal.Set(PuppetContextKey, cf)
		doer(cf)
	}()
}

// Go calls the given function in a new go routine. The CurrentContext is forked and becomes
// the CurrentContext for that routine.
func Go(f ContextDoer) {
	Fork(CurrentContext(), f)
}

// StackTop returns the top of the stack contained in the current context or a location determined
// as the Go function that was the caller of the caller of this function, i.e. runtime.Caller(2).
func StackTop() issue.Location {
	if ctx, ok := threadlocal.Get(PuppetContextKey); ok {
		return ctx.(Context).StackTop()
	}
	_, file, line, _ := runtime.Caller(2)
	return issue.NewLocation(file, line, 0)
}

// ResolveResolvables resolves types, constructions, or functions that has been recently added by
// init() functions
var ResolveResolvables func(c Context)

// Resolve
var ResolveTypes func(c Context, types ...ResolvableType)

// Wrap converts the given value into a Value
var Wrap func(c Context, v interface{}) Value

// WrapReflected converts the given reflect.Value into a Value
var WrapReflected func(c Context, v reflect.Value) Value
