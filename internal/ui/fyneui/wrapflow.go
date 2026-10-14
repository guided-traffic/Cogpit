package fyneui

import (
	"strings"

	"fyne.io/fyne/v2"
)

// wrapFlow is wrapLines for prose that may hold one word wider than width, such as an image's
// digest in the node shell's confirmation: that word breaks at the widest prefix that fits
// (wrapHard), and the text after it flows on from its rest, which wrapHard alone would leave on a
// line of its own.
func wrapFlow(text string, width, size float32, style fyne.TextStyle) []string {
	if width <= 0 {
		return wrapLines(text, width, size, style)
	}
	paras := strings.Split(text, "\n")
	for i, para := range paras {
		var words []string
		for _, word := range strings.Fields(para) {
			if fyne.MeasureText(word, size, style).Width > width {
				words = append(words, wrapHard(word, width, size, style)...)
				continue
			}
			words = append(words, word)
		}
		paras[i] = strings.Join(words, " ")
	}
	return wrapLines(strings.Join(paras, "\n"), width, size, style)
}
