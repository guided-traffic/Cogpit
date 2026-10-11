// Command cogpit-cli is Cogpit's headless command-line tool. It is pure Go (no cgo) and will serve
// the CLI commands and the MCP stdio bridge.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "cogpit-cli: no commands yet")
	os.Exit(1)
}
