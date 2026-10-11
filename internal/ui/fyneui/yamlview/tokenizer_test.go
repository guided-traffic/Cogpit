package yamlview

import (
	"fmt"
	"strings"
	"testing"
)

// spans renders tokens as "kind:text" for readable table tests.
func spans(line string, toks []token) []string {
	names := [numTokKinds]string{"plain", "key", "str", "num", "lit", "com", "pun", "anc", "doc"}
	out := make([]string, 0, len(toks))
	for _, t := range toks {
		out = append(out, names[t.kind]+":"+line[t.start:t.end])
	}
	return out
}

func TestTokenizeLine(t *testing.T) {
	cases := []struct {
		line string
		want string
	}{
		{"apiVersion: apps/v1", "key:apiVersion pun::"},
		{"  replicas: 3", "key:replicas pun:: num:3"},
		{"  ratio: -2.5e3", "key:ratio pun:: num:-2.5e3"},
		{"  hex: 0x1F", "key:hex pun:: num:0x1F"},
		{"  inf: .inf", "key:inf pun:: num:.inf"},
		{"  version: 1.2.3", "key:version pun::"},
		{"  cpu: 100m", "key:cpu pun::"},
		{"  enabled: true", "key:enabled pun:: lit:true"},
		{"  enabled: yes # YAML 1.1", "key:enabled pun:: lit:yes com:# YAML 1.1"},
		{"  value: ~", "key:value pun:: lit:~"},
		{"  value: null", "key:value pun:: lit:null"},
		{`  name: "web: 1"`, `key:name pun:: str:"web: 1"`},
		{`  msg: 'it''s # fine'`, `key:msg pun:: str:'it''s # fine'`},
		{`  esc: "a \" b"`, `key:esc pun:: str:"a \" b"`},
		{`  "quoted key": v`, `key:"quoted key" pun::`},
		{"  image: nginx:1.25", "key:image pun::"},
		{"  url: http://example.com/#frag", "key:url pun::"},
		{"  - name: web", "pun:- key:name pun::"},
		{"- - a", "pun:- pun:-"},
		{"  -1", "num:-1"},
		{"? complex", "pun:?"},
		{"# a comment: with colon", "com:# a comment: with colon"},
		{"key: value # trailing", "key:key pun:: com:# trailing"},
		{"key: a#b", "key:key pun::"},
		{"anchor: &base", "key:anchor pun:: anc:&base"},
		{"alias: *base", "key:alias pun:: anc:*base"},
		{"<<: *base", "key:<< pun:: anc:*base"},
		{"ratio: !!float 0.75", "key:ratio pun:: anc:!!float num:0.75"},
		{"---", "doc:---"},
		{"--- # first", "doc:--- com:# first"},
		{"...", "doc:..."},
		{"%YAML 1.2", "doc:%YAML 1.2"},
		{"----", ""},
		{"args: [\"--port=8080\", x, 3]", `key:args pun:: pun:[ str:"--port=8080" pun:, pun:, num:3 pun:]`},
		{"limits: {cpu: 500m, memory: 1}", "key:limits pun:: pun:{ key:cpu pun:: pun:, key:memory pun:: num:1 pun:}"},
		{"empty: []", "key:empty pun:: pun:[ pun:]"},
		{"script: |", "key:script pun:: pun:|"},
		{"script: >-", "key:script pun:: pun:>-"},
		{"script: |2+ # keep", "key:script pun:: pun:|2+ com:# keep"},
		{"notheader: |x", "key:notheader pun::"},
		{"", ""},
		{"   ", ""},
	}
	for _, tc := range cases {
		toks, st := tokenize(tc.line, 0)
		got := strings.Join(spans(tc.line, toks), " ")
		if got != tc.want {
			t.Errorf("tokenize(%q)\n got %s\nwant %s", tc.line, got, tc.want)
		}
		if strings.HasSuffix(strings.TrimSpace(strings.SplitN(tc.line, " #", 2)[0]), "|") && st.ctx() != ctxBlock {
			t.Errorf("tokenize(%q): state %v, want a block scalar", tc.line, st)
		}
	}
}

// TestTokenizeContext walks multi-line documents and checks the colours that depend on earlier
// lines: block scalars, their end, and quoted scalars that wrap.
func TestTokenizeContext(t *testing.T) {
	doc := []struct {
		line string
		want string
	}{
		{"data:", "key:data pun::"},
		{"  script: |", "key:script pun:: pun:|"},
		{"    echo a: b # not a comment", "str:echo a: b # not a comment"},
		{"", ""},
		{"      indented more", "str:indented more"},
		{"  other: 1", "key:other pun:: num:1"},
		{"list:", "key:list pun::"},
		{"- |", "pun:- pun:|"},
		{"  content", "str:content"},
		{"- key: >", "pun:- key:key pun:: pun:>"},
		{"    folded", "str:folded"},
		{"  sibling: x", "key:sibling pun::"},
		{"msg: 'first line", "key:msg pun:: str:'first line"},
		{"  still: quoted'", "str:still: quoted'"},
		{"after: 2", "key:after pun:: num:2"},
		{`dq: "open \`, `key:dq pun:: str:"open \`},
		{`  closed" # c`, `str:closed" com:# c`},
		{"--- |", "doc:--- pun:|"},
		{"top level text", "str:top level text"},
		{"---", "doc:---"},
		{"k: v", "key:k pun::"},
	}
	var st lineState
	for i, tc := range doc {
		toks, next := tokenize(tc.line, st)
		got := strings.Join(spans(tc.line, toks), " ")
		if got != tc.want {
			t.Errorf("line %d %q (state %d)\n got %s\nwant %s", i, tc.line, st, got, tc.want)
		}
		if ns := nextState(tc.line, st); ns != next {
			t.Errorf("line %d: nextState %d, tokenize %d", i, ns, next)
		}
		st = next
	}
}

func TestTokenizeLongLineIsPlain(t *testing.T) {
	long := "key: " + strings.Repeat("x", maxHighlightLen)
	if toks, st := tokenize(long, 0); len(toks) != 0 || st != ctxNormal {
		t.Errorf("long line: %d tokens, state %d", len(toks), st)
	}
	block := blockState(0)
	if st := nextState("  "+strings.Repeat("y", maxHighlightLen), block); st != block {
		t.Errorf("long block scalar line ended the scalar: %d", st)
	}
}

func TestIsNumber(t *testing.T) {
	yes := []string{"0", "42", "-7", "+3", "1_000", "3.14", ".5", "5.", "1e3", "1.5E-2", "0x1f", "0o17", "0b101", ".inf", "-.Inf", ".nan"}
	no := []string{"", "-", ".", "e3", "1e", "1.2.3", "0x", "0xg", "_1", "-.nan", "100m", "1Gi", "v1", "12:30"}
	for _, s := range yes {
		if !isNumber(s) {
			t.Errorf("isNumber(%q) = false", s)
		}
	}
	for _, s := range no {
		if isNumber(s) {
			t.Errorf("isNumber(%q) = true", s)
		}
	}
}

// checkTokens is the invariant every tokenizer result keeps.
func checkTokens(line string, toks []token) error {
	prev := 0
	for i, tk := range toks {
		if tk.start < prev || tk.end <= tk.start || tk.end > len(line) || tk.kind >= numTokKinds || tk.kind == tokPlain {
			return fmt.Errorf("token %d %+v invalid after %d in %q", i, tk, prev, line)
		}
		prev = tk.end
	}
	return nil
}

func FuzzTokenize(f *testing.F) {
	for _, line := range strings.Split(sampleYAML, "\n") {
		f.Add(line, uint32(0))
	}
	f.Add("  content", uint32(blockState(0)))
	f.Add("x' y", uint32(ctxSingle))
	f.Add(`a\" b" c`, uint32(ctxDouble))
	f.Add("{[a: 'b', \"c\": {d: e}]} # z", uint32(0))
	f.Fuzz(func(t *testing.T, line string, raw uint32) {
		st := lineState(raw)
		toks, next := tokenize(line, st)
		if err := checkTokens(line, toks); err != nil {
			t.Fatal(err)
		}
		if ns := nextState(line, st); ns != next {
			t.Fatalf("nextState %d != tokenize %d for %q", ns, next, line)
		}
	})
}
