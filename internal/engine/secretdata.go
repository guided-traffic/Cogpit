package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The Secrets drawer (D-13, H-8 option b). The cache holds Secrets as metadata only, so the drawer
// of a Secret fetches its data with one direct GET when it opens: the direct read architecture §5
// allows for data the cache does not hold. The data lives in engine memory only while the drawer
// shows that Secret, and goes when the drawer closes or moves to another object. The drawer model
// lists every key with its value masked, for the GUI and agents alike; secret.reveal hands one
// value to the user's own window and refuses every other origin. Values never reach the drawer
// model, logs, errors, toasts or the audit log.

// SecretReader reads one Secret, data included, with a direct GET. *cluster.Conn implements it;
// tests put a fake in Backend.Actions. The engine reaches it with a type assertion on
// Backend.Writer, as it reaches ObjectWriter.
type SecretReader interface {
	GetSecret(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error)
}

// RevealCommand is the command a masked Secret value's "Show" button runs: GUI-only.
const RevealCommand = "secret.reveal"

// secretsKind is the kind key of core Secrets.
const secretsKind = "secrets"

// Caps on what the drawer keeps of an opened Secret. A Secret holds up to 1 MiB; the drawer lists
// keys and shows one value at a time, so only the beginning of each value is kept.
const (
	secretValueBytes   = 4 << 10 // bytes of one value a reveal shows
	secretValueLines   = 40      // lines of one value a reveal shows
	secretKeysShown    = 100     // keys the drawer lists
	secretFetchTimeout = 15 * time.Second
)

// secretGet is the request the drawer makes: the verb the access reviews are asked for.
var secretGet = Verb{Verb: "get", Resource: secretsKind}

// isSecrets reports whether a profile is the core Secrets kind (not a virtual kind made from
// Secrets, such as Helm's releases).
func isSecrets(p *model.KindProfile) bool {
	return p != nil && p.Group == "" && p.Resource == secretsKind && p.APIObject()
}

// secretData is what the drawer holds of the Secret it shows.
type secretData struct {
	mu      sync.Mutex
	done    bool   // the GET returned, or was not made
	err     string // why there is no data; never a value
	typ     string
	rv      string // the resourceVersion the GET returned
	keys    []secretKey
	more    int // keys beyond secretKeysShown
	dropped bool
}

// secretKey is one key of an opened Secret: its size, and the beginning of its value unless the
// value is binary.
type secretKey struct {
	name   string
	size   int
	binary bool
	value  []byte // at most secretValueBytes and secretValueLines; nil for a binary value
}

// fetchSecret starts the GET of the Secret a drawer opens on; wake tells the drawer it arrived.
// It returns nil for every other kind. The GET is made once per open of the drawer on the Secret:
// a watch event on the Secret's metadata does not repeat it.
func (s *Session) fetchSecret(b *Backend, p *model.KindProfile, ref model.ObjectRef, wake func()) (*secretData, context.CancelFunc) {
	if !isSecrets(p) {
		return nil, func() {}
	}
	sd := &secretData{}
	r, ok := b.Writer().(SecretReader)
	if !ok {
		sd.fail("this cluster connection cannot read Secret values")
		return sd, func() {}
	}
	if b.Access != nil {
		if allowed, known := b.Access.Allowed(ref.Namespace, secretGet); known && !allowed {
			sd.fail(fmt.Sprintf("%v: you may not %s", ErrNotPermitted, describeVerb(secretGet, ref.Namespace)))
			return sd, func() {}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), secretFetchTimeout)
	s.goSafe("secret data", func() {
		defer cancel()
		obj, err := r.GetSecret(ctx, ref.Namespace, ref.Name)
		sd.set(obj, err)
		wake()
	})
	return sd, cancel
}

func (sd *secretData) fail(reason string) {
	sd.mu.Lock()
	sd.done, sd.err = true, reason
	sd.mu.Unlock()
}

// set keeps what the drawer needs of a fetched Secret: its type and, per key, the size and the
// beginning of the value. A drawer that closed meanwhile keeps nothing.
func (sd *secretData) set(obj *unstructured.Unstructured, err error) {
	var (
		keys []secretKey
		more int
		typ  string
		rv   string
	)
	if err == nil && obj != nil {
		typ, rv = model.NestedString(obj.Object, "type"), obj.GetResourceVersion()
		data := model.NestedMap(obj.Object, "data")
		names := make([]string, 0, len(data))
		for k := range data {
			names = append(names, k)
		}
		slices.Sort(names)
		for i, k := range names {
			if i == secretKeysShown {
				more = len(names) - i
				break
			}
			keys = append(keys, newSecretKey(k, model.NestedString(data, k)))
		}
	}
	sd.mu.Lock()
	defer sd.mu.Unlock()
	if sd.dropped {
		wipe(keys)
		return
	}
	sd.done = true
	if err != nil {
		sd.err = "the Secret could not be read: " + err.Error()
		return
	}
	sd.typ, sd.rv, sd.keys, sd.more = typ, rv, keys, more
}

// newSecretKey decodes one value (the API sends base64) and keeps its size and, for text, its
// beginning. A value that does not decode counts as binary.
func newSecretKey(name, encoded string) secretKey {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return secretKey{name: name, size: base64.StdEncoding.DecodedLen(len(encoded)), binary: true}
	}
	k := secretKey{name: name, size: len(raw), binary: !isText(raw)}
	if !k.binary {
		k.value = append([]byte(nil), clip(raw)...)
	}
	wipeBytes(raw)
	return k
}

// isText reports whether a value can be shown as text: valid UTF-8 without control characters
// other than tabs and line breaks.
func isText(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	for _, r := range string(b) {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

// clip returns the beginning of a text value within the caps, cut at a line or character boundary.
func clip(b []byte) []byte {
	if len(b) > secretValueBytes {
		cut := secretValueBytes
		for cut > 0 && !utf8.RuneStart(b[cut]) {
			cut--
		}
		b = b[:cut]
	}
	lines := 0
	for i, c := range b {
		if c == '\n' {
			if lines++; lines == secretValueLines {
				return b[:i]
			}
		}
	}
	return b
}

// drop forgets the data: the drawer closed or moved to another object. A GET still running keeps
// nothing when it returns.
func (sd *secretData) drop() {
	if sd == nil {
		return
	}
	sd.mu.Lock()
	defer sd.mu.Unlock()
	sd.dropped = true
	wipe(sd.keys)
	sd.keys, sd.typ = nil, ""
}

// wipe overwrites the kept values before they are dropped: they would otherwise stay in memory
// until the collector reuses it.
func wipe(keys []secretKey) {
	for _, k := range keys {
		wipeBytes(k.value)
	}
}

func wipeBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// drawerSecret returns the data the drawer holds for ref, or nil when the drawer does not show it.
func (s *Session) drawerSecret(ref model.ObjectRef) *secretData {
	st := s.state
	st.mu.Lock()
	defer st.mu.Unlock()
	if d := st.drawer; d != nil && d.secret != nil && d.ref.Same(ref) {
		return d.secret
	}
	return nil
}

// addSecretData adds what the drawer fetched of a Secret to its model: the Type to Properties, and
// a Data section with each key and its masked value, a "Show" action beside each text value. A
// Secret the drawer does not show gets a note instead.
func (s *Session) addSecretData(p *model.KindProfile, ref model.ObjectRef, obj *unstructured.Unstructured, m *DetailModel) {
	if !isSecrets(p) {
		return
	}
	sd := s.drawerSecret(ref)
	if sd == nil {
		m.Sections = append(m.Sections, model.DetailSection{Title: "Data", Items: []model.DetailItem{
			{Value: "Cogpit reads a Secret's data only while its drawer is open in Cogpit's window."},
		}})
		return
	}
	sd.mu.Lock()
	defer sd.mu.Unlock()
	if sd.typ != "" && len(m.Sections) > 0 && m.Sections[0].Title == model.TitleProperties {
		m.Sections[0].Items = append(m.Sections[0].Items, model.DetailItem{Name: "Type", Value: sd.typ})
	}
	sec := model.DetailSection{Title: "Data"}
	switch {
	case !sd.done:
		sec.Items = []model.DetailItem{{Value: "Loading…"}}
	case sd.err != "":
		sec.Items = []model.DetailItem{{Value: sd.err, Status: model.StatusWarning}}
	case len(sd.keys) == 0:
		sec.Items = []model.DetailItem{{Value: "No data"}}
	}
	for _, k := range sd.keys {
		it := model.DetailItem{Name: k.name}
		if k.binary {
			it.Value = "binary data, " + byteSize(k.size)
		} else {
			it.Value = MaskedValue + " (" + byteSize(k.size) + ")"
			params, _ := json.Marshal(RevealParams{Ref: ref, Key: k.name})
			it.Actions = []model.ItemAction{{Label: "Show", Command: RevealCommand, Params: params}}
		}
		sec.Items = append(sec.Items, it)
	}
	if sd.more > 0 {
		sec.Items = append(sec.Items, model.DetailItem{Value: strconv.Itoa(sd.more) + " more keys are not listed"})
	}
	if sd.done && sd.err == "" && sd.rv != "" && obj.GetResourceVersion() != "" && obj.GetResourceVersion() != sd.rv {
		sec.Items = append(sec.Items, model.DetailItem{
			Value: "The Secret changed since its drawer opened; open it again to read its new data.", Status: model.StatusWarning,
		})
	}
	m.Sections = append(m.Sections, sec)
}

// MaskedValue is what the drawer shows in place of a Secret's value.
const MaskedValue = "••••••••"

// RevealParams are secret.reveal's parameters: the Secret the drawer shows and one of its keys.
type RevealParams struct {
	Ref model.ObjectRef `json:"ref" jsonschema:"the Secret the drawer shows: kind secrets, its namespace and name"`
	Key string          `json:"key" jsonschema:"the key whose value to show"`
}

// SecretValue is one revealed value, for the user's own window only.
type SecretValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Size  int    `json:"size"`          // the whole value's size in bytes
	Cut   bool   `json:"cut,omitempty"` // Value is the beginning only (4 KiB, 40 lines)
}

// ErrNoSecretData refuses a reveal for a Secret whose data the drawer does not hold.
var ErrNoSecretData = errors.New("the Secret's data is held only while its drawer is open")

// SecretKeyHeld reports, without a value, whether the drawer holds a text value for ref and key:
// secret.reveal's Enabled check.
func (s *Session) SecretKeyHeld(ref model.ObjectRef, key string) error {
	sd := s.drawerSecret(ref)
	if sd == nil {
		return ErrNoSecretData
	}
	sd.mu.Lock()
	defer sd.mu.Unlock()
	for _, k := range sd.keys {
		if k.name == key {
			if k.binary {
				return errors.New("binary values are shown by their size only")
			}
			return nil
		}
	}
	if !sd.done {
		return errors.New("the Secret's data is still loading")
	}
	return fmt.Errorf("the Secret has no key %q", key)
}

// SecretValue returns one value of the Secret the drawer shows, as the drawer fetched it. The
// caller (secret.reveal) hands it to the GUI only.
func (s *Session) SecretValue(ref model.ObjectRef, key string) (SecretValue, error) {
	if err := s.SecretKeyHeld(ref, key); err != nil {
		return SecretValue{}, err
	}
	sd := s.drawerSecret(ref)
	if sd == nil {
		return SecretValue{}, ErrNoSecretData
	}
	sd.mu.Lock()
	defer sd.mu.Unlock()
	for _, k := range sd.keys {
		if k.name == key && !k.binary {
			return SecretValue{Key: key, Value: string(k.value), Size: k.size, Cut: len(k.value) < k.size}, nil
		}
	}
	return SecretValue{}, ErrNoSecretData
}

// byteSize renders a size: "1 byte", "24 bytes", "4.0 KiB", "1.5 MiB".
func byteSize(n int) string {
	switch {
	case n == 1:
		return "1 byte"
	case n < 1<<10:
		return strconv.Itoa(n) + " bytes"
	case n < 1<<20:
		return strconv.FormatFloat(float64(n)/(1<<10), 'f', 1, 64) + " KiB"
	}
	return strconv.FormatFloat(float64(n)/(1<<20), 'f', 1, 64) + " MiB"
}
