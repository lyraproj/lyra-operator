package api

import "github.com/lyraproj/issue/issue"

const (
	FailedToLoadPlugin          = `WF_FAILED_TO_LOAD_PLUGIN`
	LyraLinkNoMap               = `WF_LYRA_LINK_NO_MAP`
	LyraLinkNoExe               = `WF_LYRA_LINK_NO_EXE`
	NoSuchAttribute             = `WF_NO_SUCH_ATTRIBUTE`
	NoSuchReferencedValue       = `WF_NO_SUCH_REFERENCED_VALUE`
	NoStepContext               = `WF_NO_STEP_CONTEXT`
	MissingRequiredProperty     = `WF_MISSING_REQUIRED_PROPERTY`
	MultipleErrors              = `WF_MULTIPLE_ERRORS`
	UnableToLoadRequired        = `WF_UNABLE_TO_LOAD_REQUIRED`
	UnableToDetermineExternalId = `WF_UNABLE_TO_DETERMINE_EXTERNAL_ID`
)

func init() {
	issue.Hard(FailedToLoadPlugin, `error while loading plugin executable '%{executable}': %{message}`)
	issue.Hard(LyraLinkNoMap, `Lyra Link did not contain a YAML map`)
	issue.Hard(LyraLinkNoExe, `Lyra Link did not contain a valid 'executable' entry`)
	issue.Hard2(NoSuchReferencedValue, `referenced %{activity} has no %{valueType} named '%{name}'`,
		issue.HF{`activity`: issue.Label})
	issue.Hard2(NoSuchAttribute, `%{step} has no attribute named '%{name}'`, issue.HF{`step`: issue.Label})
	issue.Hard(NoStepContext, `no step context was found in current scope`)
	issue.Hard(MissingRequiredProperty, `definition %{service} %{definition} is missing required property '%{key}'`)
	issue.Hard2(MultipleErrors, `multiple errors: %{errors}`, issue.HF{`errors`: issue.JoinErrors})
	issue.Hard(UnableToLoadRequired, `unable to load required %{namespace} '%{name}'`)
	issue.Hard(UnableToDetermineExternalId, `unable to determine external ID for %{style} '%{id}'`)
}
