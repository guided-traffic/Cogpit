package yamlview

// The tokenizer highlights one line at a time. YAML is not line-oriented everywhere: a block scalar
// (`key: |`) and a quoted scalar that wraps across lines change how the following lines read. That
// context is a lineState, which the document keeps for a prefix of its lines, so a visible line is
// tokenized from its own state without looking at its neighbours.
//
// The tokenizer is a highlighter, not a parser: it never fails, it accepts invalid YAML and colours
// it as best it can, and it gives up on lines longer than maxHighlightLen.

import "strings"

// tokKind classifies a highlighted span.
type tokKind uint8

const (
	tokPlain    tokKind = iota // plain scalars and anything unclassified
	tokKey                     // mapping keys
	tokString                  // quoted scalars and block scalar content
	tokNumber                  // integers, floats, .inf and .nan
	tokLiteral                 // booleans and null
	tokComment                 // comments
	tokPunct                   // indicators: - ? : , [ ] { } and block scalar headers
	tokAnchor                  // anchors, aliases and tags
	tokDocument                // document markers and directives
	numTokKinds
)

// token is a highlighted span of a line, in bytes.
type token struct {
	start, end int
	kind       tokKind
}

// maxHighlightLen is the longest line, in bytes, that is highlighted. Longer lines are shown plain,
// so that one pathological line cannot make a keystroke slow.
const maxHighlightLen = 4096

// lineState is the tokenizer's context at the start of a line. The low two bits say which context;
// for a block scalar the rest holds the parent indentation plus one.
type lineState uint32

const (
	ctxNormal = iota // ordinary YAML
	ctxBlock         // inside a block scalar
	ctxSingle        // inside a single-quoted scalar that started on an earlier line
	ctxDouble        // inside a double-quoted scalar that started on an earlier line
)

func (s lineState) ctx() int { return int(s & 3) }

// parentIndent is the indentation a block scalar's content must exceed.
func (s lineState) parentIndent() int { return int(s>>2) - 1 }

func blockState(parentIndent int) lineState {
	return lineState(ctxBlock) | lineState(max(parentIndent+1, 0))<<2
}

func quotedState(q byte) lineState {
	if q == '"' {
		return ctxDouble
	}
	return ctxSingle
}

// tokenize returns the tokens of line, read in context st, and the context of the next line.
// Tokens are sorted, do not overlap and lie within the line.
func tokenize(line string, st lineState) ([]token, lineState) {
	return tokenizeAs(line, st, false)
}

// tokenizeAs is tokenize that, with plainStrings, colours a plain scalar that is no boolean, null
// or number as a string (View.SetPlainStrings).
func tokenizeAs(line string, st lineState, plainStrings bool) ([]token, lineState) {
	l := lexer{s: line, emit: true, plainStrings: plainStrings}
	next := l.run(st)
	return l.toks, next
}

// nextState returns the context of the line after line without collecting tokens.
func nextState(line string, st lineState) lineState {
	// Only a quote or a block scalar header leaves the normal context, so most lines need no
	// lexing; this keeps the scan to the end of a 5 MB document short.
	if st == ctxNormal && !strings.ContainsAny(line, `'"|>`) {
		return ctxNormal
	}
	l := lexer{s: line}
	return l.run(st)
}

type lexer struct {
	s    string
	toks []token
	emit bool
	// plainStrings colours plain scalars that are no boolean, null or number as strings.
	plainStrings bool
}

// scalar classifies a plain scalar (classifyPlain), a string when plainStrings asks for it.
func (l *lexer) scalar(v string) tokKind {
	k := classifyPlain(v)
	if k == tokPlain && l.plainStrings {
		return tokString
	}
	return k
}

func (l *lexer) add(start, end int, k tokKind) {
	if l.emit && end > start && k != tokPlain {
		l.toks = append(l.toks, token{start, end, k})
	}
}

func (l *lexer) run(st lineState) lineState {
	s := l.s
	if len(s) > maxHighlightLen {
		// Not highlighted. A block scalar keeps its context across the line; anything else is
		// assumed to end in the normal context.
		if st.ctx() == ctxBlock && indentOf(s) > st.parentIndent() {
			return st
		}
		return ctxNormal
	}
	switch st.ctx() {
	case ctxBlock:
		if !isDocMarker(s) { // a document marker ends every block scalar
			ind := indentOf(s)
			if ind == len(s) || isBlank(s[ind:]) {
				return st // blank lines belong to the scalar
			}
			if ind > st.parentIndent() {
				l.add(ind, len(s), tokString)
				return st
			}
		}
	case ctxSingle, ctxDouble:
		q := byte('\'')
		if st.ctx() == ctxDouble {
			q = '"'
		}
		j, closed := scanQuoted(s, 0, q)
		l.add(indentOf(s), j, tokString)
		if !closed {
			return st
		}
		return l.node(j, indentOf(s)-1)
	}
	if isDocMarker(s) {
		l.add(0, 3, tokDocument)
		return l.node(3, -1)
	}
	if len(s) > 0 && s[0] == '%' {
		l.add(0, len(s), tokDocument)
		return ctxNormal
	}
	return l.node(0, indentOf(s)-1)
}

// node reads block-context content from i to the end of the line. base is the indentation a block
// scalar header found here would take as its parent: the column of the last key or sequence dash.
func (l *lexer) node(i, base int) lineState {
	s := l.s
	for {
		i = skipSpaces(s, i)
		if i >= len(s) {
			return ctxNormal
		}
		c := s[i]
		switch {
		case c == '#' && (i == 0 || isSpace(s[i-1])):
			l.add(i, len(s), tokComment)
			return ctxNormal
		case (c == '-' || c == '?' || c == ':') && (i+1 == len(s) || isSpace(s[i+1])):
			l.add(i, i+1, tokPunct)
			base = i
			i++
		case c == '&' || c == '*' || c == '!':
			j := scanName(s, i+1)
			l.add(i, j, tokAnchor)
			i = j
		case (c == '|' || c == '>') && isBlockHeader(s, i):
			j := i + 1
			for j < len(s) && (s[j] == '+' || s[j] == '-' || (s[j] >= '1' && s[j] <= '9')) {
				j++
			}
			l.add(i, j, tokPunct)
			if j = skipSpaces(s, j); j < len(s) && s[j] == '#' {
				l.add(j, len(s), tokComment)
			}
			return blockState(base)
		case c == '"' || c == '\'':
			j, closed := scanQuoted(s, i+1, c)
			if !closed {
				l.add(i, len(s), tokString)
				return quotedState(c)
			}
			if k := skipSpaces(s, j); k < len(s) && s[k] == ':' && (k+1 == len(s) || isSpace(s[k+1])) {
				l.add(i, j, tokKey)
				l.add(k, k+1, tokPunct)
				base = i
				i = k + 1
			} else {
				l.add(i, j, tokString)
				i = j
			}
		case c == '[' || c == '{':
			var st lineState
			if i, st = l.flow(i); st != ctxNormal {
				return st
			}
		default:
			j := scanPlainBlock(s, i)
			end := trimRightSpace(s, i, j)
			if j < len(s) && s[j] == ':' {
				l.add(i, end, tokKey)
				l.add(j, j+1, tokPunct)
				base = i
				i = j + 1
			} else {
				l.add(i, end, l.scalar(s[i:end]))
				i = max(j, i+1)
			}
		}
	}
}

// flow reads a flow collection starting at the '[' or '{' at i and returns the index after its
// closing bracket. A collection that continues on the next line is read to the end of this line and
// the next line starts in the normal context; that is wrong for the rare multi-line flow collection
// but costs only colours. An unclosed quote inside returns its quoted context.
func (l *lexer) flow(i int) (int, lineState) {
	s := l.s
	depth := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == '[' || c == '{':
			depth++
			l.add(i, i+1, tokPunct)
			i++
		case c == ']' || c == '}':
			depth--
			l.add(i, i+1, tokPunct)
			i++
			if depth <= 0 {
				return i, ctxNormal
			}
		case c == ',':
			l.add(i, i+1, tokPunct)
			i++
		case isSpace(c):
			i++
		case c == '#' && isSpace(s[i-1]): // i > 0: the collection opened before it
			l.add(i, len(s), tokComment)
			return len(s), ctxNormal
		case c == ':' && (i+1 == len(s) || isSpace(s[i+1]) || isFlowIndicator(s[i+1])):
			l.add(i, i+1, tokPunct)
			i++
		case c == '"' || c == '\'':
			j, closed := scanQuoted(s, i+1, c)
			if !closed {
				l.add(i, len(s), tokString)
				return len(s), quotedState(c)
			}
			if k := skipSpaces(s, j); k < len(s) && s[k] == ':' {
				l.add(i, j, tokKey)
			} else {
				l.add(i, j, tokString)
			}
			i = j
		case c == '&' || c == '*' || c == '!':
			j := scanName(s, i+1)
			l.add(i, j, tokAnchor)
			i = j
		default:
			j := scanPlainFlow(s, i)
			end := trimRightSpace(s, i, j)
			if j < len(s) && s[j] == ':' {
				l.add(i, end, tokKey)
			} else {
				l.add(i, end, l.scalar(s[i:end]))
			}
			i = max(j, i+1)
		}
	}
	return len(s), ctxNormal
}

// isBlockHeader reports whether the '|' or '>' at i starts a block scalar header: the indicator,
// optional chomping and indentation indicators, then a space, a comment or the end of the line.
func isBlockHeader(s string, i int) bool {
	j := i + 1
	for j < len(s) && j <= i+2 && (s[j] == '+' || s[j] == '-' || (s[j] >= '1' && s[j] <= '9')) {
		j++
	}
	return j == len(s) || isSpace(s[j])
}

// scanQuoted returns the index after the closing quote q, searching from i, and whether it was
// found. A single-quoted scalar escapes a quote by doubling it, a double-quoted one with a
// backslash.
func scanQuoted(s string, i int, q byte) (int, bool) {
	for i < len(s) {
		switch c := s[i]; {
		case q == '"' && c == '\\':
			i += 2
		case c == q:
			if q == '\'' && i+1 < len(s) && s[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1, true
		default:
			i++
		}
	}
	return len(s), false
}

// scanPlainBlock returns where a plain scalar starting at i ends in block context: at ": ", at a
// colon ending the line, at " #", or at the end of the line.
func scanPlainBlock(s string, i int) int {
	for j := i; j < len(s); j++ {
		switch s[j] {
		case ':':
			if j+1 == len(s) || isSpace(s[j+1]) {
				return j
			}
		case '#':
			if j > i && isSpace(s[j-1]) {
				return j
			}
		}
	}
	return len(s)
}

// scanPlainFlow is scanPlainBlock for flow context, where flow indicators end a scalar too.
func scanPlainFlow(s string, i int) int {
	for j := i; j < len(s); j++ {
		switch c := s[j]; c {
		case ',', '[', ']', '{', '}':
			return j
		case ':':
			if j+1 == len(s) || isSpace(s[j+1]) || isFlowIndicator(s[j+1]) {
				return j
			}
		case '#':
			if j > i && isSpace(s[j-1]) {
				return j
			}
		}
	}
	return len(s)
}

// scanName returns the end of an anchor, alias or tag name starting at i.
func scanName(s string, i int) int {
	for i < len(s) && !isSpace(s[i]) && !isFlowIndicator(s[i]) {
		i++
	}
	return i
}

// classifyPlain colours a plain scalar: YAML 1.1 booleans and null (Kubernetes reads YAML 1.1),
// numbers, or plain text.
func classifyPlain(v string) tokKind {
	switch v {
	case "true", "True", "TRUE", "false", "False", "FALSE",
		"yes", "Yes", "YES", "no", "No", "NO", "on", "On", "ON", "off", "Off", "OFF",
		"null", "Null", "NULL", "~":
		return tokLiteral
	}
	if isNumber(v) {
		return tokNumber
	}
	return tokPlain
}

// isNumber recognises YAML integers (decimal, 0x, 0o, 0b, with '_' separators), floats with an
// optional exponent, and .inf and .nan.
func isNumber(v string) bool {
	s := v
	if s != "" && (s[0] == '+' || s[0] == '-') {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	switch s {
	case ".inf", ".Inf", ".INF":
		return true
	case ".nan", ".NaN", ".NAN":
		return len(s) == len(v)
	}
	if len(s) > 2 && s[0] == '0' {
		switch s[1] {
		case 'x', 'X':
			return allIn(s[2:], "0123456789abcdefABCDEF_")
		case 'o':
			return allIn(s[2:], "01234567_")
		case 'b':
			return allIn(s[2:], "01_")
		}
	}
	if s[0] != '.' && !isDigit(s[0]) {
		return false
	}
	i, digits := 0, 0
	for i < len(s) && (isDigit(s[i]) || s[i] == '_') {
		if isDigit(s[i]) {
			digits++
		}
		i++
	}
	if i < len(s) && s[i] == '.' {
		i++
		for i < len(s) && (isDigit(s[i]) || s[i] == '_') {
			if isDigit(s[i]) {
				digits++
			}
			i++
		}
	}
	if digits == 0 {
		return false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		exp := 0
		for i < len(s) && isDigit(s[i]) {
			exp++
			i++
		}
		if exp == 0 {
			return false
		}
	}
	return i == len(s)
}

func allIn(s, set string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		found := false
		for j := 0; j < len(set); j++ {
			if s[i] == set[j] {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isSpace(c byte) bool { return c == ' ' || c == '\t' }

func isFlowIndicator(c byte) bool {
	return c == ',' || c == '[' || c == ']' || c == '{' || c == '}'
}

// isDocMarker reports whether the line starts with "---" or "..." on its own or followed by a space.
func isDocMarker(s string) bool {
	if len(s) < 3 || (s[:3] != "---" && s[:3] != "...") {
		return false
	}
	return len(s) == 3 || isSpace(s[3])
}

// indentOf counts the leading spaces; YAML indents with spaces only.
func indentOf(s string) int {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	return i
}

func isBlank(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isSpace(s[i]) {
			return false
		}
	}
	return true
}

func skipSpaces(s string, i int) int {
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	return i
}

// trimRightSpace returns the end of s[i:j] without trailing spaces.
func trimRightSpace(s string, i, j int) int {
	for j > i && isSpace(s[j-1]) {
		j--
	}
	return j
}
