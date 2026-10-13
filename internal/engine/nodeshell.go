package engine

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/terminal"
)

// The node shell (G7, G8; D-37; docs/security/privilege-footprint.md › Node shell): a root shell on
// a node, as Lens opens it. node.shell (commands/nodeshell.go), confirmed by the user for every
// origin, creates a privileged pod in kube-system on the node that shares the host's PID
// namespace; the dock's node shell tab waits for it to run (a watch on that pod) and execs nsenter
// into PID 1's namespaces through the pod shell's terminal path (terminal.StartNodeShell). The pod
// carries activeDeadlineSeconds, so that it ends even if Cogpit dies, and Cogpit deletes it when
// the shell ends, the tab closes, or Cogpit quits. Its image and an image pull secret are a
// cluster setting (Cluster Settings › Node Shell), the image pinned by digest by default.

// The node shell's names and bounds.
const (
	// NodeShellCommand is the id of the node shell's command.
	NodeShellCommand = "node.shell"
	// NodeShellNamespace is where the pod runs: kube-system, as Lens runs it.
	NodeShellNamespace = "kube-system"
	// NodeShellContainer is the pod's one container.
	NodeShellContainer = "shell"
	// NodeShellDeadline is the pod's activeDeadlineSeconds: it ends by then whatever Cogpit does
	// (Cogpit's default, D-37; Lens's is not known).
	NodeShellDeadline = 2 * time.Hour
	// DefaultNodeShellImage is Lens's default image, the tag docker.io/alpine:3.19, pinned to the
	// digest of its multi-arch index as `docker buildx imagetools inspect docker.io/alpine:3.19`
	// read it on 2026-10-13 (Alpine 3.19.9, built 2025-10-08): its BusyBox has nsenter and sleep.
	DefaultNodeShellImage = "docker.io/library/alpine:3.19@sha256:6baf43584bcb78f2e5847d1de515f23499913ac9f12bdf834811a3145eb11ca1"

	// DockNodeShell is a node shell's dock tab.
	DockNodeShell DockType = "nodeShell"

	nodeShellNamePrefix    = "cogpit-node-shell-"
	nodeShellStartTimeout  = 3 * time.Minute  // the pod pulls its image and starts
	nodeShellCreateTimeout = 30 * time.Second // the create request
	nodeShellDeleteTimeout = 30 * time.Second // one delete request
	nodeShellQuitWait      = 10 * time.Second // what Close waits for the deletes
	maxNodeShellImage      = 512
)

// NodeShellLabels mark a node shell's pod as Cogpit's.
var NodeShellLabels = map[string]string{
	"app.kubernetes.io/name":       "cogpit-node-shell",
	"app.kubernetes.io/managed-by": "cogpit",
}

// ErrNodeShellGUIOnly refuses an agent or the CLI setting the node shell's image or pull secret:
// they decide what runs as root on the node at the user's next node shell. Resetting them to the
// defaults is allowed.
var ErrNodeShellGUIOnly = errors.New("the node shell's image and image pull secret are set only by the user, in Cogpit's window " +
	"(Cluster Settings › Node Shell): an agent or the CLI may reset them to the defaults, not set them")

// NodeShellActions are the cluster side of a node shell. *cluster.Conn implements them; tests put
// a fake in Backend.Actions.
type NodeShellActions interface {
	// CreatePod creates the pod as given.
	CreatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error)
	// WaitPodRunning waits, watching the one pod, until its container runs.
	WaitPodRunning(ctx context.Context, namespace, name, container string) error
	// DeletePod deletes the pod; a non-empty uid is a precondition.
	DeletePod(ctx context.Context, namespace, name, uid string) error
}

var _ NodeShellActions = (*cluster.Conn)(nil)

// NodeShellTab is what a node shell tab names (DockTab.NodeShell): the node and the pod.
type NodeShellTab struct {
	Node      string `json:"node"`
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Image     string `json:"image"`
	// Pinned says whether the image names its digest; false: a floating tag.
	Pinned bool `json:"pinned"`
	// DeadlineSeconds is the pod's activeDeadlineSeconds.
	DeadlineSeconds int64 `json:"deadlineSeconds"`
}

// imageDigest is the end of an image reference pinned by digest.
var imageDigest = regexp.MustCompile(`@sha256:[0-9a-f]{64}$`)

// ImagePinned reports whether an image reference names its digest (repository@sha256:...).
func ImagePinned(image string) bool { return imageDigest.MatchString(image) }

// CheckNodeShellImage refuses what cannot be an image reference: empty is the default, else one
// line of at most 512 bytes without spaces, a URL's scheme or control characters. Whether the
// registry has it is the kubelet's to say.
func CheckNodeShellImage(image string) error {
	switch {
	case image == "":
		return nil
	case len(image) > maxNodeShellImage:
		return fmt.Errorf("the node shell's image: at most %d characters", maxNodeShellImage)
	case strings.IndexFunc(image, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0:
		return errors.New("the node shell's image: a reference without spaces, such as docker.io/library/alpine:3.19@sha256:…")
	case strings.Contains(image, "://"):
		return errors.New("the node shell's image: a reference such as docker.io/library/alpine:3.19, not a URL")
	}
	return nil
}

// CheckNodeShellPullSecret refuses a pull secret's name a Secret cannot have; empty is none.
func CheckNodeShellPullSecret(name string) error {
	if name == "" {
		return nil
	}
	if errs := validation.IsDNS1123Subdomain(name); len(errs) > 0 {
		return fmt.Errorf("the node shell's image pull secret %q is not a Secret's name: %s", name, strings.Join(errs, "; "))
	}
	return nil
}

// NodeShellSettings returns a context's node shell image and image pull secret: its cluster
// settings', else DefaultNodeShellImage and none.
func (s *Session) NodeShellSettings(context string) (image, pullSecret string) {
	cs, err := s.ClusterSettingsOf(context)
	if err != nil {
		cs = ClusterSettings{}
	}
	image = cs.NodeShellImage
	if image == "" {
		image = DefaultNodeShellImage
	}
	return image, cs.NodeShellPullSecret
}

// NodeShellPod is the pod of a node shell on node: in kube-system, bound to the node (nodeName),
// tolerating every taint, sharing the host's PID namespace, its one container privileged and
// sleeping until the deadline ends the pod; no service account token, no service links, never
// restarted, a second to stop.
func NodeShellPod(name, node, image, pullSecret string) *corev1.Pod {
	deadline := int64(NodeShellDeadline / time.Second)
	grace := int64(1)
	privileged, noToken, noLinks := true, false, false
	pod := &corev1.Pod{
		TypeMeta:   metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: NodeShellNamespace, Labels: maps.Clone(NodeShellLabels)},
		Spec: corev1.PodSpec{
			NodeName:                      node,
			HostPID:                       true,
			RestartPolicy:                 corev1.RestartPolicyNever,
			ActiveDeadlineSeconds:         &deadline,
			TerminationGracePeriodSeconds: &grace,
			AutomountServiceAccountToken:  &noToken,
			EnableServiceLinks:            &noLinks,
			Tolerations:                   []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
			Containers: []corev1.Container{{
				Name:            NodeShellContainer,
				Image:           image,
				Command:         []string{"sleep", fmt.Sprint(deadline)},
				SecurityContext: &corev1.SecurityContext{Privileged: &privileged},
			}},
		},
	}
	if pullSecret != "" {
		pod.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: pullSecret}}
	}
	return pod
}

// NodeShellDeadlineText is NodeShellDeadline as a person writes it.
var NodeShellDeadlineText = fmt.Sprintf("%d hours", int(NodeShellDeadline/time.Hour))

// NodeShellDetail is the node shell's sentence in the confirmation: that it creates a privileged
// pod with root access to the node, its namespace, node and image, whether the image is pinned,
// the pull secret, and how long the pod lives.
func NodeShellDetail(node, image, pullSecret string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "It creates a privileged pod in namespace %s on node %s with root access to the node: the pod shares "+
		"the node's processes and the shell enters the node's namespaces as root. ", NodeShellNamespace, node)
	if ImagePinned(image) { // the digest on its own, so that a dialog can break the line before it
		at := strings.LastIndex(image, "@")
		fmt.Fprintf(&b, "Image: %s, pinned by digest %s", image[:at], image[at+1:])
	} else {
		fmt.Fprintf(&b, "Image: %s, a floating tag, not pinned by digest: what it runs can change without Cogpit knowing", image)
	}
	if pullSecret != "" {
		fmt.Fprintf(&b, ", pulled with the image pull secret %s", pullSecret)
	}
	fmt.Fprintf(&b, ". The pod is deleted when the shell closes and ends by itself after %s.", NodeShellDeadlineText)
	return b.String()
}

// nodeShellName returns a new pod name: the prefix and eight random characters.
func nodeShellName() string {
	return nodeShellNamePrefix + strings.ToLower(rand.Text()[:8])
}

// StartNodeShell creates the node shell's pod on the node ref names and opens its dock tab. The
// caller (node.shell) was confirmed: the pod is created only for the confirmed context, node and
// detail (image, pull secret), else ErrWrongCluster or ErrTargetsChanged.
func (s *Session) StartNodeShell(ctx context.Context, ref model.ObjectRef) (DockTab, error) {
	b := s.Backend()
	if b == nil {
		return DockTab{}, ErrNotConnected
	}
	w, ok := b.Writer().(NodeShellActions)
	if !ok {
		return DockTab{}, errors.New("this cluster connection cannot open node shells")
	}
	if err := CheckConfirmedContext(ctx, b.Context); err != nil {
		return DockTab{}, err
	}
	r, _, err := s.ResolveRef(ref)
	if err != nil {
		return DockTab{}, err
	}
	image, pullSecret := s.NodeShellSettings(b.Context)
	if req, ok := Confirmed(ctx); ok && req.Detail != NodeShellDetail(r.Name, image, pullSecret) {
		return DockTab{}, fmt.Errorf("%w: the node shell's image or pull secret changed since you confirmed; nothing was created", ErrTargetsChanged)
	}
	if uid, bound := ConfirmedUID(ctx, r); bound {
		if obj, err := s.Lookup(r); err != nil || string(obj.GetUID()) != uid {
			return DockTab{}, fmt.Errorf("%w: node %s is not the one you confirmed; nothing was created", ErrTargetsChanged, r.Name)
		}
	}
	cctx, cancel := context.WithTimeout(ctx, nodeShellCreateTimeout)
	defer cancel()
	created, err := w.CreatePod(cctx, NodeShellPod(nodeShellName(), r.Name, image, pullSecret))
	if err != nil {
		return DockTab{}, fmt.Errorf("node shell on %s: %w", r.Name, err)
	}
	tab := DockTab{
		ID: s.NewDockID(DockNodeShell), Type: DockNodeShell, Title: "Node shell: " + r.Name,
		Context: b.Context, Ref: &r, Container: NodeShellContainer,
		NodeShell: &NodeShellTab{Node: r.Name, Namespace: created.Namespace, Pod: created.Name, Image: image,
			Pinned: ImagePinned(image), DeadlineSeconds: int64(NodeShellDeadline / time.Second)},
	}
	s.shells.add(tab.ID, &nodeShellPod{namespace: created.Namespace, name: created.Name, uid: string(created.UID), actions: w})
	s.UpdateUI(func(u *UIState) {
		u.Dock = append(u.Dock, tab)
		u.ActiveDock = tab.ID
	})
	return tab, nil
}

// openNodeShell starts a node shell tab's session: the notice, the wait for the pod (a watch,
// bounded by nodeShellStartTimeout), then nsenter through the pod shell's exec stream. When the
// session ends (the shell exited, the tab closed), the pod is deleted.
func (s *Session) openNodeShell(ctx context.Context, tab DockTab, b *Backend) (terminal.Session, error) {
	p, ok := s.shells.get(tab.ID)
	if !ok || tab.NodeShell == nil {
		return nil, errors.New("this node shell has ended and its pod is deleted: open a new one")
	}
	notice := fmt.Sprintf("[cogpit] Node shell on %s: privileged pod %s/%s, image %s; the pod ends by itself after %s.\r\n"+
		"[cogpit] Waiting for the pod to run…\r\n", tab.NodeShell.Node, p.namespace, p.name, tab.NodeShell.Image, NodeShellDeadlineText)
	before := func(ctx context.Context) error {
		wctx, cancel := context.WithTimeout(ctx, nodeShellStartTimeout)
		defer cancel()
		err := p.actions.WaitPodRunning(wctx, p.namespace, p.name, NodeShellContainer)
		if err != nil && ctx.Err() == nil && errors.Is(wctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("the pod did not run within %s", nodeShellStartTimeout)
		}
		return err
	}
	sess, err := terminal.StartNodeShell(ctx, b.Conn.RESTConfig(), b.Conn.Kube(), terminal.NodeShellOptions{
		ExecOptions: terminal.ExecOptions{Namespace: p.namespace, Pod: p.name, Container: NodeShellContainer},
		Notice:      notice, Before: before,
	})
	if err != nil {
		s.endNodeShell(tab.ID)
		return nil, err
	}
	go func() {
		defer s.recoverGoroutine("node shell")
		<-sess.Done()
		s.endNodeShell(tab.ID) // the shell ended or the tab closed: the pod goes
	}()
	return sess, nil
}

// nodeShellPod is a node shell's pod, kept until it is deleted.
type nodeShellPod struct {
	namespace, name, uid string
	actions              NodeShellActions
}

// nodeShellSet holds the session's node shell pods by their dock tab, and the deletes in flight.
type nodeShellSet struct {
	mu   sync.Mutex
	pods map[string]*nodeShellPod
	wg   sync.WaitGroup
}

func (n *nodeShellSet) add(tab string, p *nodeShellPod) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.pods == nil {
		n.pods = map[string]*nodeShellPod{}
	}
	n.pods[tab] = p
}

func (n *nodeShellSet) get(tab string) (*nodeShellPod, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	p, ok := n.pods[tab]
	return p, ok
}

// take removes and returns a tab's pod, counting its delete as in flight (wg) under the same lock,
// so that a quit's wait sees every delete a pod was taken for; nil when it has none (any more).
func (n *nodeShellSet) take(tab string) *nodeShellPod {
	n.mu.Lock()
	defer n.mu.Unlock()
	p := n.pods[tab]
	if p != nil {
		delete(n.pods, tab)
		n.wg.Add(1)
	}
	return p
}

// takeAll removes and returns every pod, each delete counted as in flight as take does.
func (n *nodeShellSet) takeAll() []*nodeShellPod {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := make([]*nodeShellPod, 0, len(n.pods))
	for _, p := range n.pods {
		out = append(out, p)
		n.wg.Add(1)
	}
	n.pods = nil
	return out
}

// NodeShellPods returns the pods of the session's open node shells by their dock tab: namespace
// and name.
func (s *Session) NodeShellPods() map[string]string {
	s.shells.mu.Lock()
	defer s.shells.mu.Unlock()
	out := make(map[string]string, len(s.shells.pods))
	for tab, p := range s.shells.pods {
		out[tab] = p.namespace + "/" + p.name
	}
	return out
}

// endNodeShell deletes a node shell tab's pod in the background, once.
func (s *Session) endNodeShell(tab string) {
	p := s.shells.take(tab) // counted in s.shells.wg
	if p == nil {
		return
	}
	go func() {
		defer s.shells.wg.Done()
		defer s.recoverGoroutine("node shell delete")
		s.deleteNodeShellPod(p)
	}()
}

// deleteNodeShellPod deletes one pod with its UID as precondition; a pod that is gone already is
// no failure. A failure is a warning toast: the pod lives until its deadline.
func (s *Session) deleteNodeShellPod(p *nodeShellPod) {
	ctx, cancel := context.WithTimeout(context.Background(), nodeShellDeleteTimeout)
	defer cancel()
	err := p.actions.DeletePod(ctx, p.namespace, p.name, p.uid)
	if err == nil || apierrors.IsNotFound(err) {
		return
	}
	s.log.Warn("node shell: deleting its pod failed", "pod", p.namespace+"/"+p.name, "err", err)
	s.emit(Event{Type: EventToast, Level: LevelWarning, Message: fmt.Sprintf(
		"The node shell's pod %s/%s was not deleted (%v); it ends by itself within %s of its start.", p.namespace, p.name, err, NodeShellDeadlineText)})
}

// endClosedNodeShells deletes the pods of the node shell tabs that left the dock.
func (s *Session) endClosedNodeShells(old, now []DockTab) {
	for _, t := range old {
		if t.Type == DockNodeShell && !slices.ContainsFunc(now, func(n DockTab) bool { return n.ID == t.ID }) {
			s.endNodeShell(t.ID)
		}
	}
}

// endNodeShells deletes every node shell's pod and waits for the deletes, at most
// nodeShellQuitWait: Cogpit quits.
func (s *Session) endNodeShells() {
	for _, p := range s.shells.takeAll() { // each counted in s.shells.wg
		go func() {
			defer s.shells.wg.Done()
			defer s.recoverGoroutine("node shell delete")
			s.deleteNodeShellPod(p)
		}()
	}
	done := make(chan struct{})
	go func() {
		s.shells.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(nodeShellQuitWait):
		s.log.Warn("node shell: pods still being deleted at quit; they end by their deadline")
	}
}

// recoverGoroutine logs a panic of one of the session's own goroutines instead of ending the
// process.
func (s *Session) recoverGoroutine(what string) {
	if r := recover(); r != nil {
		s.log.Error("engine: goroutine panicked", "goroutine", what, "panic", fmt.Sprint(r))
	}
}
