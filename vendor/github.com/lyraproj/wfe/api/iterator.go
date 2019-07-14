package api

import (
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/servicesdk/wf"
)

type Iterator interface {
	Step

	// Style returns the style of iterator, times, range, each, or eachPair.
	IterationStyle() wf.IterationStyle

	// Producer returns the Step that will be invoked once for each iteration
	Producer() Step

	// Over returns what this iterator will iterate over
	Over() px.Value

	// Variables returns the variables that this iterator will produce for each iteration. These
	// variables will be removed from the declared parameters set when the final requirements
	// for the step are computed.
	Variables() []px.Parameter
}
