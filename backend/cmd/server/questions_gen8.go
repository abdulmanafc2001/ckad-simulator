package main

import (
	"fmt"
	"strings"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// Final expansion pack (~40 questions): expert-tier scenarios that go
// beyond typical exam difficulty — manual Endpoints, native sidecars,
// ephemeral debug containers, startup/gRPC probes, StatefulSet update
// partitions, HPA behavior policies, namespace-scoped NetworkPolicies,
// blue/green service switches, DNS tuning and more.

// -------------------------------------- service without selector

func genP8ManualEndpoints() []*models.Question {
	type v struct {
		ns, svc, ep, ip, port string
	}
	variants := []v{
		{"ckad-p8me01", "ext-svc-a", "ext-ep-a", "10.240.1.50", "8080"},
		{"ckad-p8me02", "ext-svc-b", "ext-ep-b", "192.168.99.10", "9090"},
		{"ckad-p8me03", "legacy-svc", "legacy-ep", "172.20.5.77", "8443"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  ports:
  - port: %s
---
apiVersion: v1
kind: Endpoints
metadata:
  name: %s          # MUST match the Service name
subsets:
- addresses:
  - ip: %s
  ports:
  - port: %s`, x.svc, x.ns, x.port, x.svc, x.ip, x.port)
		out = append(out, gq(
			fmt.Sprintf("qg-p8manualep-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Selector-less Service %s", x.svc),
			"A Service without a selector can target arbitrary IPs via a hand-written Endpoints object.",
			fmt.Sprintf("In namespace %s, create a Service named '%s' WITHOUT a selector exposing port %s, plus a matching Endpoints object routing to IP %s on port %s.", x.ns, x.svc, x.port, x.ip, x.port),
			solution, x.ns,
			genHints(
				"The Endpoints object's name must equal the Service's name.",
				"subsets[].addresses[].ip carries the backend IP.",
			),
			[]models.Check{
				gcs("svc", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcr("port", fmt.Sprintf("Service port %s", x.port), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", "^"+x.port+"$"),
				gcs("ep-ip", fmt.Sprintf("Endpoints route to %s", x.ip), 3,
					"get endpoints "+x.svc+" -n "+x.ns+" -o jsonpath={.subsets[0].addresses[0].ip}", x.ip),
				gcr("ep-port", fmt.Sprintf("Endpoints port %s", x.port), 2,
					"get endpoints "+x.svc+" -n "+x.ns+" -o jsonpath={.subsets[0].ports[0].port}", "^"+x.port+"$"),
			},
		))
	}
	return out
}

// --------------------------------------- ingress default backend

func genP8DefaultBackend() []*models.Question {
	type v struct {
		ns, ing, dflt string
	}
	variants := []v{
		{"ckad-p8db01", "fallback-ing", "default-svc"},
		{"ckad-p8db02", "catchall-ing", "notfound-svc"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create backing service",
			CommandArgs: fmt.Sprintf("create deployment web --image=nginx:1.25 -n %s && expose deployment web --port=80 --name=%s -n %s", x.ns, x.dflt, x.ns),
		}}
		solution := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
spec:
  defaultBackend:
    service:
      name: %s
      port:
        number: 80`, x.ing, x.ns, x.dflt)
		out = append(out, gqp(
			fmt.Sprintf("qg-p8defbackend-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Default backend on %s", x.ing),
			"An Ingress defaultBackend catches traffic that matches no rule.",
			fmt.Sprintf("In namespace %s (Service '%s' exists), create an Ingress named '%s' whose defaultBackend routes unmatched traffic to '%s' on port 80.", x.ns, x.dflt, x.ing, x.dflt),
			solution, x.ns, prepare,
			genHints(
				"defaultBackend sits directly under spec — no rules needed.",
				"Useful for custom 404 pages.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Ingress %s exists", x.ing), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o name", "ingress.networking.k8s.io/"+x.ing),
				gcs("backend", fmt.Sprintf("defaultBackend -> %s", x.dflt), 3,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.defaultBackend.service.name}", x.dflt),
				gcr("port", "defaultBackend port 80", 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.defaultBackend.service.port.number}", "^80$"),
			},
		))
	}
	return out
}

// ------------------------------- networkpolicy namespace selector

func genP8NetpolNamespace() []*models.Question {
	type v struct {
		ns, np, selKey, selVal, nsLabelVal string
	}
	variants := []v{
		{"ckad-p8npns01", "from-monitoring", "app", "api", "monitoring"},
		{"ckad-p8npns02", "from-prod-only", "tier", "backend", "production"},
		{"ckad-p8npns03", "from-ci", "app", "webhook", "ci"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: %s
  namespace: %s
spec:
  podSelector:
    matchLabels:
      %s: %s
  policyTypes: [Ingress]
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: %s`, x.np, x.ns, x.selKey, x.selVal, x.nsLabelVal)
		out = append(out, gq(
			fmt.Sprintf("qg-p8npns-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Cross-namespace policy %s", x.np),
			"namespaceSelector admits traffic based on the SOURCE namespace's labels.",
			fmt.Sprintf("In namespace %s, create a NetworkPolicy named '%s' allowing ingress to Pods labeled %s=%s ONLY from namespaces labeled kubernetes.io/metadata.name=%s.", x.ns, x.np, x.selKey, x.selVal, x.nsLabelVal),
			solution, x.ns,
			genHints(
				"Every namespace automatically carries kubernetes.io/metadata.name=<name>.",
				"Combine podSelector AND namespaceSelector in one from entry to require both.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("NetworkPolicy %s exists", x.np), 1,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o name", "networkpolicy.networking.k8s.io/"+x.np),
				gcs("sel", fmt.Sprintf("Selects %s=%s", x.selKey, x.selVal), 1,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.podSelector.matchLabels."+x.selKey+"}", x.selVal),
				gcs("ns-sel", fmt.Sprintf("Allows from namespace %s", x.nsLabelVal), 3,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.ingress[0].from[0].namespaceSelector.matchLabels.kubernetes\\.io/metadata\\.name}", x.nsLabelVal),
			},
		))
	}
	return out
}

// ------------------------------------- netpol port ranges (endPort)

func genP8NetpolPortRange() []*models.Question {
	type v struct {
		ns, np, start, end string
	}
	variants := []v{
		{"ckad-p8npr01", "range-a", "8000", "8999"},
		{"ckad-p8npr02", "range-b", "30000", "32767"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: %s
  namespace: %s
spec:
  podSelector: {}
  policyTypes: [Ingress]
  ingress:
  - ports:
    - protocol: TCP
      port: %s
      endPort: %s`, x.np, x.ns, x.start, x.end)
		out = append(out, gq(
			fmt.Sprintf("qg-p8nprange-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Port-range policy %s", x.np),
			"endPort extends a NetworkPolicy rule to cover an entire port range.",
			fmt.Sprintf("In namespace %s, create a NetworkPolicy named '%s' allowing TCP ingress to ALL Pods for ports %s-%s.", x.ns, x.np, x.start, x.end),
			solution, x.ns,
			genHints(
				"endPort must be >= port; both required for ranges.",
				"podSelector: {} selects every Pod in the namespace.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("NetworkPolicy %s exists", x.np), 1,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o name", "networkpolicy.networking.k8s.io/"+x.np),
				gcr("start", fmt.Sprintf("Range starts at %s", x.start), 2,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.ingress[0].ports[0].port}", "^"+x.start+"$"),
				gcr("end", fmt.Sprintf("Range ends at %s", x.end), 3,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.ingress[0].ports[0].endPort}", "^"+x.end+"$"),
			},
		))
	}
	return out
}

// ---------------------------------- statefulset update partition

func genP8StsPartition() []*models.Question {
	type v struct {
		ns, sts, img, part string
	}
	variants := []v{
		{"ckad-p8stp01", "canary-sts", "nginx:1.26", "2"},
		{"ckad-p8stp02", "phased-sts", "redis:7.2", "1"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name: "create base statefulset",
			YAML: fmt.Sprintf(`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: %s
spec:
  serviceName: %s-hd
  replicas: 4
  selector: {matchLabels: {app: %s}}
  template:
    metadata: {labels: {app: %s}}
    spec:
      containers:
      - {name: web, image: nginx:1.25}`, x.sts, x.sts, x.sts, x.sts),
			Namespace: x.ns,
		}}
		solution := fmt.Sprintf(`# edit spec.updateStrategy then apply
updateStrategy:
  type: RollingUpdate
  rollingUpdate:
    partition: %s`, x.part)
		out = append(out, gqp(
			fmt.Sprintf("qg-p8stspart-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Partitioned rollout on %s", x.sts),
			"The partition field phases a StatefulSet rollout — Pods numbered >= partition update first.",
			fmt.Sprintf("In namespace %s, the StatefulSet '%s' (4 replicas) should roll to image '%s' ONLY for Pods with ordinal >= %s. Set the rollingUpdate partition accordingly.", x.ns, x.sts, x.img, x.part),
			solution, x.ns, prepare,
			genHints(
				"spec.updateStrategy.rollingUpdate.partition controls phased rollouts.",
				"Canary on StatefulSets = high partition value.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("StatefulSet %s exists", x.sts), 1,
					"get sts "+x.sts+" -n "+x.ns+" -o name", "statefulset.apps/"+x.sts),
				gcs("type", "updateStrategy.type=RollingUpdate", 1,
					"get sts "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.updateStrategy.type}", "RollingUpdate"),
				gcr("partition", fmt.Sprintf("partition=%s", x.part), 3,
					"get sts "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.updateStrategy.rollingUpdate.partition}", "^"+x.part+"$"),
			},
		))
	}
	return out
}

// ------------------------------------ hpa behavior policies

func genP8HpaBehavior() []*models.Question {
	type v struct {
		ns, dep, win, podsDown string
	}
	variants := []v{
		{"ckad-p8hpb01", "steady-app", "300", "2"},
		{"ckad-p8hpb02", "gentle-app", "600", "1"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=2", x.dep, x.ns),
		}}
		solution := fmt.Sprintf(`apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: %s
  namespace: %s
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: %s
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target: {type: Utilization, averageUtilization: 70}
  behavior:
    scaleDown:
      stabilizationWindowSeconds: %s
      policies:
      - {type: Pods, value: %s, periodSeconds: 60}`, x.dep, x.ns, x.dep, x.win, x.podsDown)
		out = append(out, gqp(
			fmt.Sprintf("qg-p8hpabehavior-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Scale-down guardrails on %s", x.dep),
			"HPA behavior policies throttle scaling velocity — stabilization windows and per-period caps.",
			fmt.Sprintf("In namespace %s, create an HPA (autoscaling/v2) for Deployment '%s': min 1, max 10, CPU 70%%, scaleDown stabilizationWindowSeconds=%s and a Pods policy limiting removal to %s pod(s)/minute.", x.ns, x.dep, x.win, x.podsDown),
			solution, x.ns, prepare,
			genHints(
				"autoscaling/v2 moves metrics and behavior under spec.behavior.",
				"stabilizationWindowSeconds delays reaction to metric drops.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("HPA %s exists", x.dep), 1,
					"get hpa "+x.dep+" -n "+x.ns+" -o name", "horizontalpodautoscaler.autoscaling/"+x.dep),
				gcr("win", fmt.Sprintf("stabilizationWindow=%ss", x.win), 3,
					"get hpa "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.behavior.scaleDown.stabilizationWindowSeconds}", "^"+x.win+"$"),
				gcr("policy", fmt.Sprintf("max %s Pods down/min", x.podsDown), 2,
					"get hpa "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.behavior.scaleDown.policies[0].value}", "^"+x.podsDown+"$"),
			},
		))
	}
	return out
}

// ------------------------------------- native sidecar containers

func genP8NativeSidecar() []*models.Question {
	type v struct {
		ns, pod, side, main string
	}
	variants := []v{
		{"ckad-p8nsc01", "mesh-pod", "envoy-init", "payments"},
		{"ckad-p8nsc02", "log-pod", "log-agent", "frontend"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  initContainers:
  - name: %s
    image: busybox:1.36
    command: ['sh','-c','sleep infinity']
    restartPolicy: Always     # this makes it a NATIVE sidecar
  containers:
  - name: %s
    image: nginx:1.25`, x.pod, x.ns, x.side, x.main)
		out = append(out, gq(
			fmt.Sprintf("qg-p8nativesidecar-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Native sidecar on %s", x.pod),
			"An init container with restartPolicy: Always becomes a sidecar that runs for the Pod's whole life.",
			fmt.Sprintf("In namespace %s, create Pod '%s' with init container '%s' (busybox:1.36, sleep infinity) marked as a NATIVE SIDECAR via restartPolicy, plus main container '%s' (nginx:1.25).", x.ns, x.pod, x.side, x.main),
			solution, x.ns,
			genHints(
				"restartPolicy: Always goes INSIDE the init container entry.",
				"Native sidecars start before regular containers and stop after them.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("side", fmt.Sprintf("Init container %s present", x.side), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.initContainers[0].name}", "^"+x.side+"$"),
				gcs("policy", "restartPolicy=Always (native sidecar)", 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.initContainers[0].restartPolicy}", "Always"),
				gcs("main", fmt.Sprintf("Main container %s present", x.main), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].name}", x.main),
			},
		))
	}
	return out
}

// ----------------------------------- ephemeral debug containers

func genP8EphemeralDebug() []*models.Question {
	type v struct {
		ns, pod, dbg, target string
	}
	variants := []v{
		{"ckad-p8dbg01", "misbehaving-a", "debugger", "app"},
		{"ckad-p8dbg02", "misbehaving-b", "troubleshoot", "web"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create target pod",
			CommandArgs: fmt.Sprintf("run %s --image=busybox:1.36 -n %s --command -- sleep 3600", x.pod, x.ns),
		}}
		solution := fmt.Sprintf("kubectl debug pod/%s --image=busybox:1.36 --container=%s --target=%s -n %s -- sh",
			x.pod, x.dbg, x.target, x.ns)
		out = append(out, gqp(
			fmt.Sprintf("qg-p8debug-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Debug %s with an ephemeral container", x.pod),
			"Ephemeral containers attach troubleshooting tools to running Pods without restarting them.",
			fmt.Sprintf("In namespace %s, attach an EPHEMERAL debug container named '%s' (busybox:1.36) to the running Pod '%s', targeting its '%s' container so they share the process namespace.", x.ns, x.dbg, x.pod, x.target),
			solution, x.ns, prepare,
			genHints(
				"'kubectl debug pod/NAME --image=... --target=...' does this live.",
				"Ephemeral containers cannot be removed once added — recreate the Pod to clear them.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("img", fmt.Sprintf("Ephemeral image busybox:1.36 (%s)", x.dbg), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.ephemeralContainers[0].image}", "busybox:1.36"),
				gcs("target", fmt.Sprintf("Targets %s", x.target), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.ephemeralContainers[0].targetContainerName}", x.target),
			},
		))
	}
	return out
}

// ---------------------------------------- startup + grpc probes

func genP8StartupGrpcProbes() []*models.Question {
	type v struct {
		ns, pod, kind, path, port string
		isGRPC                    bool
	}
	variants := []v{
		{"ckad-p8sp01", "slow-boot", "startupProbe", "/boot-done", "8080", false},
		{"ckad-p8sp02", "warm-starter", "startupProbe", "/ready-flag", "80", false},
		{"ckad-p8gp01", "grpc-api", "readinessProbe", "", "50051", true},
		{"ckad-p8gp02", "grpc-stream", "livenessProbe", "", "9000", true},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var probeBlock, desc string
		if x.isGRPC {
			probeBlock = fmt.Sprintf(`    %s:
      grpc:
        port: %s`, x.kind, x.port)
			desc = fmt.Sprintf("%s of type grpc on port %s", strings.Title(strings.TrimSuffix(x.kind, "Probe")), x.port)
		} else {
			probeBlock = fmt.Sprintf(`    %s:
      httpGet:
        path: %s
        port: %s
      failureThreshold: 30
      periodSeconds: 5`, x.kind, x.path, x.port)
			desc = fmt.Sprintf("%s httpGet %s:%s tolerant of slow boots (failureThreshold 30)", strings.Title(strings.TrimSuffix(x.kind, "Probe")), x.path, x.port)
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: nginx:1.25
%s`, x.pod, x.ns, probeBlock)
		out = append(out, gq(
			fmt.Sprintf("qg-p8advprobe-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Advanced probe on %s", x.pod),
			"startupProbe shields slow starters; grpc probes check gRPC health endpoints.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) with a %s.", x.ns, x.pod, desc),
			solution, x.ns,
			genHints(
				"A passing startupProbe gates liveness/readiness execution.",
				"gRPC probes need containerPort exposure and the grpc field.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("kind", fmt.Sprintf("%s defined", x.kind), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0]."+x.kind+"}", ""),
				gcr("port", fmt.Sprintf("Probe port %s", x.port), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0]."+x.kind+(map[bool]string{true: ".grpc.port", false: ".httpGet.port"})[!x.isGRPC]+"}", "^"+x.port+"$"),
			},
		))
	}
	return out
}

// --------------------------------------- cronjob time zones

func genP8CronTimeZone() []*models.Question {
	type v struct {
		ns, name, tz, sched string
	}
	variants := []v{
		{"ckad-p8tz01", "tokyo-report", "Asia/Tokyo", "0 9 * * *"},
		{"ckad-p8tz02", "ny-backup", "America/New_York", "30 2 * * *"},
		{"ckad-p8tz03", "berlin-cleanup", "Europe/Berlin", "0 3 * * 1"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: batch/v1
kind: CronJob
metadata:
  name: %s
  namespace: %s
spec:
  schedule: '%s'
  timeZone: %s
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
          - {name: tick, image: busybox:1.36, command: ['sh','-c','date']}`, x.name, x.ns, x.sched, x.tz)
		out = append(out, gq(
			fmt.Sprintf("qg-p8crontz-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Timezone-aware CronJob %s", x.name),
			"CronJobs can pin schedules to an IANA timezone instead of kube-controller-manager's clock.",
			fmt.Sprintf("In namespace %s, create CronJob '%s' (busybox:1.36) scheduled '%s' IN TIMEZONE %s.", x.ns, x.name, x.sched, x.tz),
			solution, x.ns,
			genHints(
				"timeZone takes IANA names like Europe/Berlin.",
				"It sits next to schedule under spec.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("CronJob %s exists", x.name), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o name", "cronjob.batch/"+x.name),
				gcs("sched", fmt.Sprintf("schedule '%s'", x.sched), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.schedule}", x.sched),
				gcs("tz", fmt.Sprintf("timeZone %s", x.tz), 3,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.timeZone}", x.tz),
			},
		))
	}
	return out
}

// --------------------------------------------- suspended jobs

func genP8SuspendedJobs() []*models.Question {
	type v struct {
		ns, job string
	}
	variants := []v{
		{"ckad-p8sj01", "paused-migration"},
		{"ckad-p8sj02", "hold-training"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: %s
spec:
  suspend: true
  template:
    spec:
      restartPolicy: Never
      containers:
      - {name: work, image: busybox:1.36, command: ['sh','-c','echo later']}`, x.job, x.ns)
		out = append(out, gq(
			fmt.Sprintf("qg-p8suspend-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Suspended Job %s", x.job),
			"suspend: true creates a Job that stays queued until flipped back.",
			fmt.Sprintf("In namespace %s, create a Job named '%s' (busybox:1.36) in SUSPENDED state (no Pods should start).", x.ns, x.job),
			solution, x.ns,
			genHints(
				"spec.suspend pauses controller reconciliation.",
				"CronJobs accept the same field.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Job %s exists", x.job), 1,
					"get job "+x.job+" -n "+x.ns+" -o name", "job.batch/"+x.job),
				gcs("suspend", "suspend=true", 3,
					"get job "+x.job+" -n "+x.ns+" -o jsonpath={.spec.suspend}", "true"),
			},
		))
	}
	return out
}

// ------------------------------------------ configmap binaryData

func genP8BinaryConfigMaps() []*models.Question {
	type v struct {
		ns, cm, key, b64 string
	}
	variants := []v{
		{"ckad-p8bin01", "assets-cm", "logo.png.b64", "aUNLQUQtU0lNVUxBVE9S"},
		{"ckad-p8bin02", "fonts-cm", "font.bin.b64", "VzJGZEFHc3ZNVEF4"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
binaryData:
  %s: %s`, x.cm, x.ns, x.key, x.b64)
		out = append(out, gq(
			fmt.Sprintf("qg-p8bindata-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("binaryData ConfigMap %s", x.cm),
			"binaryData stores base64 payloads (images, fonts) alongside text data.",
			fmt.Sprintf("In namespace %s, create a ConfigMap named '%s' using binaryData: key %s with EXACTLY the base64 value %s.", x.ns, x.cm, x.key, x.b64),
			solution, x.ns,
			genHints(
				"binaryData values must be valid base64.",
				"data and binaryData can coexist.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("ConfigMap %s exists", x.cm), 1,
					"get configmap "+x.cm+" -n "+x.ns+" -o name", "configmap/"+x.cm),
				gcs("b64", fmt.Sprintf("binaryData.%s exact", x.key), 3,
					"get configmap "+x.cm+" -n "+x.ns+" -o jsonpath={.binaryData."+x.key+"}", x.b64),
			},
		))
	}
	return out
}

// ------------------------------------------------ immutable secrets

func genP8ImmutableSecrets() []*models.Question {
	type v struct {
		ns, sec, key, val string
	}
	variants := []v{
		{"ckad-p8is01", "frozen-secret-a", "root_password", "ChangeMe!2026"},
		{"ckad-p8is02", "frozen-secret-b", "signing_key", "sk_9f8e7d6c"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
immutable: true
type: Opaque
stringData:
  %s: %s`, x.sec, x.ns, x.key, x.val)
		out = append(out, gq(
			fmt.Sprintf("qg-p8immsecret-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Immutable Secret %s", x.sec),
			"Immutable Secrets reject any spec change — protecting critical credentials.",
			fmt.Sprintf("In namespace %s, create an IMMUTABLE Secret named '%s' (stringData) with key %s=%s.", x.ns, x.sec, x.key, x.val),
			solution, x.ns,
			genHints(
				"immutable: true goes above stringData.",
				"To 'change' it you must delete and recreate.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Secret %s exists", x.sec), 1,
					"get secret "+x.sec+" -n "+x.ns+" -o name", "secret/"+x.sec),
				gcs("immutable", "immutable=true", 3,
					"get secret "+x.sec+" -n "+x.ns+" -o jsonpath={.immutable}", "true"),
				gcr("data", fmt.Sprintf(".data.%s populated", x.key), 1,
					"get secret "+x.sec+" -n "+x.ns+" -o jsonpath={.data."+x.key+"}", `^.+$`),
			},
		))
	}
	return out
}

// ------------------------------------------- dnsConfig tuning

func genP8DnsConfig() []*models.Question {
	type v struct {
		ns, pod, nameserver, search, optName, optVal string
	}
	variants := []v{
		{"ckad-p8dns01", "dns-tuned-a", "8.8.8.8", "corp.svc.cluster.local", "ndots", "2"},
		{"ckad-p8dns02", "dns-tuned-b", "1.1.1.1", "lab.internal", "single-request-reopen", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		optYAML := ""
		if x.optVal != "" {
			optYAML = fmt.Sprintf("\n        value: \"%s\"", x.optVal)
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  dnsConfig:
    nameservers:
    - %s
    searches:
    - %s
    options:
    - name: %s%s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.nameserver, x.search, x.optName, optYAML)
		out = append(out, gq(
			fmt.Sprintf("qg-p8dnsconfig-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("dnsConfig on %s", x.pod),
			"dnsConfig layers custom resolvers, search paths and resolver options onto the Pod.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) with dnsConfig: nameserver %s, search domain %s, option %s%s.", x.ns, x.pod, x.nameserver, x.search, x.optName, optDesc(x.optVal)),
			solution, x.ns,
			genHints(
				"dnsConfig merges with the chosen dnsPolicy.",
				"Options carry a name and optional value.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("ns-server", fmt.Sprintf("nameserver %s", x.nameserver), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.dnsConfig.nameservers[0]}", x.nameserver),
				gcs("search", fmt.Sprintf("search %s", x.search), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.dnsConfig.searches[0]}", x.search),
				gcs("opt", fmt.Sprintf("option %s", x.optName), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.dnsConfig.options[0].name}", x.optName),
			},
		))
	}
	return out
}

func optDesc(val string) string {
	if val == "" {
		return ""
	}
	return "=" + val
}

// -------------------------------------------- multi-host ingress

func genP8MultiHostIngress() []*models.Question {
	type v struct {
		ns, ing, h1, s1, h2, s2 string
	}
	variants := []v{
		{"ckad-p8mh01", "dual-host", "shop.io", "shop-svc", "admin.shop.io", "admin-svc"},
		{"ckad-p8mh02", "region-hosts", "eu.app.io", "eu-svc", "us.app.io", "us-svc"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{Name: "create backing services", CommandArgs: fmt.Sprintf("create deployment web --image=nginx:1.25 -n %s", x.ns)},
			{Name: "expose " + x.s1, CommandArgs: fmt.Sprintf("expose deployment web --port=80 --name=%s -n %s", x.s1, x.ns)},
			{Name: "expose " + x.s2, CommandArgs: fmt.Sprintf("expose deployment web --port=80 --name=%s -n %s", x.s2, x.ns)},
		}
		solution := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
spec:
  rules:
  - host: %s
    http:
      paths:
      - {path: /, pathType: Prefix, backend: {service: {name: %s, port: {number: 80}}}}
  - host: %s
    http:
      paths:
      - {path: /, pathType: Prefix, backend: {service: {name: %s, port: {number: 80}}}}`, x.ing, x.ns, x.h1, x.s1, x.h2, x.s2)
		out = append(out, gqp(
			fmt.Sprintf("qg-p8multihost-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Two-host Ingress %s", x.ing),
			"One Ingress can route by virtual host — each rule pins a different Service.",
			fmt.Sprintf("In namespace %s (Services '%s' and '%s' exist), create Ingress '%s' with TWO host rules: %s -> '%s' and %s -> '%s' (both path / Prefix, port 80).", x.ns, x.s1, x.s2, x.ing, x.h1, x.s1, x.h2, x.s2),
			solution, x.ns, prepare,
			genHints(
				"rules is a LIST — one entry per host.",
				"Host-based routing happens at L7 before path matching.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Ingress %s exists", x.ing), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o name", "ingress.networking.k8s.io/"+x.ing),
				gcs("h1", fmt.Sprintf("First host %s", x.h1), 2,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].host}", x.h1),
				gcs("b1", fmt.Sprintf("First host -> %s", x.s1), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[0].backend.service.name}", x.s1),
				gcs("h2", fmt.Sprintf("Second host %s", x.h2), 2,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[1].host}", x.h2),
				gcs("b2", fmt.Sprintf("Second host -> %s", x.s2), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[1].http.paths[0].backend.service.name}", x.s2),
			},
		))
	}
	return out
}

// --------------------------------- role with resourceNames

func genP8RoleResourceNames() []*models.Question {
	type v struct {
		ns, role, sa, res, verb, resName string
	}
	variants := []v{
		{"ckad-p8rr01", "secret-reader-narrow", "audit-bot", "secrets", "get", "app-credentials"},
		{"ckad-p8rr02", "cm-updater-narrow", "config-bot", "configmaps", "update", "live-settings"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create serviceaccount",
			CommandArgs: fmt.Sprintf("create serviceaccount %s -n %s", x.sa, x.ns),
		}}
		solution := fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: %s
  namespace: %s
rules:
- apiGroups: [""]
  resources: [%s]
  verbs: [%s]
  resourceNames: [%s]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: %s-binding
  namespace: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: %s
subjects:
- kind: ServiceAccount
  name: %s`, x.role, x.ns, x.res, x.verb, x.resName, x.role, x.ns, x.role, x.sa)
		out = append(out, gqp(
			fmt.Sprintf("qg-p8roleres-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Narrow RBAC role %s", x.role),
			"resourceNames restricts a rule to specific OBJECT instances, not whole types.",
			fmt.Sprintf("In namespace %s (SA '%s' exists), create Role '%s' allowing verb '%s' on %s but ONLY the object named '%s', then bind it to the SA.", x.ns, x.sa, x.role, x.verb, x.res, x.resName),
			solution, x.ns, prepare,
			genHints(
				"resourceNames lives inside the rule, sibling to verbs.",
				"Core group uses apiGroups: [''].",
			),
			[]models.Check{
				gcs("role", fmt.Sprintf("Role %s exists", x.role), 1,
					"get role "+x.role+" -n "+x.ns+" -o name", "role.rbac.authorization.k8s.io/"+x.role),
				gcs("verb", fmt.Sprintf("verb %s", x.verb), 1,
					"get role "+x.role+" -n "+x.ns+" -o jsonpath={.rules[0].verbs[0]}", x.verb),
				gcs("resname", fmt.Sprintf("resourceNames includes %s", x.resName), 3,
					"get role "+x.role+" -n "+x.ns+" -o jsonpath={.rules[0].resourceNames[0]}", x.resName),
				gcs("binding", fmt.Sprintf("Bound to %s", x.sa), 2,
					"get rolebinding "+x.role+"-binding -n "+x.ns+" -o jsonpath={.subjects[0].name}", x.sa),
			},
		))
	}
	return out
}

// --------------------------------- blue/green service switch

func genP8BlueGreenSwitch() []*models.Question {
	type v struct {
		ns, svc, blue, green string
	}
	variants := []v{
		{"ckad-p8bg01", "vote-svc", "vote-blue", "vote-green"},
		{"ckad-p8bg02", "pay-svc", "pay-blue", "pay-green"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{Name: "create blue deployment", CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.blue, x.ns)},
			{Name: "label blue", CommandArgs: fmt.Sprintf("label deployment %s version=blue -n %s --overwrite", x.blue, x.ns)},
			{Name: "create green deployment", CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.green, x.ns)},
			{Name: "label green", CommandArgs: fmt.Sprintf("label deployment %s version=green -n %s --overwrite", x.green, x.ns)},
			{Name: "service pointing at BLUE", YAML: fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
spec:
  selector:
    app: %s
    version: blue
  ports:
  - port: 80`, x.svc, x.blue), Namespace: x.ns},
		}
		solution := fmt.Sprintf("# flip the live selector:\nkubectl patch service %s -n %s -p '{\"spec\":{\"selector\":{\"version\":\"green\"}}}'\n# rollback = flip back to blue", x.svc, x.ns)
		out = append(out, gqp(
			fmt.Sprintf("qg-p8bluegreen-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Blue/Green switch on %s", x.svc),
			"Blue/green releases flip ONE Service selector between two full stacks.",
			fmt.Sprintf("In namespace %s, Deployments '%s' (version=blue) and '%s' (version=green) are live behind Service '%s'. Switch production traffic to GREEN by changing only the Service selector.", x.ns, x.blue, x.green, x.svc),
			solution, x.ns, prepare,
			genHints(
				"kubectl patch svc -p or kubectl annotate-free edit both work.",
				"Instant rollback = set version back to blue.",
			),
			[]models.Check{
				gcs("blue", fmt.Sprintf("Deployment %s intact", x.blue), 1,
					"get deploy "+x.blue+" -n "+x.ns+" -o name", "deployment.apps/"+x.blue),
				gcs("green", fmt.Sprintf("Deployment %s intact", x.green), 1,
					"get deploy "+x.green+" -n "+x.ns+" -o name", "deployment.apps/"+x.green),
				gcs("switched", "Service now selects version=green", 4,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.selector.version}", "green"),
			},
		))
	}
	return out
}

// helper kept tiny; avoids struct-method noise above.
