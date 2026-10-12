package engine

// CustomKindsSettled reports whether the backend knows its custom resource kinds: the definitions'
// initial list is done (or failed: without list and watch on definitions there are none), or the
// backend watches none (the demo cluster, a cluster that serves no definitions). Until then a
// custom kind's key is unknown and Kind says the definitions are still being listed. A headless
// caller that is handed a kind key (cogpit-cli get) waits for it before it resolves a key that is
// not built in; the GUI and agents need not, since the sidebar and kinds.list follow the
// definitions as they arrive.
func (b *Backend) CustomKindsSettled() bool {
	set := b.customSet()
	return set == nil || set.synced || set.err != ""
}
