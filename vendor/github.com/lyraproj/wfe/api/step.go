package api

import (
	"net/url"

	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/servicesdk/wf"
)

// An Step of a Workflow. The workflow is an Step in itself and can be used in
// another Workflow.
type Step interface {
	issue.Labeled

	// When returns an optional Condition that controls whether or not this step participates
	// in the workflow.
	When() wf.Condition

	// Identifier returns a string that uniquely identifies the step within a resource. The string
	// is guaranteed to remain stable across invocations provided that no step names, resource types
	// or iterator parameters changes within the parent chain of this Step.
	Identifier() string

	// IdParams returns optional URL parameter values that becomes part of the Identifier
	IdParams() url.Values

	// The Id of the service that provides this step
	ServiceId() px.TypedName

	// Returns a copy of this Step with index set to the given value
	WithIndex(index int) Step

	// Style returns the step style, 'workflow', 'resource', 'stateHandler', or 'action'.
	Style() string

	// Name returns the fully qualified name of the Step
	Name() string

	// Parameters returns the parameters requirements for the Step
	Parameters() []px.Parameter

	// Returns returns the definition of that this Step will produce
	Returns() []px.Parameter

	// Run will execute this Step. The given parameters must match the declared Parameters. It will return
	// a value that corresponds to the Returns declaration.
	Run(ctx px.Context, parameters px.OrderedMap) px.OrderedMap
}
