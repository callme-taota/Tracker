package pluginhub

// Deprecated: FormatsCompatible keeps the pre-DAG permissive semantics.
// New code should use EdgeFormatsCompatible and keep this helper behind the
// runtime.legacy_formats_compat feature flag until legacy callers are removed.
func FormatsCompatible(outputFormats, inputFormats []string) bool {
	if len(inputFormats) == 0 {
		return true
	}
	for _, in := range inputFormats {
		if in == "*" {
			return true
		}
	}
	if len(outputFormats) == 0 {
		return true
	}
	for _, o := range outputFormats {
		for _, in := range inputFormats {
			if o == in {
				return true
			}
		}
	}
	return false
}

// EdgeFormatsCompatible enforces format tokens when both ends participate in item flow.
// Empty downstream input_formats means no format constraint. "*" in input accepts any.
// Empty upstream output with a constrained downstream input fails.
func EdgeFormatsCompatible(outputFormats, inputFormats []string) bool {
	if len(inputFormats) == 0 {
		return true
	}
	for _, in := range inputFormats {
		if in == "*" {
			return true
		}
	}
	if len(outputFormats) == 0 {
		return false
	}
	for _, o := range outputFormats {
		for _, in := range inputFormats {
			if o == in {
				return true
			}
		}
	}
	return false
}
