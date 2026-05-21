package scanner

func matchGlob(pattern string, value string) bool {
	if pattern == "" {
		return false
	}
	if pattern == value {
		return true
	}
	return matchGlobParts(pattern, value)
}

func matchGlobParts(pattern string, value string) bool {
	for pattern != "" {
		switch pattern[0] {
		case '*':
			for len(pattern) > 0 && pattern[0] == '*' {
				pattern = pattern[1:]
			}
			if pattern == "" {
				return true
			}
			for index := 0; index <= len(value); index++ {
				if matchGlobParts(pattern, value[index:]) {
					return true
				}
			}
			return false
		case '?':
			if value == "" {
				return false
			}
			pattern = pattern[1:]
			value = value[1:]
		default:
			if value == "" || pattern[0] != value[0] {
				return false
			}
			pattern = pattern[1:]
			value = value[1:]
		}
	}
	return value == ""
}
