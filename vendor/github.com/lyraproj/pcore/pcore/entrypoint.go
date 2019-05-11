package pcore

import (
	"context"

	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/pximpl"
)

// DefineSetting defines a new setting with a given valueType and default
// value.
func DefineSetting(key string, valueType px.Type, dflt px.Value) {
	pximpl.InitializeRuntime().DefineSetting(key, valueType, dflt)
}

// Do executes a given function with an initialized Context instance.
//
// The Context will be parented by the Go context returned by context.Background()
func Do(f func(c px.Context)) {
	pximpl.InitializeRuntime().Do(f)
}

// DoWithParent executes a given function with an initialized Context instance.
//
// The context will be parented by the given Go context
func DoWithParent(parentCtx context.Context, actor func(px.Context)) {
	pximpl.InitializeRuntime().DoWithParent(parentCtx, actor)
}

// EnvironmentLoader returns the loader that finds things declared
// in the environment and its modules. This loader is parented
// by the SystemLoader
func EnvironmentLoader() px.Loader {
	return pximpl.InitializeRuntime().EnvironmentLoader()
}

// Get returns a setting or calls the given defaultProducer
// function if the setting does not exist
func Get(key string, defaultProducer px.Producer) px.Value {
	return pximpl.InitializeRuntime().Get(key, defaultProducer)
}

// Loader returns a loader for module.
func Loader(key string) px.Loader {
	return pximpl.InitializeRuntime().Loader(key)
}

func NewContext(loader px.Loader, logger px.Logger) px.Context {
	return pximpl.NewContext(loader, logger)
}

// Logger returns the logger that this instance was created with
func Logger() px.Logger {
	return pximpl.InitializeRuntime().Logger()
}

// Reset clears all settings and loaders, except the static loader
func Reset() {
	pximpl.InitializeRuntime().Reset()
}

// RootContext returns a new Context that is parented by the context.Background()
// and is initialized with a loader that is parented by the EnvironmentLoader.
func RootContext() px.Context {
	return pximpl.InitializeRuntime().RootContext()
}

// Set changes a setting
func Set(key string, value px.Value) {
	pximpl.InitializeRuntime().Set(key, value)
}

// SetLogger changes the logger
func SetLogger(logger px.Logger) {
	pximpl.InitializeRuntime().SetLogger(logger)
}

// SystemLoader returns the loader that finds all built-ins. It's parented
// by a static loader.
func SystemLoader() px.Loader {
	return pximpl.InitializeRuntime().SystemLoader()
}

// Try executes a given function with an initialized Context instance. If an error occurs,
// it is caught and returned. The error returned from the given function is returned when
// no other error is caught.
//
// The Context will be parented by the Go context returned by context.Background()
func Try(actor func(px.Context) error) (err error) {
	return pximpl.InitializeRuntime().Try(actor)
}

// TryWithParent executes a given function with an initialized Context instance.  If an error occurs,
// it is caught and returned. The error returned from the given function is returned when no other
// error is caught
//
// The context will be parented by the given Go context
func TryWithParent(parentCtx context.Context, actor func(px.Context) error) (err error) {
	return pximpl.InitializeRuntime().TryWithParent(parentCtx, actor)
}

func WithParent(parent context.Context, loader px.Loader, logger px.Logger, ir px.ImplementationRegistry) px.Context {
	return pximpl.WithParent(parent, loader, logger, ir)
}
