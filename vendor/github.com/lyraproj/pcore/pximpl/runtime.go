package pximpl

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sync"

	"github.com/lyraproj/pcore/loader"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/threadlocal"
	"github.com/lyraproj/pcore/types"
)

type (
	rt struct {
		lock              sync.RWMutex
		logger            px.Logger
		systemLoader      px.Loader
		environmentLoader px.Loader
		settings          map[string]*setting
	}

	// RuntimeAPI is the interface to the runtime. The runtime should normally not be
	// accessed using the instance. Instead, use the exported functions in the
	// runtime package.
	RuntimeAPI interface {
		// Reset clears all settings and loaders, except the static loaders
		Reset()

		// SystemLoader returns the loader that finds all built-ins. It's parented
		// by a static loader.
		SystemLoader() px.Loader

		// EnvironmentLoader returns the loader that finds things declared
		// in the environment and its modules. This loader is parented
		// by the SystemLoader
		EnvironmentLoader() px.Loader

		// Loader returns a loader for module.
		Loader(moduleName string) px.Loader

		// Logger returns the logger that this instance was created with
		Logger() px.Logger

		// RootContext returns a new Context that is parented by the context.Background()
		// and is initialized with a loader that is parented by the EnvironmentLoader.
		//
		RootContext() px.Context

		// Get returns a setting or calls the given defaultProducer
		// function if the setting does not exist
		Get(key string, defaultProducer px.Producer) px.Value

		// Set changes a setting
		Set(key string, value px.Value)

		// SetLogger changes the logger
		SetLogger(px.Logger)

		// Do executes a given function with an initialized Context instance.
		//
		// The Context will be parented by the Go context returned by context.Background()
		Do(func(px.Context))

		// DoWithParent executes a given function with an initialized Context instance.
		//
		// The context will be parented by the given Go context
		DoWithParent(context.Context, func(px.Context))

		// Try executes a given function with an initialized Context instance. If an error occurs,
		// it is caught and returned. The error returned from the given function is returned when
		// no other error is caught.
		//
		// The Context will be parented by the Go context returned by context.Background()
		Try(func(px.Context) error) error

		// TryWithParent executes a given function with an initialized Context instance.  If an error occurs,
		// it is caught and returned. The error returned from the given function is returned when no other
		// error is caught
		//
		// The context will be parented by the given Go context
		TryWithParent(context.Context, func(px.Context) error) error

		// DefineSetting defines a new setting with a given valueType and default
		// value.
		DefineSetting(key string, valueType px.Type, dflt px.Value)
	}
)

var staticLock sync.Mutex
var pcoreRuntime = &rt{settings: make(map[string]*setting, 32)}
var topImplRegistry px.ImplementationRegistry

func init() {
	pcoreRuntime.DefineSetting(`environment`, types.DefaultStringType(), types.WrapString(`production`))
	pcoreRuntime.DefineSetting(`environmentpath`, types.DefaultStringType(), nil)
	pcoreRuntime.DefineSetting(`module_path`, types.DefaultStringType(), nil)
	pcoreRuntime.DefineSetting(`strict`, types.NewEnumType([]string{`off`, `warning`, `error`}, true), types.WrapString(`warning`))
	pcoreRuntime.DefineSetting(`tasks`, types.DefaultBooleanType(), types.WrapBoolean(false))
	pcoreRuntime.DefineSetting(`workflow`, types.DefaultBooleanType(), types.WrapBoolean(false))
}

func InitializeRuntime() RuntimeAPI {
	// First call initializes the static loader. There can be only one since it receives
	// most of its contents from Go init() functions
	staticLock.Lock()
	defer staticLock.Unlock()

	if pcoreRuntime.logger != nil {
		return pcoreRuntime
	}

	pcoreRuntime.logger = px.NewStdLogger()

	px.RegisterResolvableType(types.NewTypeAliasType(`Pcore::MemberName`, nil, types.TypeMemberName))
	px.RegisterResolvableType(types.NewTypeAliasType(`Pcore::SimpleTypeName`, nil, types.TypeSimpleTypeName))
	px.RegisterResolvableType(types.NewTypeAliasType(`Pcore::typeName`, nil, types.TypeTypeName))
	px.RegisterResolvableType(types.NewTypeAliasType(`Pcore::QRef`, nil, types.TypeQualifiedReference))

	c := NewContext(loader.StaticLoader, pcoreRuntime.logger)
	px.ResolveResolvables(c)
	topImplRegistry = c.ImplementationRegistry()
	return pcoreRuntime
}

func (p *rt) Reset() {
	p.lock.Lock()
	p.systemLoader = nil
	p.environmentLoader = nil
	for _, s := range p.settings {
		s.reset()
	}
	p.lock.Unlock()
}

func (p *rt) SetLogger(logger px.Logger) {
	p.logger = logger
}

func (p *rt) SystemLoader() px.Loader {
	p.lock.Lock()
	p.ensureSystemLoader()
	p.lock.Unlock()
	return p.systemLoader
}

// not exported, provides unprotected access to shared object
func (p *rt) ensureSystemLoader() px.Loader {
	if p.systemLoader == nil {
		p.systemLoader = px.NewParentedLoader(loader.StaticLoader)
	}
	return p.systemLoader
}

func (p *rt) EnvironmentLoader() px.Loader {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.environmentLoader == nil {
		p.ensureSystemLoader()
		envLoader := p.systemLoader // TODO: Add proper environment loader
		s := p.settings[`module_path`]
		if s.isSet() {
			mds := make([]px.ModuleLoader, 0)
			modulesPath := s.get().String()
			fis, err := ioutil.ReadDir(modulesPath)
			if err == nil {
				lds := []px.PathType{px.PuppetFunctionPath, px.PuppetDataTypePath, px.PlanPath, px.TaskPath}
				for _, fi := range fis {
					if fi.IsDir() && px.IsValidModuleName(fi.Name()) {
						ml := px.NewFileBasedLoader(envLoader, filepath.Join(modulesPath, fi.Name()), fi.Name(), lds...)
						mds = append(mds, ml)
					}
				}
			}
			if len(mds) > 0 {
				envLoader = px.NewDependencyLoader(mds)
			}
		}
		p.environmentLoader = envLoader
	}
	return p.environmentLoader
}

func (p *rt) Loader(key string) px.Loader {
	envLoader := p.EnvironmentLoader()
	if key == `` {
		return envLoader
	}
	if dp, ok := envLoader.(px.DependencyLoader); ok {
		return dp.LoaderFor(key)
	}
	return nil
}

func (p *rt) DefineSetting(key string, valueType px.Type, dflt px.Value) {
	s := &setting{name: key, valueType: valueType, defaultValue: dflt}
	if dflt != nil {
		s.set(dflt)
	}
	p.lock.Lock()
	p.settings[key] = s
	p.lock.Unlock()
}

func (p *rt) Get(key string, defaultProducer px.Producer) px.Value {
	p.lock.RLock()
	v, ok := p.settings[key]
	p.lock.RUnlock()

	if ok {
		if v.isSet() {
			return v.get()
		}
		if defaultProducer == nil {
			return px.Undef
		}
		return defaultProducer()
	}
	panic(fmt.Sprintf(`Attempt to access unknown setting '%s'`, key))
}

func (p *rt) Logger() px.Logger {
	return p.logger
}

func (p *rt) RootContext() px.Context {
	InitializeRuntime()
	c := WithParent(context.Background(), p.EnvironmentLoader(), p.logger, topImplRegistry)
	threadlocal.Init()
	threadlocal.Set(px.PuppetContextKey, c)
	px.ResolveResolvables(c)
	return c
}

func (p *rt) Do(actor func(px.Context)) {
	p.DoWithParent(p.RootContext(), actor)
}

func (p *rt) DoWithParent(parentCtx context.Context, actor func(px.Context)) {
	InitializeRuntime()
	if ec, ok := parentCtx.(px.Context); ok {
		ctx := ec.Fork()
		px.DoWithContext(ctx, actor)
	} else {
		ctx := WithParent(parentCtx, p.EnvironmentLoader(), p.logger, topImplRegistry)
		px.DoWithContext(ctx, func(ctx px.Context) {
			px.ResolveResolvables(ctx)
			actor(ctx)
		})
	}
}

func (p *rt) Try(actor func(px.Context) error) (err error) {
	return p.TryWithParent(p.RootContext(), actor)
}

func (p *rt) TryWithParent(parentCtx context.Context, actor func(px.Context) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if ri, ok := r.(error); ok {
				err = ri
			} else {
				panic(r)
			}
		}
	}()
	p.DoWithParent(parentCtx, func(c px.Context) {
		err = actor(c)
	})
	return
}

func (p *rt) Set(key string, value px.Value) {
	p.lock.RLock()
	v, ok := p.settings[key]
	p.lock.RUnlock()

	if ok {
		v.set(value)
		return
	}
	panic(fmt.Sprintf(`Attempt to assign unknown setting '%s'`, key))
}
