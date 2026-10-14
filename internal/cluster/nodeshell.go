package cluster

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
)

// The cluster side of a node shell (engine/nodeshell.go, docs/security/privilege-footprint.md ›
// Node shell): the privileged pod is created as the engine built it, its start is awaited by a
// watch on that one pod (never polled), and it is deleted with its UID as precondition (DeletePod).

// ErrPodNotStarted is returned when a pod ends, or its container cannot start, before it runs.
var ErrPodNotStarted = errors.New("the pod did not start")

// waitingFailures are a container's waiting reasons that do not pass by waiting: the image cannot be
// pulled or named, or the container cannot be made.
var waitingFailures = []string{"ErrImagePull", "ImagePullBackOff", "InvalidImageName", "ErrImageNeverPull",
	"CreateContainerConfigError", "CreateContainerError", "RunContainerError"}

// rewatchPause is how long WaitPodRunning waits before it reads the pod again after a watch that
// closed without an event.
const rewatchPause = time.Second

// CreatePod creates a pod as given and returns it as the API server stored it.
func (c *Conn) CreatePod(ctx context.Context, pod *corev1.Pod) (*corev1.Pod, error) {
	out, err := c.kube.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("create pod %s/%s: %w", pod.Namespace, pod.Name, err)
	}
	return out, nil
}

// GetPod reads one pod by name: the node shell journal's sweep reads a pod recorded without its
// UID, to tell Cogpit's from another of the same name (engine/nodeshellsweep.go, H-71).
func (c *Conn) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	pod, err := c.kube.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod %s/%s: %w", namespace, name, err)
	}
	return pod, nil
}

// WaitPodRunning waits until a pod runs its container: one get, then a watch on that pod alone
// (field selector metadata.name) from the version the get returned, again after a watch the server
// closed. It returns an error wrapping ErrPodNotStarted when the pod ends, is deleted, or the
// container waits for a reason that does not pass (an image that cannot be pulled), naming the
// reason; and ctx's error when ctx ends first.
func (c *Conn) WaitPodRunning(ctx context.Context, namespace, name, container string) error {
	pods := c.kube.CoreV1().Pods(namespace)
	selector := fields.OneTermEqualSelector("metadata.name", name).String()
	for {
		pod, err := pods.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("pod %s/%s: %w", namespace, name, err)
		}
		if done, err := podStarted(pod, container); done || err != nil {
			return err
		}
		w, err := pods.Watch(ctx, metav1.ListOptions{FieldSelector: selector, ResourceVersion: pod.ResourceVersion})
		if err != nil {
			return fmt.Errorf("watch pod %s/%s: %w", namespace, name, err)
		}
		done, events, err := followPod(ctx, w, container)
		w.Stop()
		if done || err != nil {
			return err
		}
		if events == 0 { // a watch that ended at once: do not spin
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(rewatchPause):
			}
		}
	}
}

// followPod reads a pod's watch until it runs the container, cannot start, or the watch ends.
func followPod(ctx context.Context, w watch.Interface, container string) (done bool, events int, err error) {
	for {
		select {
		case <-ctx.Done():
			return false, events, ctx.Err()
		case ev, ok := <-w.ResultChan():
			if !ok {
				return false, events, nil
			}
			events++
			switch ev.Type {
			case watch.Deleted:
				return false, events, fmt.Errorf("%w: it was deleted", ErrPodNotStarted)
			case watch.Error:
				return false, events, nil // read it again: a version too old, the server's restart
			}
			pod, isPod := ev.Object.(*corev1.Pod)
			if !isPod {
				continue
			}
			if done, err := podStarted(pod, container); done || err != nil {
				return done, events, err
			}
		}
	}
}

// podStarted reports whether the pod runs the container, and an error wrapping ErrPodNotStarted
// when it never will.
func podStarted(pod *corev1.Pod, container string) (bool, error) {
	switch pod.Status.Phase {
	case corev1.PodSucceeded, corev1.PodFailed:
		return false, fmt.Errorf("%w: it is %s%s", ErrPodNotStarted, pod.Status.Phase, reasonText(pod.Status.Reason, pod.Status.Message))
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name != container {
			continue
		}
		if cs.State.Running != nil && pod.Status.Phase == corev1.PodRunning {
			return true, nil
		}
		if w := cs.State.Waiting; w != nil && slices.Contains(waitingFailures, w.Reason) {
			return false, fmt.Errorf("%w: its container is waiting%s", ErrPodNotStarted, reasonText(w.Reason, w.Message))
		}
		if t := cs.State.Terminated; t != nil {
			return false, fmt.Errorf("%w: its container ended%s", ErrPodNotStarted, reasonText(t.Reason, t.Message))
		}
	}
	return false, nil
}

// reasonText is ": reason (message)", as far as the status gives them.
func reasonText(reason, message string) string {
	switch {
	case reason != "" && message != "":
		return ": " + reason + " (" + message + ")"
	case reason != "":
		return ": " + reason
	case message != "":
		return ": " + message
	}
	return ""
}
