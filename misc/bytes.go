package misc

// Given a text and a prefix, returns how much of prefix can be found in the text.
func PrefixLength(fulltext, prefix []byte) int {
	i := 0

	limit := len(prefix)
	if len(fulltext) < len(prefix) {
		limit = len(fulltext)
	}

	for {
		if i >= limit {
			break
		}

		if prefix[i] != fulltext[i] {
			break
		}

		i++
	}

	return i
}
