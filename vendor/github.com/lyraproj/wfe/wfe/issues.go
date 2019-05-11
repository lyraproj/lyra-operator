package wfe

import "github.com/lyraproj/issue/issue"

const (
	GraphDotMarshal                = `WF_GRAPH_DOT_MARSHAL`
	AlreadyDefined                 = `WF_ALREADY_DEFINED`
	MultipleProducersOfValue       = `WF_MULTIPLE_PRODUCERS_OF_VALUE`
	NoProducerOfValue              = `WF_NO_PRODUCER_OF_VALUE`
	IterationStepWrongParameters   = `WF_ITERATION_STEP_WRONG_PARAMETERS`
	IterationStepWrongReturns      = `WF_ITERATION_STEP_WRONG_OUTPUT`
	IterationParameterInvalidCount = `WF_ITERATION_PARAMETER_INVALID_COUNT`
	IterationParameterWrongType    = `WF_ITERATION_PARAMETER_WRONG_TYPE`
	IterationVariableInvalidCount  = `WF_ITERATION_VARIABLE_INVALID_COUNT`
	ParameterUnresolved            = `WF_PARAMETER_UNRESOLVED`
	TooManyGuards                  = `WF_TOO_MANY_GUARDS`
)

func init() {
	issue.Hard(GraphDotMarshal, `error while marshalling graph to dot: %{detail}`)
	issue.Hard2(AlreadyDefined, `%{step} is already defined`, issue.HF{`step`: issue.Label})
	issue.Hard2(MultipleProducersOfValue, `both %{step1} and %{step2} returns the value '%{value}'`, issue.HF{`step1`: issue.Label, `step2`: issue.Label})
	issue.Hard2(NoProducerOfValue, `%{step} value '%{value}' is never produced`, issue.HF{`step`: issue.Label})
	issue.Hard2(IterationStepWrongParameters, `%{iterator} parameters must consume returns produced by the iterator`, issue.HF{`iterator`: issue.Label})
	issue.Hard2(IterationStepWrongReturns, `%{iterator} returns must consist of a 'key' and a 'value'`, issue.HF{`iterator`: issue.Label})
	issue.Hard2(IterationParameterInvalidCount, `%{iterator} wrong number of parameters. Expected %{expected}, actual %{actual}`, issue.HF{`iterator`: issue.Label})
	issue.Hard2(IterationParameterInvalidCount, `%{iterator} wrong number of parameters. Expected %{expected}, actual %{actual}`, issue.HF{`iterator`: issue.Label})
	issue.Hard2(IterationParameterWrongType, `%{iterator} parameter %{parameter} if of wrong type. Expected %{expected}, actual %{actual}`, issue.HF{`iterator`: issue.Label})
	issue.Hard2(IterationVariableInvalidCount, `%{iterator}, wrong number of variables. Expected %{expected}, actual %{actual}`, issue.HF{`iterator`: issue.Label})
	issue.Hard2(TooManyGuards, `%{step} is too complex. Expected %{max} guards maximum. Have %{actual}`, issue.HF{`step`: issue.Label})
	issue.Hard2(ParameterUnresolved, `%{step}, parameter %{parameter} cannot be resolved`, issue.HF{`step`: issue.Label})
}
