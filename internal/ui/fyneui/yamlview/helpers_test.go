package yamlview

import (
	"fmt"
	"strings"
)

// sampleYAML is a ConfigMap that uses the rarer YAML features followed by a realistic Kubernetes
// object as `kubectl get -o yaml` prints it, so that one screen shows every token kind.
const sampleYAML = `# ConfigMap and Deployment as returned by the API server (trimmed)
apiVersion: v1
kind: ConfigMap
metadata:
  name: web-config # rendered by the chart
data:
  defaults: &defaults
    retries: 3
    timeout: 2.5e1
  production: *defaults
  nginx.conf: |-
    server {
      listen 8080;  # not a comment: block scalar content
    }
  ratio: !!float 0.75
  enabled: yes
---
apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "4"
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"web"}}
  creationTimestamp: "2026-10-01T08:12:44Z"
  generation: 4
  labels:
    app.kubernetes.io/name: web
    app.kubernetes.io/part-of: shop
  name: web
  namespace: production
  resourceVersion: "918273"
  uid: 5b7c1e0a-3f2d-4e8b-9a61-0c2d4e6f8a10
spec:
  progressDeadlineSeconds: 600
  replicas: 3
  selector:
    matchLabels:
      app.kubernetes.io/name: web
  template:
    spec:
      containers:
      - name: web
        image: registry.example.com/shop/web:1.42.0
        imagePullPolicy: IfNotPresent
        args: ["--port=8080", "--log-level=info"]
        env:
        - name: FEATURE_CHECKOUT_V2
          value: "true"
        ports:
        - containerPort: 8080
          name: http
          protocol: TCP
        readinessProbe:
          httpGet: {path: /healthz, port: 8080}
          periodSeconds: 5
        resources:
          limits: {cpu: 500m, memory: 512Mi}
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          runAsUser: 10001
      tolerations: ~
status:
  conditions:
  - lastTransitionTime: "2026-10-01T08:13:02Z"
    message: 'ReplicaSet "web-6d4cf56db6" has successfully progressed.'
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  readyReplicas: 3
...
`

// syntheticYAML returns at least size bytes of pod-like documents with every token kind, block
// scalars and long quoted messages; deterministic, so that benchmarks are comparable.
func syntheticYAML(size int) string {
	var sb strings.Builder
	sb.Grow(size + 4096)
	for i := 0; sb.Len() < size; i++ {
		fmt.Fprintf(&sb, `---
# pod %[1]d of the synthetic benchmark document
apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Pod","metadata":{"name":"web-%[1]d","namespace":"team-%[2]d"}}
    prometheus.io/scrape: "true"
  creationTimestamp: "2026-10-01T08:%02[3]d:44Z"
  labels:
    app.kubernetes.io/name: web
    pod-template-hash: 6d4cf56db6
  name: web-%[1]d
  namespace: team-%[2]d
  ownerReferences:
  - apiVersion: apps/v1
    blockOwnerDeletion: true
    controller: true
    kind: ReplicaSet
    name: web-6d4cf56db6
    uid: 5b7c1e0a-3f2d-4e8b-9a61-%012[1]d
  resourceVersion: "%[4]d"
spec:
  containers:
  - args: ["--port=8080", "--log-level=info"]
    env:
    - name: POD_INDEX
      value: "%[1]d"
    - name: CACHE_TTL_SECONDS
      value: "300"
    image: registry.example.com/shop/web:1.42.%[2]d
    imagePullPolicy: IfNotPresent
    name: web
    ports:
    - containerPort: 8080
      name: http
      protocol: TCP
    resources:
      limits: {cpu: 500m, memory: 512Mi}
      requests:
        cpu: 100m
        memory: 128Mi
    securityContext:
      allowPrivilegeEscalation: false
      runAsUser: 10001
  nodeName: node-%[2]d
  restartPolicy: Always
  terminationGracePeriodSeconds: 30
  tolerations: ~
status:
  conditions:
  - lastProbeTime: null
    lastTransitionTime: "2026-10-01T08:13:02Z"
    message: 'containers with unready status: [web]; the readiness probe failed with
      a timeout after 1s'
    status: "False"
    type: Ready
  phase: Running
  podIP: 10.%[2]d.%[3]d.%[5]d
  qosClass: Burstable
`, i, i%17, i%60, 900000+i, i%250)
	}
	return sb.String()
}
