package mcp

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

// EndpointFileName is the endpoint file's name, next to the token file: the URL of the running
// server, which the stdio bridge (cogpit-cli mcp) needs besides the token, because the port is
// new on every start. It is written owner-only like the token and removed with it.
const EndpointFileName = "mcp-endpoint"

// maxEndpointFile bounds what ReadEndpoint reads of each file: a URL or a token is one short line.
const maxEndpointFile = 4 << 10

// ErrNoEndpoint says that no Cogpit serves MCP over HTTP here: the endpoint or the token file is
// missing (Cogpit is not running, or the user switched its MCP server off). Match it with
// errors.Is.
var ErrNoEndpoint = errors.New("no running Cogpit has written an MCP endpoint")

// Endpoint is a running server's address and bearer token, as its files say.
type Endpoint struct {
	URL   string
	Token string
}

// DefaultTokenPath returns <user config dir>/Cogpit/mcp-token, where the desktop app writes its
// token.
func DefaultTokenPath() (string, error) { return defaultTokenPath() }

// endpointFile is the endpoint file that belongs to a token file.
func endpointFile(tokenPath string) string {
	return filepath.Join(filepath.Dir(tokenPath), EndpointFileName)
}

// ReadEndpoint reads the endpoint and token files of the server whose token file is tokenPath.
// Either file missing is ErrNoEndpoint. It accepts only what Start writes: an http URL on a
// loopback address with the path /mcp, and a token of 32 hex-encoded bytes, each from a regular
// file, so that a file someone else wrote can never send the token anywhere but to this machine.
// The token is not checked against a server: a stale pair (Cogpit crashed) is found by
// connecting.
func ReadEndpoint(tokenPath string) (Endpoint, error) {
	rawURL, err := readLine(endpointFile(tokenPath))
	if err != nil {
		return Endpoint{}, err
	}
	token, err := readLine(tokenPath)
	if err != nil {
		return Endpoint{}, err
	}
	if err := checkEndpointURL(rawURL); err != nil {
		return Endpoint{}, err
	}
	if b, err := hex.DecodeString(token); err != nil || len(b) != tokenBytes {
		return Endpoint{}, fmt.Errorf("mcp: the token file %s does not hold a token Cogpit wrote", tokenPath)
	}
	return Endpoint{URL: rawURL, Token: token}, nil
}

// readLine reads a short regular file and returns its content without surrounding white space.
func readLine(path string) (string, error) {
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("%w (%s is missing)", ErrNoEndpoint, filepath.Base(path))
	}
	if err != nil {
		return "", fmt.Errorf("mcp: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("mcp: %s is not a regular file", path)
	}
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%w (%s is missing)", ErrNoEndpoint, filepath.Base(path))
		}
		return "", fmt.Errorf("mcp: %w", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, maxEndpointFile+1))
	if err != nil {
		return "", fmt.Errorf("mcp: %w", err)
	}
	if len(data) > maxEndpointFile {
		return "", fmt.Errorf("mcp: %s is too large", path)
	}
	return string(bytes.TrimSpace(data)), nil
}

// checkEndpointURL accepts what Start writes: http://<loopback IP>:<port>/mcp.
func checkEndpointURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.Path != endpointPath || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("mcp: the endpoint file does not hold an endpoint Cogpit wrote")
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		return fmt.Errorf("mcp: the endpoint file does not hold an endpoint Cogpit wrote")
	}
	if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("mcp: the endpoint file names a server that is not on this machine; refusing to send the token there")
	}
	if p, err := strconv.ParseUint(port, 10, 16); err != nil || p == 0 {
		return fmt.Errorf("mcp: the endpoint file does not hold an endpoint Cogpit wrote")
	}
	return nil
}
