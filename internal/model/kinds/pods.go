package kinds

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Pod status texts that the token rule and the summary refer to more than once.
const (
	podRunning     = "Running"
	podCompleted   = "Completed"
	podSucceeded   = "Succeeded"
	podPending     = "Pending"
	podFailed      = "Failed"
	podTerminating = "Terminating"
	podNotReady    = "NotReady"
	podInitPrefix  = "Init:"
)

// podsResource is the pods kind key and plural resource.
const podsResource = "pods"

var pods = &model.KindProfile{
	Key:        podsResource,
	Version:    "v1",
	Resource:   podsResource,
	Kind:       "Pod",
	Title:      "Pods",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "containers", Title: "Containers", Type: model.ColumnNumber},
		{ID: "restarts", Title: "Restarts", Type: model.ColumnNumber},
		{ID: "controlledBy", Title: "Controlled By", Type: model.ColumnText},
		{ID: "node", Title: "Node", Type: model.ColumnText},
		{ID: "qos", Title: "QoS", Type: model.ColumnText},
		// Status before Age, as the running Lens Desktop shows them (capture, 2026-10-11); its
		// documentation lists them the other way round.
		statusColumn,
		ageColumn,
	},
	Cells: podCells,
	Fields: map[string]fieldFunc{
		"podIP": func(obj *unstructured.Unstructured) []string {
			return podIPs(obj, "podIP", "podIPs")
		},
		"hostIP": func(obj *unstructured.Unstructured) []string {
			return podIPs(obj, "hostIP", "hostIPs")
		},
		"node": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "spec", "nodeName"))
		},
		"image": func(obj *unstructured.Unstructured) []string {
			return podSpecImages(model.NestedMap(obj, "spec"))
		},
		"phase": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "status", "phase"))
		},
		idStatus: func(obj *unstructured.Unstructured) []string {
			return nonEmpty(summarisePod(obj).text)
		},
		"controlledBy": func(obj *unstructured.Unstructured) []string {
			kind, name := controller(obj)
			if kind == "" && name == "" {
				return nil
			}
			return []string{kind + "/" + name}
		},
		"qos": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "status", "qosClass"))
		},
	},
	Detail: podDetail,
}

func podCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	s := summarisePod(obj)
	ownerKind, _ := controller(obj)
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		ratio(s.ready, s.total),
		{Num: s.restarts},
		{Text: ownerKind},
		{Text: model.NestedString(obj, "spec", "nodeName")},
		{Text: model.NestedString(obj, "status", "qosClass")},
		{Text: s.text, Status: s.status},
		model.AgeCell(created(obj)),
	}, s.status
}

// podSummary is what the pod table derives from containers and their statuses.
type podSummary struct {
	text     string // the Status column: Running, CrashLoopBackOff, Init:1/2, Terminating
	status   model.Status
	ready    int64 // ready app and sidecar containers
	total    int64 // app and sidecar containers
	restarts int64
}

// summarisePod derives the Status column the way kubectl users read it. In order of precedence:
//
//  1. A deletion timestamp makes the pod Terminating (Unknown if its node was lost).
//  2. A pod-level reason (Evicted, NodeAffinity, Shutdown) explains the pod as a whole, so it
//     beats whatever its containers were doing when it stopped.
//  3. While init containers run, the first one that has not finished decides: Init:N/M while it
//     works, Init:<reason> or Init:ExitCode:N when it fails.
//  4. Otherwise the first app container that waits or has terminated decides with its reason
//     (CrashLoopBackOff, ImagePullBackOff, OOMKilled, Completed). A pod whose finished
//     containers sit next to running ones is Running when the pod is Ready, NotReady otherwise.
//  5. Otherwise the phase.
//
// Sidecars (init containers with restartPolicy Always) run alongside the app containers, so they
// count towards ready, total and restarts once started, as app containers do. Restarts count the
// containers that matter now: init containers while initialising, app containers afterwards.
func summarisePod(obj any) podSummary {
	var s podSummary
	specContainers := model.NestedSlice(obj, "spec", "containers")
	specInit := model.NestedSlice(obj, "spec", "initContainers")
	statuses := model.NestedSlice(obj, "status", "containerStatuses")
	initStatuses := model.NestedSlice(obj, "status", "initContainerStatuses")
	phase := model.NestedString(obj, "status", "phase")
	podReason := model.NestedString(obj, "status", "reason")

	appCount := len(specContainers)
	if appCount == 0 {
		appCount = len(statuses)
	}
	for _, c := range specInit {
		if isSidecar(c) {
			s.total++
		}
	}
	s.total += int64(appCount)

	var appRestarts, initRestarts, sidecarRestarts int64
	for _, cs := range statuses {
		if ready, _ := model.NestedBool(cs, "ready"); ready {
			s.ready++
		}
		appRestarts += nonNegative(cs, "restartCount")
	}

	initCount := len(specInit)
	if initCount == 0 {
		initCount = len(initStatuses)
	}
	initText := ""
	for i, cs := range initStatuses {
		sidecar := isSidecar(byName(specInit, model.NestedString(cs, "name")))
		if sidecar {
			sidecarRestarts += nonNegative(cs, "restartCount")
			if ready, _ := model.NestedBool(cs, "ready"); ready {
				s.ready++
			}
		} else {
			initRestarts += nonNegative(cs, "restartCount")
		}
		if initText != "" || initContainerDone(cs, sidecar) {
			continue
		}
		initText = initContainerText(cs, i, initCount)
	}
	// The Initialized condition outranks the init container statuses: after a node restart the
	// kubelet may report them afresh although the pod is long past initialisation.
	initialising := initText != "" && conditionStatus(obj, "Initialized", "status", "conditions") != condTrue
	if initialising {
		s.restarts = initRestarts + sidecarRestarts
	} else {
		s.restarts = appRestarts + sidecarRestarts
	}

	text := phase
	switch {
	case podReason != "":
		text = podReason
	case initialising:
		text = initText
	default:
		containerText, hasRunning := appContainersText(statuses)
		if containerText != "" {
			text = containerText
		}
		if text == podCompleted && hasRunning {
			text = podNotReady
			if conditionStatus(obj, "Ready", "status", "conditions") == condTrue {
				text = podRunning
			}
		}
	}
	if text == "" {
		text = textUnknown
	}
	if deleting(obj) {
		text = podTerminating
		if podReason == "NodeLost" {
			text = textUnknown
		}
	}
	s.text = text
	s.status = podToken(text, phase, s.ready, s.total)
	return s
}

// isSidecar reports whether an init container spec is a sidecar: restartPolicy Always.
func isSidecar(spec any) bool {
	return model.NestedString(spec, "restartPolicy") == "Always"
}

// initContainerDone reports whether an init container no longer holds up initialisation: it
// exited with code 0, or it is a sidecar that has started.
func initContainerDone(status any, sidecar bool) bool {
	if term := model.NestedMap(status, "state", "terminated"); term != nil {
		code, _ := model.NestedInt64(term, "exitCode")
		return code == 0
	}
	if sidecar {
		started, _ := model.NestedBool(status, "started")
		return started
	}
	return false
}

// initContainerText describes the init container that blocks initialisation; index is its
// position, which is also the number of init containers done before it.
func initContainerText(status any, index, count int) string {
	if term := model.NestedMap(status, "state", "terminated"); term != nil {
		return terminatedText(term, podInitPrefix)
	}
	if reason := model.NestedString(status, "state", "waiting", "reason"); reason != "" && reason != "PodInitializing" {
		return podInitPrefix + reason
	}
	return podInitPrefix + strconv.Itoa(index) + "/" + strconv.Itoa(count)
}

// appContainersText returns the reason of the first app container that waits or has
// terminated, and whether any container runs and is ready.
func appContainersText(statuses []any) (text string, hasRunning bool) {
	for _, cs := range statuses {
		if waiting := model.NestedMap(cs, "state", "waiting"); waiting != nil {
			if reason := model.NestedString(waiting, "reason"); reason != "" && text == "" {
				text = reason
			}
			continue
		}
		if term := model.NestedMap(cs, "state", "terminated"); term != nil {
			if text == "" {
				text = terminatedText(term, "")
			}
			continue
		}
		if model.NestedMap(cs, "state", "running") != nil {
			if ready, _ := model.NestedBool(cs, "ready"); ready {
				hasRunning = true
			}
		}
	}
	return text, hasRunning
}

// terminatedText names a terminated state: its reason, else the signal, else the exit code.
func terminatedText(term map[string]any, prefix string) string {
	if reason := model.NestedString(term, "reason"); reason != "" {
		return prefix + reason
	}
	if signal, _ := model.NestedInt64(term, "signal"); signal != 0 {
		return prefix + "Signal:" + itoa(signal)
	}
	code, _ := model.NestedInt64(term, "exitCode")
	return prefix + "ExitCode:" + itoa(code)
}

// podToken colours a pod status text. Texts it does not know fall back to the phase, so a reason
// a newer kubelet invents still gets a sensible colour.
func podToken(text, phase string, ready, total int64) model.Status {
	switch text {
	case podRunning:
		if ready >= total {
			return model.StatusOK
		}
		return model.StatusWarning
	case podCompleted, podSucceeded:
		return model.StatusSucceeded
	case podPending, "ContainerCreating", "PodInitializing":
		return model.StatusPending
	case podTerminating:
		return model.StatusTerminating
	case textUnknown, "NodeLost":
		return model.StatusUnknown
	case podNotReady:
		return model.StatusWarning
	}
	if rest, ok := strings.CutPrefix(text, podInitPrefix); ok {
		if isInitProgress(rest) {
			return model.StatusPending
		}
		return model.StatusError
	}
	if isErrorReason(text) || strings.HasPrefix(text, "Signal:") || strings.HasPrefix(text, "ExitCode:") {
		return model.StatusError
	}
	switch phase {
	case podFailed:
		return model.StatusError
	case podPending:
		return model.StatusPending
	case podRunning:
		return model.StatusWarning
	case podSucceeded:
		return model.StatusSucceeded
	}
	return model.StatusUnknown
}

// isErrorReason reports the container and pod reasons that mean the pod cannot run as it is.
func isErrorReason(reason string) bool {
	switch reason {
	case "CrashLoopBackOff", "Error", podFailed, "OOMKilled", "Evicted",
		"ImagePullBackOff", "ErrImagePull", "ErrImageNeverPull", "InvalidImageName",
		"ImageInspectError", "RegistryUnavailable", "SignatureValidationFailed",
		"CreateContainerConfigError", "CreateContainerError", "RunContainerError",
		"PreStartHookError", "PostStartHookError", "ContainerCannotRun",
		"ContainerStatusUnknown", "DeadlineExceeded", "UnexpectedAdmissionError":
		return true
	}
	return false
}

// isInitProgress reports whether s is "N/M": init containers progressing normally.
func isInitProgress(s string) bool {
	done, total, ok := strings.Cut(s, "/")
	return ok && isDigits(done) && isDigits(total)
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// controller returns the kind and name of the object's controlling owner, or of its first owner
// when none is marked as the controller.
func controller(obj any) (kind, name string) {
	owners := model.NestedSlice(obj, "metadata", "ownerReferences")
	var first any
	for _, o := range owners {
		if _, ok := o.(map[string]any); !ok {
			continue
		}
		if first == nil {
			first = o
		}
		if isController, _ := model.NestedBool(o, "controller"); isController {
			return model.NestedString(o, "kind"), model.NestedString(o, "name")
		}
	}
	if first == nil {
		return "", ""
	}
	return model.NestedString(first, "kind"), model.NestedString(first, "name")
}

// podIPs returns a pod's addresses: the plural list (dual-stack) and the singular field, which
// repeats its first entry.
func podIPs(obj any, single, plural string) []string {
	var out []string
	for _, e := range model.NestedSlice(obj, "status", plural) {
		out = appendUnique(out, model.NestedString(e, "ip"))
	}
	return appendUnique(out, model.NestedString(obj, "status", single))
}

func podDetail(obj *unstructured.Unstructured) []model.DetailSection {
	s := summarisePod(obj)
	var status detailItems
	status.addStatus(titleStatus, s.text, s.status)
	status.add("Node", model.NestedString(obj, "spec", "nodeName"))
	status.add("Pod IPs", strings.Join(podIPs(obj, "podIP", "podIPs"), ", "))
	status.add("Host IP", strings.Join(podIPs(obj, "hostIP", "hostIPs"), ", "))
	status.add("QoS Class", model.NestedString(obj, "status", "qosClass"))
	priority := model.NestedString(obj, "spec", "priorityClassName")
	if n, ok := model.NestedInt64(obj, "spec", "priority"); ok && priority != "" {
		priority += " (" + itoa(n) + ")"
	}
	status.add("Priority Class", priority)
	account := model.NestedString(obj, "spec", "serviceAccountName")
	if account == "" {
		account = model.NestedString(obj, "spec", "serviceAccount") // deprecated alias
	}
	status.add("Service Account", account)
	status.add("Restart Policy", model.NestedString(obj, "spec", "restartPolicy"))
	sections := []model.DetailSection{{Title: titleStatus, Items: status}}

	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), podConditionToken); ok {
		sections = append(sections, cs)
	}
	// Readiness applies to app containers and sidecars only; init and ephemeral containers have
	// no probes, so a running one is fine without being ready.
	groups := []struct {
		title, spec, status string
		probed              bool
	}{
		{"Init Container ", "initContainers", "initContainerStatuses", false},
		{"Container ", "containers", "containerStatuses", true},
		{"Ephemeral Container ", "ephemeralContainers", "ephemeralContainerStatuses", false},
	}
	for _, g := range groups {
		statuses := model.NestedSlice(obj, "status", g.status)
		for _, c := range model.NestedSlice(obj, "spec", g.spec) {
			name := model.NestedString(c, "name")
			probed := g.probed || isSidecar(c)
			sections = append(sections, containerSection(g.title+name, c, byName(statuses, name), probed))
		}
	}
	return sections
}

// podConditionToken colours a pod condition. A False condition is a warning, not an error: it
// is also what a pod that is still starting shows, and the Status item says whether it failed.
func podConditionToken(c condition) model.Status {
	switch {
	case c.Reason == "PodCompleted":
		return model.StatusSucceeded // a finished pod is not ready, and that is fine
	case c.Type == "DisruptionTarget":
		return conditionToken(c.Status, model.StatusWarning, model.StatusOK)
	}
	return conditionToken(c.Status, model.StatusOK, model.StatusWarning)
}

// containerSection renders one container: its spec and, when the kubelet reported one, its
// status. Environment values from Secrets and ConfigMaps are named, never resolved. probed says
// whether readiness applies to the container.
func containerSection(title string, spec, status any, probed bool) model.DetailSection {
	var d detailItems
	image := model.NestedString(spec, "image")
	if image == "" {
		image = model.NestedString(status, "image")
	}
	d.add("Image", image)
	if isSidecar(spec) {
		d.add("Restart Policy", "Always")
	}
	if status != nil {
		ready, _ := model.NestedBool(status, "ready")
		text, token := containerState(model.NestedMap(status, "state"), ready || !probed)
		d.addStatus("State", text, token)
		last, _ := containerState(model.NestedMap(status, "lastState"), false)
		d.add("Last State", last)
		d.add("Ready", boolText(ready))
		d.add("Restarts", itoa(nonNegative(status, "restartCount")))
	}
	d.add("Ports", containerPorts(spec))
	d.add("Command", shellJoin(model.NestedStrings(spec, "command")))
	d.add("Args", shellJoin(model.NestedStrings(spec, "args")))
	d.add("Env", containerEnv(spec))
	d.add("Mounts", containerMounts(spec))
	return model.DetailSection{Title: title, Items: d}
}

func boolText(b bool) string {
	if b {
		return condTrue
	}
	return condFalse
}

// containerState renders a container state: "Running, started …", "Waiting: Reason (message)",
// "Terminated: Reason (exit code N), started …, finished …". An empty state renders as "". A
// running container that is not ready is a warning.
func containerState(state map[string]any, ready bool) (string, model.Status) {
	if running := model.NestedMap(state, "running"); running != nil {
		text := podRunning
		if started := timestamp(model.NestedTime(running, "startedAt")); started != "" {
			text += ", started " + started
		}
		if ready {
			return text, model.StatusOK
		}
		return text, model.StatusWarning
	}
	if waiting := model.NestedMap(state, "waiting"); waiting != nil {
		text := "Waiting"
		reason := model.NestedString(waiting, "reason")
		if reason != "" {
			text += ": " + reason
		}
		if msg := model.NestedString(waiting, "message"); msg != "" {
			text += " (" + msg + ")"
		}
		if isErrorReason(reason) {
			return text, model.StatusError
		}
		return text, model.StatusPending
	}
	if term := model.NestedMap(state, "terminated"); term != nil {
		text := "Terminated"
		if reason := model.NestedString(term, "reason"); reason != "" {
			text += ": " + reason
		}
		code, _ := model.NestedInt64(term, "exitCode")
		text += " (exit code " + itoa(code)
		if signal, _ := model.NestedInt64(term, "signal"); signal != 0 {
			text += ", signal " + itoa(signal)
		}
		text += ")"
		if started := timestamp(model.NestedTime(term, "startedAt")); started != "" {
			text += ", started " + started
		}
		if finished := timestamp(model.NestedTime(term, "finishedAt")); finished != "" {
			text += ", finished " + finished
		}
		if code == 0 {
			return text, model.StatusSucceeded
		}
		return text, model.StatusError
	}
	return "", model.StatusNone
}

// containerPorts renders "name: 8080/TCP" lines, with the host port when one is bound.
func containerPorts(spec any) string {
	var lines []string
	for _, p := range model.NestedSlice(spec, "ports") {
		port := model.NestedIntOrString(p, "containerPort")
		if port == "" {
			continue
		}
		line := port + "/" + protocol(p)
		if name := model.NestedString(p, "name"); name != "" {
			line = name + ": " + line
		}
		if host := model.NestedIntOrString(p, "hostPort"); host != "" && host != "0" {
			line += " (host port " + host + ")"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// protocol returns a port's protocol; the API defaults it to TCP.
func protocol(port any) string {
	if p := model.NestedString(port, "protocol"); p != "" {
		return p
	}
	return "TCP"
}

// containerEnv renders the environment: literal values as NAME=value, references by what they
// point at. References are never resolved: the drawer must not become a way to read Secrets.
func containerEnv(spec any) string {
	var lines []string
	for _, e := range model.NestedSlice(spec, "envFrom") {
		var line string
		switch {
		case model.NestedMap(e, "secretRef") != nil:
			line = "from secret " + model.NestedString(e, "secretRef", "name")
		case model.NestedMap(e, "configMapRef") != nil:
			line = "from config map " + model.NestedString(e, "configMapRef", "name")
		default:
			continue
		}
		if prefix := model.NestedString(e, "prefix"); prefix != "" {
			line += " (prefix " + prefix + ")"
		}
		lines = append(lines, line)
	}
	for _, e := range model.NestedSlice(spec, "env") {
		name := model.NestedString(e, "name")
		if name == "" {
			continue
		}
		from := model.NestedMap(e, "valueFrom")
		if from == nil {
			lines = append(lines, name+"="+model.NestedString(e, "value"))
			continue
		}
		lines = append(lines, name+": "+envSource(from))
	}
	return strings.Join(lines, "\n")
}

func envSource(from map[string]any) string {
	switch {
	case model.NestedMap(from, "secretKeyRef") != nil:
		return "from secret " + model.NestedString(from, "secretKeyRef", "name") + "/" +
			model.NestedString(from, "secretKeyRef", "key")
	case model.NestedMap(from, "configMapKeyRef") != nil:
		return "from config map " + model.NestedString(from, "configMapKeyRef", "name") + "/" +
			model.NestedString(from, "configMapKeyRef", "key")
	case model.NestedMap(from, "fieldRef") != nil:
		return "from field " + model.NestedString(from, "fieldRef", "fieldPath")
	case model.NestedMap(from, "resourceFieldRef") != nil:
		return "from resource " + model.NestedString(from, "resourceFieldRef", "resource")
	case model.NestedMap(from, "fileKeyRef") != nil:
		return "from file " + model.NestedString(from, "fileKeyRef", "volumeName") + "/" +
			model.NestedString(from, "fileKeyRef", "path") + " key " +
			model.NestedString(from, "fileKeyRef", "key")
	}
	return "from an unknown source"
}

// containerMounts renders "/path from volume (ro, sub-path x)" lines.
func containerMounts(spec any) string {
	var lines []string
	for _, m := range model.NestedSlice(spec, "volumeMounts") {
		path := model.NestedString(m, "mountPath")
		if path == "" {
			continue
		}
		line := path + " from " + model.NestedString(m, "name")
		var opts []string
		if ro, _ := model.NestedBool(m, "readOnly"); ro {
			opts = append(opts, "ro")
		}
		if sub := model.NestedString(m, "subPath"); sub != "" {
			opts = append(opts, "sub-path "+sub)
		}
		if len(opts) > 0 {
			line += " (" + strings.Join(opts, ", ") + ")"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// shellJoin renders an argument vector so that arguments with spaces stay recognisable.
func shellJoin(args []string) string {
	if len(args) == 0 {
		return ""
	}
	quoted := make([]string, len(args))
	for i, a := range args {
		if a == "" || strings.ContainsAny(a, " \t\n\"'\\") {
			quoted[i] = strconv.Quote(a)
		} else {
			quoted[i] = a
		}
	}
	return strings.Join(quoted, " ")
}
