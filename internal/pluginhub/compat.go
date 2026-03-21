package pluginhub

// FormatsCompatible returns true if upstream outputs can feed downstream inputs.
// Empty lists are treated as permissive. "*" in input accepts any.
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
