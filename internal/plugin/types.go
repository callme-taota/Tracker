package plugin

// Type identifies the kind of plugin (source, processor, summary, interest, dispatch).
type Type string

const (
	TypeSource    Type = "source"
	TypeProcessor Type = "processor"
	TypeSummary   Type = "summary"
	TypeInterest  Type = "interest"
	TypeDispatch  Type = "dispatch"
)
