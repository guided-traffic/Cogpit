package helm

import (
	"context"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"helm.sh/helm/v4/pkg/action"
	chartcommon "helm.sh/helm/v4/pkg/chart/common"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/guided-traffic/cogpit/internal/model"
)

// driverFixture is the Helm SDK on a fake clientset whose Secrets and ConfigMaps hold the release
// records, as Helm's own drivers write them: the storage each action's configuration gets is the
// driver it asks for, recorded, so that a test sees which records an action reads and writes.
type driverFixture struct {
	cs *k8sfake.Clientset

	mu    sync.Mutex
	asked []Driver
}

func newDriverFixture(t *testing.T) (*driverFixture, *Actions) {
	t.Helper()
	f := &driverFixture{cs: k8sfake.NewClientset()}
	a := &Actions{config: func(namespace string, d Driver, _ *slog.Logger) (*action.Configuration, error) {
		f.mu.Lock()
		f.asked = append(f.asked, d)
		f.mu.Unlock()
		var store *storage.Storage
		switch d {
		case DriverConfigMap:
			store = storage.Init(driver.NewConfigMaps(f.cs.CoreV1().ConfigMaps(namespace)))
		default:
			store = storage.Init(driver.NewSecrets(f.cs.CoreV1().Secrets(namespace)))
		}
		return &action.Configuration{Releases: store, KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
			Capabilities: chartcommon.DefaultCapabilities}, nil
	}}
	return f, a
}

func (f *driverFixture) drivers() []Driver {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]Driver(nil), f.asked...)
}

// records returns the release records the clientset holds in a namespace, by kind: the names of
// its release Secrets and ConfigMaps.
func (f *driverFixture) records(t *testing.T, namespace string) (secrets, configMaps []string) {
	t.Helper()
	ss, err := f.cs.CoreV1().Secrets(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: ReleaseSelector})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ss.Items {
		secrets = append(secrets, s.Name)
	}
	cms, err := f.cs.CoreV1().ConfigMaps(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: ReleaseSelector})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cms.Items {
		configMaps = append(configMaps, c.Name)
	}
	return secrets, configMaps
}

// asUnstructured converts a typed record as the cache would hold it.
func asUnstructured(t *testing.T, obj runtime.Object, kind string) *unstructured.Unstructured {
	t.Helper()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		t.Fatal(err)
	}
	u := &unstructured.Unstructured{Object: m}
	u.SetAPIVersion("v1")
	u.SetKind(kind)
	if u.GetUID() == "" { // the fake clientset sets none; an API server always does
		u.SetUID(types.UID("uid-" + u.GetNamespace() + "-" + u.GetName()))
	}
	return u
}

// A release kept in ConfigMaps is installed, upgraded, rolled back and uninstalled through Helm's
// ConfigMaps driver: every record the SDK writes is a ConfigMap, none a Secret, and Cogpit reads
// what it wrote (the records decode into the release, values and all). The zero driver is
// Secrets, Helm's default.
func TestActionsUseTheReleasesDriver(t *testing.T) {
	f, a := newDriverFixture(t)
	ctx := context.Background()
	ch := installChart(t, false)
	rev, err := a.Install(ctx, InstallRequest{Namespace: "apps", Name: "web", Chart: ch, Driver: DriverConfigMap,
		Values: values(t, "greeting: hi\npassword: s3cr3t-value\n")})
	if err != nil || rev != 1 {
		t.Fatalf("install: %d %v", rev, err)
	}
	if s, c := f.records(t, "apps"); len(s) != 0 || strings.Join(c, ",") != "sh.helm.release.v1.web.v1" {
		t.Fatalf("after the install: Secrets %v, ConfigMaps %v", s, c)
	}
	cm, err := f.cs.CoreV1().ConfigMaps("apps").Get(ctx, "sh.helm.release.v1.web.v1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !model.IsHelmRecord(asUnstructured(t, cm, "ConfigMap")) {
		t.Error("the record is no release ConfigMap")
	}
	vals, err := ReleaseValues(asUnstructured(t, cm, "ConfigMap"), false)
	if err != nil || !strings.Contains(vals.YAML, "s3cr3t-value") {
		t.Errorf("Cogpit reads the values back: %+v %v", vals, err)
	}
	objs := NewReleaseProfile().Derive([]*unstructured.Unstructured{asUnstructured(t, cm, "ConfigMap")})
	if len(objs) != 1 || ReleaseStorage(objs[0]) != DriverConfigMap || ReleaseStatus(objs[0]) != "deployed" {
		t.Errorf("derived %v", objs)
	}

	// A second install of the name with the same driver is refused before anything changes.
	if _, err := a.Install(ctx, InstallRequest{Namespace: "apps", Name: "web", Chart: ch, Driver: DriverConfigMap, Values: values(t, "greeting: x\n")}); err == nil {
		t.Error("a second install of web")
	}

	cmA := *a
	cmA.Driver = DriverConfigMap
	if rev, err := cmA.Upgrade(ctx, "apps", "web", "", 1, values(t, "greeting: again\n")); err != nil || rev != 2 {
		t.Fatalf("upgrade: %d %v", rev, err)
	}
	if err := cmA.Rollback("apps", "web", 1, 2); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if s, c := f.records(t, "apps"); len(s) != 0 || len(c) != 3 {
		t.Fatalf("after upgrade and rollback: Secrets %v, ConfigMaps %v", s, c)
	}
	last, err := storage.Init(driver.NewConfigMaps(f.cs.CoreV1().ConfigMaps("apps"))).Last("web")
	if rls, ok := last.(*rspb.Release); err != nil || !ok || rls.Version != 3 || rls.Info.Status != common.StatusDeployed {
		t.Errorf("newest record %v %v", last, err)
	}

	// The Secrets driver does not see a release kept in ConfigMaps: an uninstall there fails and
	// changes nothing.
	if err := a.Uninstall("apps", "web"); err == nil {
		t.Error("the Secrets driver uninstalled a release kept in ConfigMaps")
	}
	if err := cmA.Uninstall("apps", "web"); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if s, c := f.records(t, "apps"); len(s)+len(c) != 0 {
		t.Errorf("after the uninstall: Secrets %v, ConfigMaps %v", s, c)
	}

	// The zero driver is Helm's default.
	if _, err := a.Install(ctx, InstallRequest{Namespace: "apps", Name: "api", Chart: ch, Values: values(t, "greeting: hi\n")}); err != nil {
		t.Fatal(err)
	}
	if s, c := f.records(t, "apps"); strings.Join(s, ",") != "sh.helm.release.v1.api.v1" || len(c) != 0 {
		t.Errorf("a default install: Secrets %v, ConfigMaps %v", s, c)
	}
	want := []Driver{DriverConfigMap, DriverConfigMap, DriverConfigMap, DriverConfigMap, DriverSecret, DriverConfigMap, DriverSecret}
	if got := f.drivers(); strings.Join(driverNames(got), ",") != strings.Join(driverNames(want), ",") {
		t.Errorf("drivers asked for %v, want %v", got, want)
	}
}

func driverNames(ds []Driver) []string {
	out := make([]string, len(ds))
	for i, d := range ds {
		out[i] = string(d)
	}
	return out
}
