package agent

import "strings"

// Thinking models (qwen3 and others) write their reasoning before the answer, either in
// reasoning_content (or reasoning) deltas, which the accumulator only notes, or inside the content
// between <think> and </think>. A chat template that opens the block in the prompt leaves only the
// closing tag in the content: everything before it was thinking. thinkFilter removes the thinking
// from the content as it streams, so that it is never shown as the answer, never kept and never
// sent back to the model (D-10: dropped, the lead's choice; a collapsed item for it was the other
// option).

const (
	thinkOpen  = "<think>"
	thinkClose = "</think>"
)

type thinkFilter struct {
	visible strings.Builder
	pending string // a tail that may be the start of a tag
	inThink bool
	opened  bool // a <think> was seen
	closed  bool // a </think> ended the thinking
	thought bool // the model thought (a tag or a reasoning delta)
}

// write takes the next piece of the content.
func (f *thinkFilter) write(s string) {
	buf := f.pending + s
	f.pending = ""
	for buf != "" {
		if f.inThink {
			i := strings.Index(buf, thinkClose)
			if i < 0 {
				f.pending = partialTag(buf, thinkClose)
				return
			}
			f.inThink, f.closed = false, true
			buf = buf[i+len(thinkClose):]
			continue
		}
		open := strings.Index(buf, thinkOpen)
		if cl := strings.Index(buf, thinkClose); cl >= 0 && !f.opened && !f.closed && (open < 0 || cl < open) {
			// The template opened the block: what came before the closing tag was thinking.
			f.visible.Reset()
			f.closed, f.thought = true, true
			buf = buf[cl+len(thinkClose):]
			continue
		}
		if open >= 0 && !f.closed {
			f.visible.WriteString(buf[:open])
			f.inThink, f.opened, f.thought = true, true, true
			buf = buf[open+len(thinkOpen):]
			continue
		}
		tail := ""
		if !f.closed {
			tail = longestPartial(buf)
		}
		f.visible.WriteString(buf[:len(buf)-len(tail)])
		f.pending = tail
		return
	}
}

// flush ends the content: a tail held back as a possible tag is text after all, unless it is
// inside an unclosed thinking block.
func (f *thinkFilter) flush() {
	if !f.inThink {
		f.visible.WriteString(f.pending)
	}
	f.pending = ""
}

// text is the answer so far.
func (f *thinkFilter) text() string {
	return strings.TrimSpace(f.visible.String())
}

// partialTag returns the longest suffix of s that is a proper prefix of tag.
func partialTag(s, tag string) string {
	for n := min(len(tag)-1, len(s)); n > 0; n-- {
		if strings.HasPrefix(tag, s[len(s)-n:]) {
			return s[len(s)-n:]
		}
	}
	return ""
}

// longestPartial is the longer of the partial opening and closing tags at s's end.
func longestPartial(s string) string {
	a, b := partialTag(s, thinkOpen), partialTag(s, thinkClose)
	if len(b) > len(a) {
		return b
	}
	return a
}
