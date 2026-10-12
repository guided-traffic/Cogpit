package cache

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"
)

// Held is a Source over a Memory that something fills only while an informer reads it: the
// objects of a virtual kind made of several kinds (the engine's Applications), which the engine
// derives from other scopes' handles. Every list and every watch of an informer over it is a use:
// Use is called first, and the done it returns when the list ends or the watch is stopped, so the
// filler can hold its handles exactly while some informer runs. A list waits for the Memory to be
// ready (wait), so that an informer's first list never claims a partial state is complete.
type Held struct {
	// Mem holds the objects; the filler sets and deletes them.
	Mem *Memory
	// Use starts a use: wait blocks until Mem holds what a list must return, or fails with the
	// error the list returns; done ends the use. An error refuses the list or watch.
	Use func() (wait func() error, done func(), err error)
}

// ListerWatcher implements Source.
func (h *Held) ListerWatcher(scope Scope) toolscache.ListerWatcher {
	return &heldLW{h: h, inner: &memoryLW{m: h.Mem, scope: scope}}
}

type heldLW struct {
	h     *Held
	inner *memoryLW
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch.
func (lw *heldLW) IsWatchListSemanticsUnSupported() bool { return true }

func (lw *heldLW) List(opts metav1.ListOptions) (runtime.Object, error) {
	wait, done, err := lw.h.Use()
	if err != nil {
		return nil, err
	}
	defer done()
	if err := wait(); err != nil {
		return nil, err
	}
	return lw.inner.List(opts)
}

func (lw *heldLW) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	_, done, err := lw.h.Use()
	if err != nil {
		return nil, err
	}
	w, err := lw.inner.Watch(opts)
	if err != nil {
		done()
		return nil, err
	}
	return &heldWatch{Interface: w, done: done}, nil
}

// heldWatch ends its use when the reflector stops it.
type heldWatch struct {
	watch.Interface
	once sync.Once
	done func()
}

func (w *heldWatch) Stop() {
	w.Interface.Stop()
	w.once.Do(w.done)
}

// compile-time check that the list-watcher has the context-free methods client-go needs.
var _ toolscache.ListerWatcher = (*heldLW)(nil)
