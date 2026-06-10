package headtail

// HeadTailParams is a type that contains the head and tail parameters.
type HeadTailParams struct {
	Head           int  `json:"head,omitempty"`
	Tail           int  `json:"tail,omitempty"`
	ApplyTailFirst bool `json:"apply_tail_first,omitempty"`
}

// Apply applies the head and tail parameters to the lines. If none
// are set, then only head is applied and the default maximum
// number of lines is returned. If both Head and Tail are set, and
// ApplyTailFirst is true, tail will be applied first, otherwise
// head will be applied first. If only one of Head or Tail is set,
// that one will be applied and ApplyTailFirst will be ignored.
func (h *HeadTailParams) Apply(lines []string, defaultMaxLines int) []string {
	// If neither Head nor Tail is set, return the default maximum number of lines.
	if h.Head == 0 && h.Tail == 0 {
		return head(lines, defaultMaxLines)
	}
	// If both Head and Tail are set, apply them in the order specified by ApplyTailFirst.
	if h.Head != 0 && h.Tail != 0 {
		if h.ApplyTailFirst {
			return head(tail(lines, h.Tail), h.Head)
		} else {
			return tail(head(lines, h.Head), h.Tail)
		}
	}
	// If only Head is set, apply it.
	if h.Head != 0 {
		return head(lines, h.Head)
	}
	// If only Tail is set, apply it.
	return tail(lines, h.Tail)
}

// head returns the first n lines of a slice of strings. It will return a new slice of strings
// with the first n lines. If n is less than or equal to 0, or greater than or equal to the
// length of the slice, it will return the entire slice.
func head(lines []string, n int) []string {
	if len(lines) == 0 {
		return lines
	}
	if n <= 0 || n >= len(lines) {
		return lines
	}
	return lines[:n]
}

// tail returns the last n lines of a slice of strings. It will return a new slice of strings
// with the last n lines. If n is less than or equal to 0, or greater than or equal to the
// length of the slice, it will return the entire slice.
func tail(lines []string, n int) []string {
	if len(lines) == 0 {
		return lines
	}
	if n <= 0 || n >= len(lines) {
		return lines
	}
	return lines[len(lines)-n:]
}
