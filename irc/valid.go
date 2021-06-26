package irc

// https://datatracker.ietf.org/doc/html/rfc2812#section-2.3.1
// with slightly extended nickname length
func validNick(n string) bool {
	if len(n) < 1 || len(n) > 12 {
		return false
	}

	validFirst := func(x rune) bool {
		return (x >= 'A' && x <= 'Z') || (x >= 'a' && x <= 'z') || (x >= '[' && x <= '^') || (x >= '{' && x <= '}')
	}

	validDigitUnder := func(x rune) bool {
		return (x >= '0' && x <= '9') || x == '_'
	}

	for i, x := range n {
		if validFirst(x) || (i > 0 && validDigitUnder(x)) {
			continue
		}
		return false
	}

	return true
}

// Drop control characters from rfc2812, allow unicode, etc.
func validChan(c string) bool {
	if len(c) < 1 || len(c) > 64 {
		return false
	}

	// Only # channels for now
	if c[0] != '#' {
		return false
	}

	if len(c) > 1 {
		for _, x := range c[1:] {
			if x <= ' ' || x == ':' || x == ',' || x == '\x7F' {
				return false
			}
		}
	}

	return true
}
