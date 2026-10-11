package cluster

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// ScaleWorkload sets the replica count of a Deployment, StatefulSet or ReplicaSet (resource is
// "deployments", "statefulsets" or "replicasets") with a merge patch on its scale subresource,
// the request kubectl scale sends. Like kubectl, it does not wait for the pods.
func (c *Conn) ScaleWorkload(ctx context.Context, resource, namespace, name string, replicas int32) error {
	if replicas < 0 {
		return fmt.Errorf("scale %s %s/%s: replicas %d is negative", resource, namespace, name, replicas)
	}
	patch := fmt.Appendf(nil, `{"spec":{"replicas":%d}}`, replicas)
	opts := metav1.PatchOptions{FieldManager: FieldManager}
	var err error
	switch resource {
	case "deployments":
		_, err = c.kube.AppsV1().Deployments(namespace).Patch(ctx, name, types.MergePatchType, patch, opts, "scale")
	case "statefulsets":
		_, err = c.kube.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.MergePatchType, patch, opts, "scale")
	case "replicasets":
		_, err = c.kube.AppsV1().ReplicaSets(namespace).Patch(ctx, name, types.MergePatchType, patch, opts, "scale")
	default:
		return fmt.Errorf("scale %s %s/%s: not a kind Cogpit scales", resource, namespace, name)
	}
	if err != nil {
		return fmt.Errorf("scale %s %s/%s: %w", resource, namespace, name, err)
	}
	return nil
}

// SetCronJobSuspend suspends (true) or resumes (false) a CronJob with a merge patch on
// spec.suspend. Jobs already running are not affected.
func (c *Conn) SetCronJobSuspend(ctx context.Context, namespace, name string, suspend bool) error {
	patch := fmt.Appendf(nil, `{"spec":{"suspend":%t}}`, suspend)
	if _, err := c.kube.BatchV1().CronJobs(namespace).Patch(ctx, name, types.MergePatchType, patch,
		metav1.PatchOptions{FieldManager: FieldManager}); err != nil {
		verb := "resume"
		if suspend {
			verb = "suspend"
		}
		return fmt.Errorf("%s cron job %s/%s: %w", verb, namespace, name, err)
	}
	return nil
}

// ManualJobAnnotation marks a Job created by hand from a CronJob, as kubectl create job --from
// sets it.
const ManualJobAnnotation = "cronjob.kubernetes.io/instantiate"

// Job names: the controller labels a Job's pods with its name, and label values are at most 63
// characters; manualSuffixLen random characters follow the "<cronjob>-manual-" prefix.
const (
	maxJobName      = 63
	manualSuffixLen = 5
)

// jobSuffix returns the random part of a manual Job's name; tests replace it.
var jobSuffix = randomSuffix

// TriggerCronJob creates a Job from a CronJob's job template, as `kubectl create job
// --from=cronjob/<name>` does: the template's labels, annotations and spec, the annotation
// cronjob.kubernetes.io/instantiate=manual, and an owner reference that makes the CronJob its
// controller. cronJob is the CronJob as the cache holds it; the API request is one create jobs.
// The Job is named "<cronjob>-manual-<5 random characters>", shortened to 63 characters; a name
// that exists already is retried with new characters. It returns the Job's name.
func (c *Conn) TriggerCronJob(ctx context.Context, cronJob *unstructured.Unstructured) (string, error) {
	if cronJob == nil {
		return "", errors.New("trigger cron job: no cron job")
	}
	var cj batchv1.CronJob
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(cronJob.Object, &cj); err != nil {
		return "", fmt.Errorf("trigger cron job %s/%s: %w", cronJob.GetNamespace(), cronJob.GetName(), err)
	}
	if cj.Namespace == "" || cj.Name == "" || cj.UID == "" {
		return "", fmt.Errorf("trigger cron job %s/%s: the object lacks a namespace, name or UID", cj.Namespace, cj.Name)
	}
	var lastErr error
	for range 3 {
		suffix, err := jobSuffix()
		if err != nil {
			return "", fmt.Errorf("trigger cron job %s/%s: %w", cj.Namespace, cj.Name, err)
		}
		job := JobFromCronJob(&cj, ManualJobName(cj.Name, suffix))
		created, err := c.kube.BatchV1().Jobs(cj.Namespace).Create(ctx, job, metav1.CreateOptions{FieldManager: FieldManager})
		if apierrors.IsAlreadyExists(err) {
			lastErr = err
			continue
		}
		if err != nil {
			return "", fmt.Errorf("trigger cron job %s/%s: %w", cj.Namespace, cj.Name, err)
		}
		return created.Name, nil
	}
	return "", fmt.Errorf("trigger cron job %s/%s: %w", cj.Namespace, cj.Name, lastErr)
}

// JobFromCronJob builds the Job `kubectl create job <name> --from=cronjob/<cronjob>` creates.
func JobFromCronJob(cj *batchv1.CronJob, name string) *batchv1.Job {
	annotations := map[string]string{ManualJobAnnotation: "manual"}
	for k, v := range cj.Spec.JobTemplate.Annotations {
		annotations[k] = v
	}
	controller := true
	return &batchv1.Job{
		TypeMeta: metav1.TypeMeta{APIVersion: batchv1.SchemeGroupVersion.String(), Kind: "Job"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   cj.Namespace,
			Annotations: annotations,
			Labels:      cj.Spec.JobTemplate.Labels,
			// Not metav1.NewControllerRef: it sets blockOwnerDeletion, which needs the right to
			// update cronjobs/finalizers (kubectl leaves it out for the same reason).
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: batchv1.SchemeGroupVersion.String(),
				Kind:       "CronJob",
				Name:       cj.Name,
				UID:        cj.UID,
				Controller: &controller,
			}},
		},
		Spec: *cj.Spec.JobTemplate.Spec.DeepCopy(),
	}
}

// ManualJobName is the name of a Job triggered by hand: "<cronjob>-manual-<suffix>", the prefix
// shortened so that the whole stays within 63 characters.
func ManualJobName(cronJob, suffix string) string {
	prefix := cronJob + "-manual-"
	if limit := maxJobName - len(suffix); len(prefix) > limit {
		prefix = prefix[:max(limit, 0)]
	}
	return prefix + suffix
}

// suffixAlphabet is the alphabet of the API server's generated names: no vowels, no 0, 1 and 3,
// so that no word is spelt by accident.
const suffixAlphabet = "bcdfghjklmnpqrstvwxz2456789"

func randomSuffix() (string, error) {
	var b strings.Builder
	n := big.NewInt(int64(len(suffixAlphabet)))
	for range manualSuffixLen {
		i, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		b.WriteByte(suffixAlphabet[i.Int64()])
	}
	return b.String(), nil
}

// EvictPod evicts one pod through the eviction API, which honours PodDisruptionBudgets. A
// non-empty uid is a precondition, so that a replacement with the same name is not evicted. When a
// budget allows no disruption now, it returns an *EvictionBlockedError naming the budget, at once:
// it does not retry.
func (c *Conn) EvictPod(ctx context.Context, namespace, name, uid string) error {
	ev := &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name}}
	if uid != "" {
		u := types.UID(uid)
		ev.DeleteOptions = &metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &u}}
	}
	if err := c.kube.PolicyV1().Evictions(namespace).Evict(ctx, ev); err != nil {
		return fmt.Errorf("evict pod %s/%s: %w", namespace, name, explainEviction(namespace+"/"+name, err))
	}
	return nil
}

// EvictionBlockedError is an eviction the API server refused for now (HTTP 429): a
// PodDisruptionBudget allows no disruption, or the server asks the client to slow down.
type EvictionBlockedError struct {
	// Pod is the pod as namespace/name.
	Pod string
	// Budget is the PodDisruptionBudget's name when the server named it; empty otherwise.
	Budget string
	// Detail is the server's explanation: "The disruption budget web needs 2 healthy pods and has
	// 2 currently".
	Detail string
	// Err is the API error.
	Err error
}

func (e *EvictionBlockedError) Error() string { return e.reason() }

// Unwrap returns the API error.
func (e *EvictionBlockedError) Unwrap() error { return e.Err }

func (e *EvictionBlockedError) reason() string {
	switch {
	case e.Budget != "":
		return fmt.Sprintf("the PodDisruptionBudget %s allows no disruption now (%s); try again when more of its pods are ready",
			e.Budget, e.Detail)
	case e.Detail != "":
		return fmt.Sprintf("a PodDisruptionBudget allows no disruption now (%s)", e.Detail)
	}
	return "the API server refused the eviction for now (HTTP 429): " + apiMessage(e.Err)
}

// budgetName reads the budget's name from the API server's cause message: "The disruption budget
// <name> needs …" or "The disruption budget <name> is still being processed by the server.".
var budgetName = regexp.MustCompile(`^The disruption budget (\S+) `)

// explainEviction turns an HTTP 429 from the eviction API into an *EvictionBlockedError; other
// errors are returned unchanged.
func explainEviction(pod string, err error) error {
	if err == nil || !apierrors.IsTooManyRequests(err) {
		return err
	}
	blocked := &EvictionBlockedError{Pod: pod, Err: err}
	var status apierrors.APIStatus
	if errors.As(err, &status) {
		if d := status.Status().Details; d != nil {
			for _, cause := range d.Causes {
				if cause.Type != policyv1.DisruptionBudgetCause {
					continue
				}
				blocked.Detail = strings.TrimSuffix(cause.Message, ".")
				if m := budgetName.FindStringSubmatch(cause.Message); m != nil {
					blocked.Budget = m[1]
				}
			}
		}
	}
	return blocked
}

// apiMessage is an API error's message without client-go's prefixes.
func apiMessage(err error) string {
	var status apierrors.APIStatus
	if errors.As(err, &status) && status.Status().Message != "" {
		return status.Status().Message
	}
	if err == nil {
		return ""
	}
	return err.Error()
}
