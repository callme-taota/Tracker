package plugin

// Type identifies the kind of plugin (source, processor, summary, interest, dispatch).
type Type string

const (
	TypeSource    Type = "source"
	TypeProcessor Type = "processor"
	TypeSummary   Type = "summary"
	TypeInterest  Type = "interest"
	TypeDispatch  Type = "dispatch"
	// TypeOperator is for system-level plugins (e.g. LLM operator API); not executable as a pipeline node.
	TypeOperator Type = "operator"
)
