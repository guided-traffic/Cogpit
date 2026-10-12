package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"

	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// Output formats of -o.
const (
	outTable = "table"
	outJSON  = "json"
	outYAML  = "yaml"
)

func validOutput(o string) bool { return o == outTable || o == outJSON || o == outYAML }

// safeText makes text from a kubeconfig or a cluster safe for a terminal: control and format
// characters (escape sequences, bidirectional overrides) and invalid UTF-8 become visible escapes.
// Context names, file names and cell values are untrusted: anyone who can drop a kubeconfig into a
// sync folder or write an object can choose them.
func safeText(s string) string { return kubeconfig.DisplayText(s, 0) }

// writeTable writes a header and rows, aligned, every cell made safe for a terminal.
func writeTable(w io.Writer, header []string, rows [][]string) error {
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	line := func(cells []string) {
		for i, c := range cells {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			// A tab or newline in a value would break the columns; safeText escapes both.
			fmt.Fprint(tw, safeText(c))
		}
		fmt.Fprint(tw, "\n")
	}
	line(header)
	for _, r := range rows {
		line(r)
	}
	return tw.Flush()
}

// writeJSON writes a command's JSON result indented, with a trailing newline.
func writeJSON(w io.Writer, data json.RawMessage) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return err
	}
	buf.WriteByte('\n')
	_, err := w.Write(buf.Bytes())
	return err
}

// writeYAML writes a command's JSON result as YAML.
func writeYAML(w io.Writer, data json.RawMessage) error {
	y, err := yaml.JSONToYAML(data)
	if err != nil {
		return err
	}
	_, err = w.Write(y)
	return err
}

// writeResult writes a command's JSON result as JSON or YAML.
func writeResult(w io.Writer, format string, data json.RawMessage) error {
	if format == outYAML {
		return writeYAML(w, data)
	}
	return writeJSON(w, data)
}
