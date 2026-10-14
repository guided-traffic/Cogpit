package engine

// The built-in agent (internal/agent) keeps its LLM connections and its conversations in Cogpit's
// configuration directory with the same owner-only rules as the engine's own files
// (docs/security/credentials.md): these export the engine's writer and reader for it.

// WritePrivateFile replaces path with data atomically, owner-only: 0600 in a 0700 directory, a
// symlink or other non-regular file at path refused (privatefile.go).
func WritePrivateFile(path string, data []byte) error { return writePrivateFile(path, data) }

// ReadPrivateJSON decodes the JSON file at path into v. A missing file leaves v alone and is not
// an error; a file larger than maxSize, a symlink or one that does not decode is.
func ReadPrivateJSON(path string, maxSize int64, v any) error {
	return readPrivateJSON(path, maxSize, v)
}
