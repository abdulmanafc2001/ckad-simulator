package main

import (
	"fmt"
	"strings"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// Ultra-hard expansion pack (#3): beyond-real-exam difficulty.
// StatefulSet claim templates, projected volumes, pod anti-affinity,
// topology spread, priority classes, PDBs, TLS secrets, RBAC roles &
// bindings, host ports, traffic policies, indexed jobs, Ingress TLS,
// seccomp, combined NetworkPolicy peers, full Deployment tuning,
// namespace governance trios, hardened sidecars, triple probes,
// multi-env sourcing, paused rollouts and more.

// --------------------------------------- STS with volumeClaimTemplates

func genP7StsClaims() []*models.Question {
	type v struct {
		ns, sts, svc, vol, size string
		replicas                int
	}
	variants := []v{
		{"ckad-p7stsc01", "pg-ha", "pg-hd", "data", "5Gi", 3},
		{"ckad-p7stsc02", "mysql-cluster", "mysql-hd", "datadir", "10Gi", 3},
		{"ckad-p7stsc03", "kafka-brokers", "kafka-hd", "logs", "20Gi", 3},
		{"ckad-p7stsc04", "zk-ensemble", "zk-hd", "snapshots", "2Gi", 5},
		{"ckad-p7stsc05", "cass-ring", "cass-hd", "commitlog", "8Gi", 4},
		{"ckad-p7stsc06", "mongo-rs", "mongo-hd", "db", "15Gi", 3},
		{"ckad-p7stsc07", "redis-grid", "redis-hd", "dump", "1Gi", 6},
		{"ckad-p7stsc08", "etcd-quorum", "etcd-hd", "wal", "4Gi", 3},
		{"ckad-p7stsc09", "crdb-cloud", "crdb-hd", "store", "12Gi", 3},
		{"ckad-p7stsc10", "rabbit-quorum", "rabbit-hd", "mnesia", "6Gi", 3},
		{"ckad-p7stsc11", "clickhouse-sh", "ch-hd", "parts", "25Gi", 2},
		{"ckad-p7stsc12", "minio-pool", "minio-hd", "export", "30Gi", 4},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: %s
  namespace: %s
spec:
  serviceName: %s
  replicas: %d
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: server
        image: nginx:1.25
        volumeMounts:
        - {name: %s, mountPath: /data}
  volumeClaimTemplates:
  - metadata:
      name: %s
    spec:
      accessModes: ['ReadWriteOnce']
      resources:
        requests:
          storage: %s`, x.sts, x.ns, x.svc, x.replicas, x.sts, x.sts, x.vol, x.vol, x.size)
		out = append(out, gq(
			fmt.Sprintf("qg-p7stsc-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("StatefulSet %s with per-Pod claims", x.sts),
			"volumeClaimTemplates give every StatefulSet Pod its own PersistentVolumeClaim.",
			fmt.Sprintf("In namespace %s, create StatefulSet '%s' (%d replicas, nginx:1.25) with serviceName '%s' and a volumeClaimTemplate named '%s' requesting %s (ReadWriteOnce), mounted at /data.", x.ns, x.sts, x.replicas, x.svc, x.vol, x.size),
			solution, x.ns,
			genHints(
				"volumeClaimTemplates sits under spec, sibling to replicas.",
				"The template creates claims named <claim>-<pod> automatically.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("StatefulSet %s exists", x.sts), 1,
					"get statefulset "+x.sts+" -n "+x.ns+" -o name", "statefulset.apps/"+x.sts),
				gcr("replicas", fmt.Sprintf("replicas=%d", x.replicas), 1,
					"get statefulset "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.replicas}", fmt.Sprintf("^%d$", x.replicas)),
				gcs("tpl-name", fmt.Sprintf("Claim template '%s'", x.vol), 2,
					"get statefulset "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.volumeClaimTemplates[0].metadata.name}", x.vol),
				gcs("tpl-size", fmt.Sprintf("Template requests %s", x.size), 2,
					"get statefulset "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.volumeClaimTemplates[0].spec.resources.requests.storage}", x.size),
				gcs("mount", "Mounted at /data", 1,
					"get statefulset "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].volumeMounts[0].mountPath}", "/data"),
			},
		))
	}
	return out
}

// ------------------------------------------------ projected volumes

func genP7Projected() []*models.Question {
	type v struct {
		ns, pod, sec, cm, mnt string
	}
	variants := []v{
		{"ckad-p7proj01", "proj-a", "cred-a", "cfg-a", "/etc/proj"},
		{"ckad-p7proj02", "proj-b", "cred-b", "cfg-b", "/opt/proj"},
		{"ckad-p7proj03", "proj-c", "cred-c", "cfg-c", "/srv/proj"},
		{"ckad-p7proj04", "proj-d", "cred-d", "cfg-d", "/var/proj"},
		{"ckad-p7proj05", "proj-e", "cred-e", "cfg-e", "/etc/stack"},
		{"ckad-p7proj06", "proj-f", "cred-f", "cfg-f", "/opt/stack"},
		{"ckad-p7proj07", "proj-g", "cred-g", "cfg-g", "/srv/stack"},
		{"ckad-p7proj08", "proj-h", "cred-h", "cfg-h", "/var/stack"},
		{"ckad-p7proj09", "proj-i", "cred-i", "cfg-i", "/etc/mix"},
		{"ckad-p7proj10", "proj-j", "cred-j", "cfg-j", "/opt/mix"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{Name: "create secret", CommandArgs: fmt.Sprintf("create secret generic %s --from-literal=user=admin -n %s", x.sec, x.ns)},
			{Name: "create configmap", CommandArgs: fmt.Sprintf("create configmap %s --from-literal=mode=fast -n %s", x.cm, x.ns)},
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']
    volumeMounts:
    - {name: all, mountPath: %s}
  volumes:
  - name: all
    projected:
      sources:
      - secret: {name: %s}
      - configMap: {name: %s}
      - downwardAPI:
          items:
          - {path: pod-labels, fieldRef: {fieldPath: metadata.labels}}`, x.pod, x.ns, x.mnt, x.sec, x.cm)
		out = append(out, gqp(
			fmt.Sprintf("qg-p7proj-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Projected volume stack in %s", x.pod),
			"A projected volume merges Secret, ConfigMap and downward API into one tree.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) with ONE projected volume at %s combining Secret '%s', ConfigMap '%s', and the Pod's labels via downwardAPI.", x.ns, x.pod, x.mnt, x.sec, x.cm),
			solution, x.ns, prepare,
			genHints(
				"projected.sources is a list; each entry names exactly one source type.",
				"downwardAPI items need both path and fieldRef.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("sec-src", fmt.Sprintf("Sources Secret %s", x.sec), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].projected.sources[0].secret.name}", x.sec),
				gcs("cm-src", fmt.Sprintf("Sources ConfigMap %s", x.cm), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].projected.sources[1].configMap.name}", x.cm),
				gcs("dw-src", "Sources downwardAPI labels", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].projected.sources[2].downwardAPI.items[0].fieldRef.fieldPath}", "metadata.labels"),
				gcs("mnt", fmt.Sprintf("Mounted at %s", x.mnt), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].mountPath}", x.mnt),
			},
		))
	}
	return out
}

// --------------------------------------------------- pod anti-affinity

func genP7AntiAffinity() []*models.Question {
	type v struct {
		ns, pod, key, val, topo string
	}
	variants := []v{
		{"ckad-p7aa01", "aa-web", "app", "frontend", "kubernetes.io/hostname"},
		{"ckad-p7aa02", "aa-api", "app", "backend", "kubernetes.io/hostname"},
		{"ckad-p7aa03", "aa-cache", "tier", "cache", "topology.kubernetes.io/zone"},
		{"ckad-p7aa04", "aa-mq", "role", "broker", "kubernetes.io/hostname"},
		{"ckad-p7aa05", "aa-db", "tier", "database", "topology.kubernetes.io/zone"},
		{"ckad-p7aa06", "aa-search", "app", "indexer", "kubernetes.io/hostname"},
		{"ckad-p7aa07", "aa-fe", "app", "edge", "topology.kubernetes.io/region"},
		{"ckad-p7aa08", "aa-wrk", "role", "worker", "kubernetes.io/hostname"},
		{"ckad-p7aa09", "aa-mon", "app", "collector", "topology.kubernetes.io/zone"},
		{"ckad-p7aa10", "aa-gw", "app", "gateway", "kubernetes.io/hostname"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
  labels:
    %s: %s
spec:
  affinity:
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchLabels:
            %s: %s
        topologyKey: %s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.key, x.val, x.key, x.val, x.topo)
		out = append(out, gq(
			fmt.Sprintf("qg-p7antiaff-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Anti-affinity spread for %s", x.pod),
			"Required pod anti-affinity keeps matching Pods off the same topology domain.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36, labeled %s=%s) with REQUIRED podAntiAffinity against Pods labeled %s=%s using topologyKey %s.", x.ns, x.pod, x.key, x.val, x.key, x.val, x.topo),
			solution, x.ns,
			genHints(
				"The selector matches OTHER Pods you want to avoid.",
				"topologyKey is a node label defining the failure domain.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("sel", fmt.Sprintf("Avoids %s=%s", x.key, x.val), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].labelSelector.matchLabels."+x.key+"}", x.val),
				gcs("topo", fmt.Sprintf("topologyKey %s", x.topo), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[0].topologyKey}", x.topo),
			},
		))
	}
	return out
}

// --------------------------------------------- topology spread

func genP7TopologySpread() []*models.Question {
	type v struct {
		ns, pod, key, val, topo, mode string
		maxSkew                       string
	}
	variants := []v{
		{"ckad-p7ts01", "ts-api", "app", "api", "topology.kubernetes.io/zone", "DoNotSchedule", "1"},
		{"ckad-p7ts02", "ts-web", "app", "web", "kubernetes.io/hostname", "ScheduleAnyway", "2"},
		{"ckad-p7ts03", "ts-cache", "tier", "cache", "topology.kubernetes.io/zone", "DoNotSchedule", "1"},
		{"ckad-p7ts04", "ts-worker", "role", "worker", "kubernetes.io/hostname", "ScheduleAnyway", "3"},
		{"ckad-p7ts05", "ts-edge", "app", "edge", "topology.kubernetes.io/region", "DoNotSchedule", "1"},
		{"ckad-p7ts06", "ts-db", "tier", "db", "topology.kubernetes.io/zone", "DoNotSchedule", "2"},
		{"ckad-p7ts07", "ts-queue", "app", "queue", "kubernetes.io/hostname", "ScheduleAnyway", "1"},
		{"ckad-p7ts08", "ts-ml", "role", "gpu", "topology.kubernetes.io/zone", "DoNotSchedule", "1"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
  labels:
    %s: %s
spec:
  topologySpreadConstraints:
  - maxSkew: %s
    topologyKey: %s
    whenUnsatisfiable: %s
    labelSelector:
      matchLabels:
        %s: %s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.key, x.val, x.maxSkew, x.topo, x.mode, x.key, x.val)
		out = append(out, gq(
			fmt.Sprintf("qg-p7spread-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Topology spread for %s", x.pod),
			"Topology spread constraints distribute Pods evenly across domains.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36, labeled %s=%s) with ONE topologySpreadConstraint: maxSkew=%s over '%s', whenUnsatisfiable=%s, selecting %s=%s.", x.ns, x.pod, x.key, x.val, x.maxSkew, x.topo, x.mode, x.key, x.val),
			solution, x.ns,
			genHints(
				"maxSkew is the allowed imbalance between domains.",
				"DoNotSchedule blocks, ScheduleAnyway only scores.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("skew", fmt.Sprintf("maxSkew=%s", x.maxSkew), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.topologySpreadConstraints[0].maxSkew}", "^"+x.maxSkew+"$"),
				gcs("topo", fmt.Sprintf("topologyKey %s", x.topo), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.topologySpreadConstraints[0].topologyKey}", x.topo),
				gcs("mode", fmt.Sprintf("whenUnsatisfiable=%s", x.mode), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.topologySpreadConstraints[0].whenUnsatisfiable}", x.mode),
			},
		))
	}
	return out
}

// -------------------------------------------------- priority classes

func genP7PriorityPods() []*models.Question {
	type v struct {
		ns, pc, pod string
		value       string
		preempt     bool
	}
	variants := []v{
		{"ckad-p7pc01", "critical-prod", "pc-crit", "1000000", true},
		{"ckad-p7pc02", "high-batch", "pc-high", "50000", false},
		{"ckad-p7pc03", "normal-app", "pc-norm", "1000", false},
		{"ckad-p7pc04", "low-besteffort", "pc-low", "100", true},
		{"ckad-p7pc05", "urgent-hotfix", "pc-hotfix", "900000", false},
		{"ckad-p7pc06", "nightly-jobs", "pc-nightly", "500", false},
		{"ckad-p7pc07", "ml-training", "pc-ml", "20000", true},
		{"ckad-p7pc08", "canary-tier", "pc-canary", "7500", false},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		preemptYAML := ""
		if x.preempt {
			preemptYAML = "\nglobalDefault: false\npreemptionPolicy: Never"
		}
		pcSolution := fmt.Sprintf(`kubectl create priorityclass %s --value=%s%s`, x.pc, x.value, preemptLine(x.preempt))
		podSolution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  priorityClassName: %s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.pc)
		prepare := []models.SetupStep{{
			Name: "create namespace",
			YAML: fmt.Sprintf(`apiVersion: v1
kind: Namespace
metadata:
  name: %s
---
apiVersion: scheduling.k8s.io/v1
kind: PriorityClass
metadata:
  name: %s
value: %s%s`, x.ns, x.pc, x.value, preemptYAML),
		}}
		checks := []models.Check{
			gcs("pod", fmt.Sprintf("Pod %s exists", x.pod), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
			gcs("pri", fmt.Sprintf("Uses PriorityClass %s", x.pc), 3,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.priorityClassName}", x.pc),
			gcr("val", fmt.Sprintf("Resolved priority >= %s", x.value), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.priority}", `^[0-9]+$`),
		}
		if x.preempt {
			checks = append(checks, gcs("nopre", "preemptionPolicy=Never honored", 1,
				"get priorityclass "+x.pc+" -o jsonpath={.preemptionPolicy}", "Never"))
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-p7prio-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("PriorityClass %s -> %s", x.pc, x.pod),
			"PriorityClasses rank Pods for scheduling and eviction decisions.",
			fmt.Sprintf("A PriorityClass '%s' (value %s) already exists. In namespace %s, create Pod '%s' (busybox:1.36) that uses it via priorityClassName.%s", x.pc, x.value, x.ns, x.pod, preemptTaskNote(x.preempt)),
			podSolution+"\n\n# reference solution for the class:\n"+pcSolution, x.ns, prepare,
			genHints(
				"priorityClassName lives under Pod spec.",
				"The scheduler stamps spec.priority from the class.",
			),
			checks,
		))
	}
	return out
}

func preemptLine(preempt bool) string {
	if preempt {
		return " --preemption-policy=Never"
	}
	return ""
}

func preemptTaskNote(preempt bool) string {
	if preempt {
		return " The class must keep preemptionPolicy=Never."
	}
	return ""
}

// -------------------------------------------------------- PDBs

func genP7PDB() []*models.Question {
	type v struct {
		ns, pdb, selKey, selVal, mode, arg string
	}
	variants := []v{
		{"ckad-p7pdb01", "pdb-web", "app", "web", "minAvailable", "2"},
		{"ckad-p7pdb02", "pdb-api", "app", "api", "minAvailable", "50%"},
		{"ckad-p7pdb03", "pdb-db", "tier", "db", "maxUnavailable", "1"},
		{"ckad-p7pdb04", "pdb-cache", "tier", "cache", "maxUnavailable", "25%"},
		{"ckad-p7pdb05", "pdb-mq", "role", "mq", "minAvailable", "1"},
		{"ckad-p7pdb06", "pdb-search", "app", "search", "minAvailable", "75%"},
		{"ckad-p7pdb07", "pdb-fe", "app", "fe", "maxUnavailable", "2"},
		{"ckad-p7pdb08", "pdb-be", "app", "be", "minAvailable", "3"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		field := x.mode
		solution := fmt.Sprintf("kubectl create pdb %s --selector=%s=%s --%s=%s", x.pdb, x.selKey, x.selVal, strings.TrimSuffix(field, "Available"), x.arg)
		if x.mode == "minAvailable" {
			solution = fmt.Sprintf("kubectl create pdb %s --selector=%s=%s --min-available=%s", x.pdb, x.selKey, x.selVal, x.arg)
		} else {
			solution = fmt.Sprintf("kubectl create pdb %s --selector=%s=%s --max-unavailable=%s", x.pdb, x.selKey, x.selVal, x.arg)
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p7pdb-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Disruption budget %s", x.pdb),
			"PodDisruptionBudgets protect workloads during voluntary evictions.",
			fmt.Sprintf("In namespace %s, create a PodDisruptionBudget named '%s' selecting %s=%s with %s=%s.", x.ns, x.pdb, x.selKey, x.selVal, x.mode, x.arg),
			solution, x.ns,
			genHints(
				"'kubectl create pdb' supports --min-available/--max-unavailable.",
				"Percentages are quoted strings in YAML.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("PDB %s exists", x.pdb), 1,
					"get pdb "+x.pdb+" -n "+x.ns+" -o name", "poddisruptionbudget.policy/"+x.pdb),
				gcs("sel", fmt.Sprintf("Selects %s=%s", x.selKey, x.selVal), 1,
					"get pdb "+x.pdb+" -n "+x.ns+" -o jsonpath={.spec.selector.matchLabels."+x.selKey+"}", x.selVal),
				gcr("budget", fmt.Sprintf("%s=%s", x.mode, x.arg), 3,
					"get pdb "+x.pdb+" -n "+x.ns+" -o jsonpath={.spec."+field+"}", "^"+x.arg+"$"),
			},
		))
	}
	return out
}

// ---------------------------------------------------- TLS secrets

func genP7TlsSecrets() []*models.Question {
	type v struct {
		ns, sec, cn string
	}
	variants := []v{
		{"ckad-p7tls01", "tls-shop", "shop.example.com"},
		{"ckad-p7tls02", "tls-api", "api.example.com"},
		{"ckad-p7tls03", "tls-auth", "auth.example.com"},
		{"ckad-p7tls04", "tls-docs", "docs.example.com"},
		{"ckad-p7tls05", "tls-admin", "admin.example.com"},
		{"ckad-p7tls06", "tls-status", "status.example.com"},
		{"ckad-p7tls07", "tls-media", "media.example.com"},
		{"ckad-p7tls08", "tls-dev", "dev.example.com"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -subj "/CN=%s" -keyout /tmp/tls.key -out /tmp/tls.crt
kubectl create secret tls %s --cert=/tmp/tls.crt --key=/tmp/tls.key -n %s`, x.cn, x.sec, x.ns)
		out = append(out, gq(
			fmt.Sprintf("qg-p7tls-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("TLS Secret %s", x.sec),
			"kubernetes.io/tls Secrets require tls.crt and tls.key keys.",
			fmt.Sprintf("In namespace %s, create a Secret named '%s' of type kubernetes.io/tls whose CN is %s (self-signed cert is fine).", x.ns, x.sec, x.cn),
			solution, x.ns,
			genHints(
				"'kubectl create secret tls' wraps cert/key files.",
				"Both tls.crt and tls.key keys MUST exist.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Secret %s exists", x.sec), 1,
					"get secret "+x.sec+" -n "+x.ns+" -o name", "secret/"+x.sec),
				gcs("type", "type is kubernetes.io/tls", 2,
					"get secret "+x.sec+" -n "+x.ns+" -o jsonpath={.type}", "kubernetes.io/tls"),
				gcr("crt", "tls.crt populated", 2,
					"get secret "+x.sec+" -n "+x.ns+" -o jsonpath={.data.tls\\.crt}", `^.+$`),
				gcr("key", "tls.key populated", 2,
					"get secret "+x.sec+" -n "+x.ns+" -o jsonpath={.data.tls\\.key}", `^.+$`),
			},
		))
	}
	return out
}

// ------------------------------------------------------ RBAC roles

func genP7Roles() []*models.Question {
	type v struct {
		ns, role, res, group, verbs string
	}
	variants := []v{
		{"ckad-p7r01", "pod-reader", "pods", "", "get,list"},
		{"ckad-p7r02", "pod-creator", "pods", "", "create,get"},
		{"ckad-p7r03", "deploy-reader", "deployments", "apps", "get,list,watch"},
		{"ckad-p7r04", "job-runner", "jobs", "batch", "create,delete"},
		{"ckad-p7r05", "svc-admin", "services", "", "get,list,create,delete"},
		{"ckad-p7r06", "cm-editor", "configmaps", "", "get,update"},
		{"ckad-p7r07", "secret-viewer", "secrets", "", "get"},
		{"ckad-p7r08", "node-reader", "nodes", "", "get,list"},
		{"ckad-p7r09", "ingress-manager", "ingresses", "networking.k8s.io", "get,create,delete"},
		{"ckad-p7r10", "cron-tuner", "cronjobs", "batch", "get,list,patch"},
		{"ckad-p7r11", "hpa-scaler", "horizontalpodautoscalers", "autoscaling", "get,update"},
		{"ckad-p7r12", "sts-operator", "statefulsets", "apps", "get,scale"},
		{"ckad-p7r13", "netpol-guard", "networkpolicies", "networking.k8s.io", "list"},
		{"ckad-p7r14", "quota-viewer", "resourcequotas", "", "get,list"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		verbList := strings.Split(x.verbs, ",")
		verbsYAML := make([]string, len(verbList))
		for j, vb := range verbList {
			verbsYAML[j] = "- " + vb
		}
		groupYAML := "''"
		if x.group != "" {
			groupYAML = x.group
		}
		solution := fmt.Sprintf(`kubectl create role %s --verb=%s --resource=%s -n %s`,
			x.role, x.verbs, x.res, x.ns)
		yamlSolution := fmt.Sprintf(`# equivalent YAML
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: %s
  namespace: %s
rules:
- apiGroups: [%s]
  resources: [%s]
  verbs:
    %s`, x.role, x.ns, groupYAML, x.res, strings.Join(verbsYAML, "\n    "))
		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Role %s exists", x.role), 1,
				"get role "+x.role+" -n "+x.ns+" -o name", "role.rbac.authorization.k8s.io/"+x.role),
			gcs("res", fmt.Sprintf("Covers resource %s", x.res), 2,
				"get role "+x.role+" -n "+x.ns+" -o jsonpath={.rules[0].resources[0]}", x.res),
		}
		for _, vb := range verbList {
			checks = append(checks, gcr("verb-"+vb, fmt.Sprintf("Verb '%s'", vb), 1,
				"get role "+x.role+" -n "+x.ns+" -o jsonpath={.rules[0].verbs[*]}", `(^| )`+vb+`( |$)`))
		}
		if x.group != "" {
			checks = append(checks, gcs("group", fmt.Sprintf("apiGroup %s", x.group), 1,
				"get role "+x.role+" -n "+x.ns+" -o jsonpath={.rules[0].apiGroups[0]}", x.group))
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p7role-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("RBAC Role %s", x.role),
			"Roles grant namespaced verb/resource permissions.",
			fmt.Sprintf("In namespace %s, create a Role named '%s' allowing verbs [%s] on resource '%s'%s.", x.ns, x.role, x.verbs, x.res, groupNote(x.group)),
			solution+"\n\n"+yamlSolution, x.ns,
			genHints(
				"'kubectl create role NAME --verb=a,b --resource=r' is fastest.",
				"Core group resources use an empty apiGroup.",
			),
			checks,
		))
	}
	return out
}

func groupNote(group string) string {
	if group == "" {
		return " (core API group)"
	}
	return fmt.Sprintf(" in API group '%s'", group)
}

// -------------------------------------------------- role bindings

func genP7RoleBindings() []*models.Question {
	type v struct {
		ns, rb, role, subjKind, subj, subjNs string
	}
	variants := []v{
		{"ckad-p7rb01", "bind-pods", "pod-reader", "User", "alice", ""},
		{"ckad-p7rb02", "bind-deploys", "deploy-reader", "Group", "team-sre", ""},
		{"ckad-p7rb03", "bind-ci", "job-runner", "ServiceAccount", "ci-bot", "ci"},
		{"ckad-p7rb04", "bind-svc", "svc-admin", "ServiceAccount", "ops-bot", "ops"},
		{"ckad-p7rb05", "bind-cm", "cm-editor", "User", "bob", ""},
		{"ckad-p7rb06", "bind-sec", "secret-viewer", "ServiceAccount", "vault-agent", "vault"},
		{"ckad-p7rb07", "bind-ing", "ingress-manager", "Group", "platform-net", ""},
		{"ckad-p7rb08", "bind-cron", "cron-tuner", "ServiceAccount", "scheduler-bot", "tools"},
		{"ckad-p7rb09", "bind-sts", "sts-operator", "User", "carol", ""},
		{"ckad-p7rb10", "bind-quota", "quota-viewer", "Group", "auditors", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create source role",
			CommandArgs: fmt.Sprintf("create role %s --verb=get --resource=pods -n %s", x.role, x.ns),
		}}
		var saFlag, subjDesc string
		switch x.subjKind {
		case "User":
			saFlag = "--user=" + x.subj
			subjDesc = fmt.Sprintf("User '%s'", x.subj)
		case "Group":
			saFlag = "--group=" + x.subj
			subjDesc = fmt.Sprintf("Group '%s'", x.subj)
		default:
			saFlag = fmt.Sprintf("--serviceaccount=%s:%s", x.subjNs, x.subj)
			subjDesc = fmt.Sprintf("ServiceAccount '%s' in namespace %s", x.subj, x.subjNs)
		}
		solution := fmt.Sprintf("kubectl create rolebinding %s --role=%s %s -n %s",
			x.rb, x.role, saFlag, x.ns)
		subjPath := fmt.Sprintf("{.subjects[0].%s}", map[string]string{"User": "name", "Group": "name", "ServiceAccount": "name"}[x.subjKind])
		checks := []models.Check{
			gcs("exists", fmt.Sprintf("RoleBinding %s exists", x.rb), 1,
				"get rolebinding "+x.rb+" -n "+x.ns+" -o name", "rolebinding.rbac.authorization.k8s.io/"+x.rb),
			gcs("role", fmt.Sprintf("References Role %s", x.role), 2,
				"get rolebinding "+x.rb+" -n "+x.ns+" -o jsonpath={.roleRef.name}", x.role),
			gcs("rkind", "roleRef.kind is Role", 1,
				"get rolebinding "+x.rb+" -n "+x.ns+" -o jsonpath={.roleRef.kind}", "Role"),
			gcs("subject", fmt.Sprintf("Binds %s", subjDesc), 2,
				"get rolebinding "+x.rb+" -n "+x.ns+" -o jsonpath="+subjPath, x.subj),
		}
		if x.subjKind == "ServiceAccount" {
			checks = append(checks, gcs("subjns", fmt.Sprintf("Subject namespace %s", x.subjNs), 1,
				"get rolebinding "+x.rb+" -n "+x.ns+" -o jsonpath={.subjects[0].namespace}", x.subjNs))
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-p7rolebind-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Bind %s to %s", x.role, x.subj),
			"RoleBindings attach Roles to users, groups or ServiceAccounts.",
			fmt.Sprintf("In namespace %s (Role '%s' exists), create a RoleBinding named '%s' binding that role to %s.", x.ns, x.role, x.rb, subjDesc),
			solution, x.ns, prepare,
			genHints(
				"'kubectl create rolebinding NAME --role=R --user/--group/--serviceaccount'.",
				"Cross-namespace ServiceAccounts need ns:name syntax.",
			),
			checks,
		))
	}
	return out
}

// ----------------------------------------------------- host ports

func genP7HostPorts() []*models.Question {
	type v struct {
		ns, pod, cport, hport string
	}
	variants := []v{
		{"ckad-p7hp01", "hp-node-a", "8080", "8081"},
		{"ckad-p7hp02", "hp-node-b", "9090", "9091"},
		{"ckad-p7hp03", "hp-node-c", "7070", "7071"},
		{"ckad-p7hp04", "hp-node-d", "6060", "6061"},
		{"ckad-p7hp05", "hp-node-e", "5050", "5051"},
		{"ckad-p7hp06", "hp-node-f", "4040", "4041"},
		{"ckad-p7hp07", "hp-node-g", "3030", "3031"},
		{"ckad-p7hp08", "hp-node-h", "2020", "2021"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: nginx:1.25
    ports:
    - {containerPort: %s, hostPort: %s}`, x.pod, x.ns, x.cport, x.hport)
		out = append(out, gq(
			fmt.Sprintf("qg-p7hostport-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("hostPort mapping on %s", x.pod),
			"hostPort binds the container port directly on the node's IP.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) exposing containerPort %s WITH hostPort %s.", x.ns, x.pod, x.cport, x.hport),
			solution, x.ns,
			genHints(
				"hostPort goes inside the same ports entry as containerPort.",
				"Each hostPort must be unique per node.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("cport", fmt.Sprintf("containerPort %s", x.cport), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].containerPort}", "^"+x.cport+"$"),
				gcr("hport", fmt.Sprintf("hostPort %s", x.hport), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].hostPort}", "^"+x.hport+"$"),
			},
		))
	}
	return out
}

// --------------------------------------------- traffic policies

func genP7TrafficPolicy() []*models.Question {
	type v struct {
		ns, svc, policy string
	}
	variants := []v{
		{"ckad-p7tp01", "tp-local-a", "Local"},
		{"ckad-p7tp02", "tp-cluster-a", "Cluster"},
		{"ckad-p7tp03", "tp-local-b", "Local"},
		{"ckad-p7tp04", "tp-cluster-b", "Cluster"},
		{"ckad-p7tp05", "tp-local-c", "Local"},
		{"ckad-p7tp06", "tp-local-d", "Local"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  type: NodePort
  externalTrafficPolicy: %s
  selector:
    app: web
  ports:
  - port: 80
    targetPort: 8080`, x.svc, x.ns, x.policy)
		out = append(out, gq(
			fmt.Sprintf("qg-p7traffic-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("%s traffic policy on %s", x.policy, x.svc),
			"externalTrafficPolicy trades source-IP preservation against load spreading.",
			fmt.Sprintf("In namespace %s, create a NodePort Service '%s' (selecting app=web, port 80->8080) with externalTrafficPolicy=%s.", x.ns, x.svc, x.policy),
			solution, x.ns,
			genHints(
				"Only NodePort/LoadBalancer Services honor externalTrafficPolicy.",
				"Local preserves client IPs but skips nodes without endpoints.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcs("policy", fmt.Sprintf("externalTrafficPolicy=%s", x.policy), 3,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.externalTrafficPolicy}", x.policy),
				gcs("type", "type is NodePort", 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.type}", "NodePort"),
			},
		))
	}
	return out
}

// -------------------------------------------------- indexed jobs

func genP7IndexedJobs() []*models.Question {
	type v struct {
		ns, job, comp, par, bl string
	}
	variants := []v{
		{"ckad-p7ij01", "shard-index", "4", "2", "2"},
		{"ckad-p7ij02", "chunk-map", "6", "3", "1"},
		{"ckad-p7ij03", "batch-part", "8", "4", "3"},
		{"ckad-p7ij04", "file-split", "5", "5", "2"},
		{"ckad-p7ij05", "row-export", "10", "2", "4"},
		{"ckad-p7ij06", "page-render", "3", "1", "1"},
		{"ckad-p7ij07", "tile-bake", "9", "3", "2"},
		{"ckad-p7ij08", "seq-train", "2", "2", "1"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: %s
spec:
  completions: %s
  parallelism: %s
  completionMode: Indexed
  backoffLimit: %s
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: worker
        image: busybox:1.36
        command: ['sh','-c','echo index $JOB_COMPLETION_INDEX']`, x.job, x.ns, x.comp, x.par, x.bl)
		out = append(out, gq(
			fmt.Sprintf("qg-p7idxjob-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Indexed Job %s", x.job),
			"Indexed Jobs give each Pod a completion index via JOB_COMPLETION_INDEX.",
			fmt.Sprintf("In namespace %s, create Job '%s' (busybox:1.36) with completionMode=Indexed, completions=%s, parallelism=%s and backoffLimit=%s.", x.ns, x.job, x.comp, x.par, x.bl),
			solution, x.ns,
			genHints(
				"completionMode: Indexed sits next to completions.",
				"Each Pod sees its index in JOB_COMPLETION_INDEX.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Job %s exists", x.job), 1,
					"get job "+x.job+" -n "+x.ns+" -o name", "job.batch/"+x.job),
				gcs("mode", "completionMode=Indexed", 3,
					"get job "+x.job+" -n "+x.ns+" -o jsonpath={.spec.completionMode}", "Indexed"),
				gcr("comp", fmt.Sprintf("completions=%s", x.comp), 1,
					"get job "+x.job+" -n "+x.ns+" -o jsonpath={.spec.completions}", "^"+x.comp+"$"),
				gcr("par", fmt.Sprintf("parallelism=%s", x.par), 1,
					"get job "+x.job+" -n "+x.ns+" -o jsonpath={.spec.parallelism}", "^"+x.par+"$"),
				gcr("bl", fmt.Sprintf("backoffLimit=%s", x.bl), 1,
					"get job "+x.job+" -n "+x.ns+" -o jsonpath={.spec.backoffLimit}", "^"+x.bl+"$"),
			},
		))
	}
	return out
}

// ------------------------------------------------- ingress TLS

func genP7IngressTls() []*models.Question {
	type v struct {
		ns, ing, host, sec, path, svc string
	}
	variants := []v{
		{"ckad-p7it01", "secure-shop", "shop.example.com", "shop-tls", "/", "shop-svc"},
		{"ckad-p7it02", "secure-api", "api.example.com", "api-tls", "/v1", "api-svc"},
		{"ckad-p7it03", "secure-auth", "auth.example.com", "auth-tls", "/login", "auth-svc"},
		{"ckad-p7it04", "secure-docs", "docs.example.com", "docs-tls", "/guide", "docs-svc"},
		{"ckad-p7it05", "secure-pay", "pay.example.com", "pay-tls", "/checkout", "pay-svc"},
		{"ckad-p7it06", "secure-mail", "mail.example.com", "mail-tls", "/webmail", "mail-svc"},
		{"ckad-p7it07", "secure-cdn", "cdn.example.com", "cdn-tls", "/assets", "cdn-svc"},
		{"ckad-p7it08", "secure-crm", "crm.example.com", "crm-tls", "/panel", "crm-svc"},
		{"ckad-p7it09", "secure-blog", "blog.example.com", "blog-tls", "/posts", "blog-svc"},
		{"ckad-p7it10", "secure-iot", "iot.example.com", "iot-tls", "/telemetry", "iot-svc"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{Name: "create backing service", CommandArgs: fmt.Sprintf("create deployment dummy --image=nginx:1.25 -n %s && expose deployment dummy --port=80 --name=%s -n %s", x.ns, x.svc, x.ns)},
			{Name: "create TLS secret", CommandArgs: fmt.Sprintf("create secret tls %s --cert=/tmp/x.crt --key=/tmp/x.key -n %s || (openssl req -x509 -nodes -days 1 -newkey rsa:2048 -subj '/CN=x' -keyout /tmp/x.key -out /tmp/x.crt && kubectl create secret tls %s --cert=/tmp/x.crt --key=/tmp/x.key -n %s)", x.sec, x.ns, x.sec, x.ns)},
		}
		solution := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
spec:
  tls:
  - hosts:
    - %s
    secretName: %s
  rules:
  - host: %s
    http:
      paths:
      - path: %s
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: 80`, x.ing, x.ns, x.host, x.sec, x.host, x.path, x.svc)
		out = append(out, gqp(
			fmt.Sprintf("qg-p7ingtls-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("HTTPS Ingress %s", x.ing),
			"Ingress TLS sections terminate HTTPS using a kubernetes.io/tls Secret.",
			fmt.Sprintf("In namespace %s (Service '%s' and TLS Secret '%s' exist), create Ingress '%s': host %s served over TLS (secret %s), path %s Prefix to the service on port 80.", x.ns, x.svc, x.sec, x.ing, x.host, x.sec, x.path),
			solution, x.ns, prepare,
			genHints(
				"spec.tls lists hosts plus the secretName.",
				"Hosts appear in BOTH tls and rules.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Ingress %s exists", x.ing), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o name", "ingress.networking.k8s.io/"+x.ing),
				gcs("tlshost", fmt.Sprintf("TLS host %s", x.host), 2,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.tls[0].hosts[0]}", x.host),
				gcs("tlssec", fmt.Sprintf("TLS secret %s", x.sec), 2,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.tls[0].secretName}", x.sec),
				gcs("rule-host", fmt.Sprintf("Rule host %s", x.host), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].host}", x.host),
				gcs("backend", fmt.Sprintf("Backend %s", x.svc), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[0].backend.service.name}", x.svc),
			},
		))
	}
	return out
}

// ------------------------------------------------------ seccomp

func genP7Seccomp() []*models.Question {
	type v struct {
		ns, pod, level, scope string
	}
	variants := []v{
		{"ckad-p7scmp01", "sc-runtime", "RuntimeDefault", "pod"},
		{"ckad-p7scmp02", "sc-local", "Localhost", "container"},
		{"ckad-p7scmp03", "sc-unconfined", "Unconfined", "pod"},
		{"ckad-p7scmp04", "sc-hard", "RuntimeDefault", "container"},
		{"ckad-p7scmp05", "sc-tight", "RuntimeDefault", "pod"},
		{"ckad-p7scmp06", "sc-profiled", "Localhost", "pod"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var solution, task string
		var checks []models.Check
		if x.scope == "pod" {
			task = fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) whose POD-level securityContext sets seccompProfile.type=%s.", x.ns, x.pod, x.level)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  securityContext:
    seccompProfile:
      type: %s
  containers:
  - name: app
    image: nginx:1.25`, x.pod, x.ns, x.level)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("scmp", fmt.Sprintf("Pod seccomp %s", x.level), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.securityContext.seccompProfile.type}", x.level),
			}
		} else {
			task = fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) whose CONTAINER sets seccompProfile.type=%s (localhostProfile profiles/seccomp.json).", x.ns, x.pod, x.level)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: nginx:1.25
    securityContext:
      seccompProfile:
        type: %s
        localhostProfile: profiles/seccomp.json`, x.pod, x.ns, x.level)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("scmp", fmt.Sprintf("Container seccomp %s", x.level), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.seccompProfile.type}", x.level),
				gcs("prof", "localhostProfile set", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.seccompProfile.localhostProfile}", "profiles/seccomp.json"),
			}
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p7seccomp-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Seccomp hardening: %s", x.pod),
			"Seccomp profiles restrict syscalls at pod or container level.",
			task, solution, x.ns,
			genHints(
				"Localhost profiles need a localhostProfile path.",
				"Pod-level defaults apply to every container.",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------- netpol multi-peer

func genP7NetpolCombo() []*models.Question {
	type v struct {
		ns, np, selK, selV, peerK, peerV, cidr, port string
	}
	variants := []v{
		{"ckad-p7npc01", "np-dual-a", "app", "core", "app", "gateway", "10.0.0.0/8", "8443"},
		{"ckad-p7npc02", "np-dual-b", "tier", "api", "role", "monitor", "192.168.0.0/16", "9090"},
		{"ckad-p7npc03", "np-dual-c", "app", "ledger", "app", "audit", "172.16.0.0/12", "5432"},
		{"ckad-p7npc04", "np-dual-d", "tier", "bus", "app", "queue", "203.0.113.0/24", "5672"},
		{"ckad-p7npc05", "np-dual-e", "app", "search", "role", "indexer", "198.51.100.0/24", "9200"},
		{"ckad-p7npc06", "np-dual-f", "app", "identity", "tier", "authz", "100.64.0.0/10", "8443"},
		{"ckad-p7npc07", "np-dual-g", "role", "worker", "app", "broker", "192.0.2.0/24", "6379"},
		{"ckad-p7npc08", "np-dual-h", "app", "media", "tier", "edge", "198.18.0.0/15", "1935"},
		{"ckad-p7npc09", "np-dual-i", "app", "billing", "role", "cron", "10.228.0.0/14", "7000"},
		{"ckad-p7npc10", "np-dual-j", "tier", "session", "app", "cache", "10.109.0.0/16", "11211"},
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
    - podSelector:
        matchLabels:
          %s: %s
    ports:
    - {protocol: TCP, port: %s}
  - from:
    - ipBlock:
        cidr: %s
    ports:
    - {protocol: TCP, port: %s}`, x.np, x.ns, x.selK, x.selV, x.peerK, x.peerV, x.port, x.cidr, x.port)
		out = append(out, gq(
			fmt.Sprintf("qg-p7npcombo-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Dual-peer policy %s", x.np),
			"One policy can hold multiple ingress entries, each with its own peers and ports.",
			fmt.Sprintf("In namespace %s, create NetworkPolicy '%s' selecting %s=%s with TWO ingress entries: (1) from Pods labeled %s=%s on TCP %s, (2) from CIDR %s on TCP %s.", x.ns, x.np, x.selK, x.selV, x.peerK, x.peerV, x.port, x.cidr, x.port),
			solution, x.ns,
			genHints(
				"Each list item under ingress is an independent OR rule.",
				"ipBlock and podSelector never mix inside one peer entry.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("NetworkPolicy %s exists", x.np), 1,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o name", "networkpolicy.networking.k8s.io/"+x.np),
				gcs("peer1", fmt.Sprintf("Peer1 %s=%s", x.peerK, x.peerV), 2,
					fmt.Sprintf("get networkpolicy %s -n %s -o jsonpath={.spec.ingress[0].from[0].podSelector.matchLabels.%s}", x.np, x.ns, x.peerK), x.peerV),
				gcs("cidr", fmt.Sprintf("Peer2 CIDR %s", x.cidr), 2,
					fmt.Sprintf("get networkpolicy %s -n %s -o jsonpath={.spec.ingress[1].from[0].ipBlock.cidr}", x.np, x.ns), x.cidr),
				gcr("port1", fmt.Sprintf("Entry1 port %s", x.port), 1,
					fmt.Sprintf("get networkpolicy %s -n %s -o jsonpath={.spec.ingress[0].ports[0].port}", x.np, x.ns), "^"+x.port+"$"),
				gcr("port2", fmt.Sprintf("Entry2 port %s", x.port), 1,
					fmt.Sprintf("get networkpolicy %s -n %s -o jsonpath={.spec.ingress[1].ports[0].port}", x.np, x.ns), "^"+x.port+"$"),
			},
		))
	}
	return out
}

// -------------------------------------------- deployment combos

func genP7DeployCombo() []*models.Question {
	type v struct {
		ns, dep, stype, surge, unavail, ready, hist string
	}
	variants := []v{
		{"ckad-p7dc01", "dc-blue", "RollingUpdate", "1", "0", "30", "5"},
		{"ckad-p7dc02", "dc-green", "Recreate", "", "", "10", "3"},
		{"ckad-p7dc03", "dc-canary", "RollingUpdate", "25%", "25%", "45", "2"},
		{"ckad-p7dc04", "dc-fast", "RollingUpdate", "3", "2", "5", "1"},
		{"ckad-p7dc05", "dc-safe", "RollingUpdate", "0", "1", "60", "10"},
		{"ckad-p7dc06", "dc-lean", "Recreate", "", "", "0", "0"},
		{"ckad-p7dc07", "dc-wide", "RollingUpdate", "4", "0", "20", "4"},
		{"ckad-p7dc08", "dc-tight", "RollingUpdate", "1", "1", "15", "6"},
		{"ckad-p7dc09", "dc-blitz", "RollingUpdate", "5", "5", "0", "2"},
		{"ckad-p7dc10", "dc-care", "RollingUpdate", "2", "0", "90", "8"},
		{"ckad-p7dc11", "dc-swap", "Recreate", "", "", "25", "1"},
		{"ckad-p7dc12", "dc-drift", "RollingUpdate", "30%", "10%", "35", "7"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=4", x.dep, x.ns),
		}}
		strategy := fmt.Sprintf("  strategy:\n    type: %s\n", x.stype)
		taskParts := []string{fmt.Sprintf("strategy type %s", x.stype)}
		if x.stype == "RollingUpdate" {
			strategy += fmt.Sprintf("    rollingUpdate:\n      maxSurge: %s\n      maxUnavailable: %s\n", x.surge, x.unavail)
			taskParts = append(taskParts, fmt.Sprintf("maxSurge=%s maxUnavailable=%s", x.surge, x.unavail))
		}
		strategy += fmt.Sprintf("  minReadySeconds: %s\n  revisionHistoryLimit: %s\n", x.ready, x.hist)
		taskParts = append(taskParts, fmt.Sprintf("minReadySeconds=%s revisionHistoryLimit=%s", x.ready, x.hist))

		solution := fmt.Sprintf(`# patch or edit the existing Deployment
spec:
%s  replicas: 4
  selector:
    matchLabels:
      app: %s
  template:
    metadata:
      labels:
        app: %s
    spec:
      containers:
      - name: nginx
        image: nginx:1.25`, strategy, x.dep, x.dep)

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
				"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
			gcs("stype", fmt.Sprintf("strategy=%s", x.stype), 1,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.strategy.type}", x.stype),
			gcr("ready", fmt.Sprintf("minReadySeconds=%s", x.ready), 1,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.minReadySeconds}", "^"+x.ready+"$"),
			gcr("hist", fmt.Sprintf("revisionHistoryLimit=%s", x.hist), 1,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.revisionHistoryLimit}", "^"+x.hist+"$"),
		}
		if x.surge != "" {
			checks = append(checks, gcr("surge", fmt.Sprintf("maxSurge=%s", x.surge), 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.strategy.rollingUpdate.maxSurge}", "^"+x.surge+"$"))
			checks = append(checks, gcr("unavail", fmt.Sprintf("maxUnavailable=%s", x.unavail), 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.strategy.rollingUpdate.maxUnavailable}", "^"+x.unavail+"$"))
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-p7depcombo-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Full rollout tuning: %s", x.dep),
			"Combine strategy, surge bounds and readiness windows in one edit.",
			fmt.Sprintf("In namespace %s, update Deployment '%s' to set: %s.", x.ns, x.dep, strings.Join(taskParts, ", ")),
			solution, x.ns, prepare,
			genHints(
				"kubectl patch --type=merge handles scalars; strategy needs care.",
				"A full 'kubectl apply' of the edited YAML is safest.",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------ namespace governance

func genP7NsGovernance() []*models.Question {
	type v struct {
		ns, nsLabel, qName, lrName, pods, cpu string
	}
	variants := []v{
		{"ckad-p7gov01", "team-alpha", "gov-q-a", "gov-lr-a", "10", "4"},
		{"ckad-p7gov02", "team-beta", "gov-q-b", "gov-lr-b", "20", "8"},
		{"ckad-p7gov03", "team-gamma", "gov-q-c", "gov-lr-c", "15", "6"},
		{"ckad-p7gov04", "team-delta", "gov-q-d", "gov-lr-d", "8", "2"},
		{"ckad-p7gov05", "team-epsilon", "gov-q-e", "gov-lr-e", "12", "5"},
		{"ckad-p7gov06", "team-zeta", "gov-q-f", "gov-lr-f", "25", "10"},
		{"ckad-p7gov07", "team-eta", "gov-q-g", "gov-lr-g", "6", "3"},
		{"ckad-p7gov08", "team-theta", "gov-q-h", "gov-lr-h", "18", "7"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`# 1) labeled namespace
kubectl create namespace %s
kubectl label namespace %s team=%s

# 2) quota
kubectl create quota %s --hard=pods=%s,requests.cpu=%s -n %s

# 3) default limits
cat <<'YAML' | kubectl apply -n %s -f -
apiVersion: v1
kind: LimitRange
metadata:
  name: %s
spec:
  limits:
  - type: Container
    default: {cpu: 200m, memory: 256Mi}
    defaultRequest: {cpu: 100m, memory: 128Mi}
YAML`, x.ns, x.ns, x.nsLabel, x.qName, x.pods, x.cpu, x.ns, x.ns, x.lrName)
		out = append(out, gq(
			fmt.Sprintf("qg-p7gov-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Govern namespace %s", x.ns),
			"Real platforms bootstrap namespaces with labels, quotas and defaults together.",
			fmt.Sprintf("Create namespace '%s' labeled team=%s, then add ResourceQuota '%s' (pods=%s, requests.cpu=%s) and LimitRange '%s' with container defaults cpu=200m/memory=256Mi and requests cpu=100m/memory=128Mi — all inside it.", x.ns, x.nsLabel, x.qName, x.pods, x.cpu, x.lrName),
			solution, x.ns,
			genHints(
				"Three objects: Namespace(+label), ResourceQuota, LimitRange.",
				"'kubectl create quota' accepts comma-separated hard limits.",
			),
			[]models.Check{
				gcs("ns", fmt.Sprintf("Namespace %s exists", x.ns), 1,
					"get namespace "+x.ns+" -o name", "namespace/"+x.ns),
				gcs("nslabel", fmt.Sprintf("Labeled team=%s", x.nsLabel), 1,
					"get namespace "+x.ns+" -o jsonpath={.metadata.labels.team}", x.nsLabel),
				gcs("quota", fmt.Sprintf("Quota %s present", x.qName), 1,
					"get resourcequota "+x.qName+" -n "+x.ns+" -o name", "resourcequota/"+x.qName),
				gcr("qpods", fmt.Sprintf("Quota pods=%s", x.pods), 1,
					"get resourcequota "+x.qName+" -n "+x.ns+" -o jsonpath={.spec.hard.pods}", "^"+x.pods+"$"),
				gcs("lr", fmt.Sprintf("LimitRange %s present", x.lrName), 1,
					"get limitrange "+x.lrName+" -n "+x.ns+" -o name", "limitrange/"+x.lrName),
				gcs("lrdef", "Default cpu=200m", 1,
					"get limitrange "+x.lrName+" -n "+x.ns+" -o jsonpath={.spec.limits[0].default.cpu}", "200m"),
			},
		))
	}
	return out
}

// ----------------------------------------------- hardened sidecars

func genP7SidecarHardened() []*models.Question {
	type v struct {
		ns, pod, side, cpuReq, memReq, period string
	}
	variants := []v{
		{"ckad-p7side01", "side-a", "log-shipper", "50m", "64Mi", "10"},
		{"ckad-p7side02", "side-b", "metrics-fed", "100m", "128Mi", "15"},
		{"ckad-p7side03", "side-c", "trace-agent", "75m", "96Mi", "20"},
		{"ckad-p7side04", "side-d", "conf-sync", "25m", "32Mi", "5"},
		{"ckad-p7side05", "side-e", "cert-watch", "60m", "72Mi", "30"},
		{"ckad-p7side06", "side-f", "health-relay", "80m", "160Mi", "12"},
		{"ckad-p7side07", "side-g", "backup-hook", "40m", "56Mi", "25"},
		{"ckad-p7side08", "side-h", "traffic-mirror", "150m", "256Mi", "8"},
		{"ckad-p7side09", "side-i", "audit-tail", "55m", "88Mi", "18"},
		{"ckad-p7side10", "side-j", "cache-warmer", "120m", "192Mi", "14"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: nginx:1.25
  - name: %s
    image: busybox:1.36
    command: ['sh','-c','while true; do date; sleep %s; done']
    readinessProbe:
      exec:
        command: ['test', '-f', '/tmp/ready']
      periodSeconds: %s
    resources:
      requests: {cpu: %s, memory: %s}
      limits: {cpu: %s, memory: %s}`,
			x.pod, x.ns, x.side, x.period, x.period, x.cpuReq, x.memReq, doubleCPU(x.cpuReq), doubleMem(x.memReq))
		out = append(out, gq(
			fmt.Sprintf("qg-p7sidehard-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Production sidecar %s", x.side),
			"Sidecars need their own probes and resources like any workload.",
			fmt.Sprintf("In namespace %s, create Pod '%s' with main container (nginx:1.25) and sidecar '%s' (busybox:1.36) that: runs a date loop every %ss, has an EXEC readiness probe ('test -f /tmp/ready') with periodSeconds=%s, requests cpu=%s memory=%s and limits cpu=%s memory=%s.",
				x.ns, x.pod, x.side, x.period, x.period, x.cpuReq, x.memReq, doubleCPU(x.cpuReq), doubleMem(x.memReq)),
			solution, x.ns,
			genHints(
				"Sidecar container is just a second entry under containers.",
				"Limits may differ from requests — here they're doubled.",
			),
			[]models.Check{
				gcr("main", "main container present", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )main( |$)`),
				gcr("side", fmt.Sprintf("Sidecar %s present", x.side), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )`+x.side+`( |$)`),
				gcs("probe", "Sidecar exec readiness probe", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[1].readinessProbe.exec.command[0]}", "test"),
				gcr("period", fmt.Sprintf("periodSeconds=%s", x.period), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[1].readinessProbe.periodSeconds}", "^"+x.period+"$"),
				gcr("cpureq", fmt.Sprintf("cpu request %s", x.cpuReq), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[1].resources.requests.cpu}", "^"+x.cpuReq+"$"),
				gcr("memlim", fmt.Sprintf("memory limit %s", doubleMem(x.memReq)), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[1].resources.limits.memory}", "^"+doubleMem(x.memReq)+"$"),
			},
		))
	}
	return out
}

func doubleCPU(s string) string {
	milli := strings.HasSuffix(s, "m")
	n := 0
	fmt.Sscanf(strings.TrimSuffix(s, "m"), "%d", &n)
	if milli {
		return fmt.Sprintf("%dm", n*2)
	}
	return fmt.Sprintf("%d", n*2)
}

func doubleMem(s string) string {
	var n int
	var unit string
	fmt.Sscanf(s, "%d%s", &n, &unit)
	return fmt.Sprintf("%d%s", n*2, unit)
}

// ------------------------------------------------ cron full tuning

func genP7CronFull() []*models.Question {
	type v struct {
		ns, name, sched, conc, sds, shist, fhist, deadline string
	}
	variants := []v{
		{"ckad-p7cf01", "cf-a", "*/2 * * * *", "Forbid", "60", "3", "1", "30"},
		{"ckad-p7cf02", "cf-b", "*/5 * * * *", "Replace", "120", "1", "2", "45"},
		{"ckad-p7cf03", "cf-c", "*/10 * * * *", "Allow", "30", "2", "3", "60"},
		{"ckad-p7cf04", "cf-d", "@hourly", "Forbid", "300", "0", "0", "120"},
		{"ckad-p7cf05", "cf-e", "*/3 * * * *", "Replace", "90", "4", "1", "20"},
		{"ckad-p7cf06", "cf-f", "*/7 * * * *", "Forbid", "150", "5", "5", "90"},
		{"ckad-p7cf07", "cf-g", "@daily", "Replace", "3600", "1", "1", "600"},
		{"ckad-p7cf08", "cf-h", "*/4 * * * *", "Forbid", "45", "2", "4", "15"},
		{"ckad-p7cf09", "cf-i", "*/6 * * * *", "Allow", "75", "3", "2", "25"},
		{"ckad-p7cf10", "cf-j", "*/9 * * * *", "Replace", "180", "6", "6", "300"},
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
  concurrencyPolicy: %s
  startingDeadlineSeconds: %s
  successfulJobsHistoryLimit: %s
  failedJobsHistoryLimit: %s
  jobTemplate:
    spec:
      activeDeadlineSeconds: %s
      template:
        spec:
          restartPolicy: Never
          containers:
          - name: tick
            image: busybox:1.36
            command: ['sh','-c','date']`, x.name, x.ns, x.sched, x.conc, x.sds, x.shist, x.fhist, x.deadline)
		out = append(out, gq(
			fmt.Sprintf("qg-p7cronfull-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Fully tuned CronJob %s", x.name),
			"CronJobs expose overlap, deadline and history controls at two levels.",
			fmt.Sprintf("In namespace %s, create CronJob '%s' (busybox:1.36): schedule '%s', concurrencyPolicy=%s, startingDeadlineSeconds=%s, successfulJobsHistoryLimit=%s, failedJobsHistoryLimit=%s, and the inner JOB gets activeDeadlineSeconds=%s.",
				x.ns, x.name, x.sched, x.conc, x.sds, x.shist, x.fhist, x.deadline),
			solution, x.ns,
			genHints(
				"activeDeadlineSeconds belongs in jobTemplate.spec, not CronJob spec.",
				"History limits sit directly under CronJob spec.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("CronJob %s exists", x.name), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o name", "cronjob.batch/"+x.name),
				gcs("sched", fmt.Sprintf("schedule '%s'", x.sched), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.schedule}", x.sched),
				gcs("conc", fmt.Sprintf("concurrencyPolicy=%s", x.conc), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.concurrencyPolicy}", x.conc),
				gcr("sds", fmt.Sprintf("startingDeadlineSeconds=%s", x.sds), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.startingDeadlineSeconds}", "^"+x.sds+"$"),
				gcr("dl", fmt.Sprintf("job activeDeadlineSeconds=%s", x.deadline), 2,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.jobTemplate.spec.activeDeadlineSeconds}", "^"+x.deadline+"$"),
			},
		))
	}
	return out
}

// ----------------------------------------- headless/named/svc combos

func genP7HeadlessNamed() []*models.Question {
	type v struct {
		ns, svc, pname, sport, affinity string
		headless                        bool
	}
	variants := []v{
		{"ckad-p7hn01", "hn-svc-a", "grpc", "50051", "", true},
		{"ckad-p7hn02", "hn-svc-b", "http", "8080", "ClientIP", false},
		{"ckad-p7hn03", "hn-svc-c", "metrics", "8888", "", true},
		{"ckad-p7hn04", "hn-svc-d", "admin", "9090", "ClientIP", false},
		{"ckad-p7hn05", "hn-svc-e", "rpc", "6000", "", true},
		{"ckad-p7hn06", "hn-svc-f", "ws", "4000", "ClientIP", false},
		{"ckad-p7hn07", "hn-svc-g", "health", "7000", "", true},
		{"ckad-p7hn08", "hn-svc-h", "files", "2121", "ClientIP", false},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		headlessYAML := ""
		taskHead := "a normal ClusterIP Service"
		if x.headless {
			headlessYAML = "  clusterIP: None\n"
			taskHead = "a HEADLESS Service (clusterIP: None)"
		}
		affinityYAML := ""
		taskAff := ""
		if x.affinity != "" {
			affinityYAML = fmt.Sprintf("  sessionAffinity: %s\n", x.affinity)
			taskAff = fmt.Sprintf(" with sessionAffinity=%s", x.affinity)
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
%s%s  selector:
    app: web
  ports:
  - name: %s
    port: %s
    targetPort: %s`, x.svc, x.ns, headlessYAML, affinityYAML, x.pname, x.sport, x.pname)
		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
				"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
			gcs("pname", fmt.Sprintf("Named port %s", x.pname), 2,
				"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].name}", x.pname),
			gcr("sport", fmt.Sprintf("port %s", x.sport), 1,
				"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", "^"+x.sport+"$"),
			gcs("tport", fmt.Sprintf("targetPort name %s", x.pname), 2,
				"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].targetPort}", x.pname),
		}
		if x.headless {
			checks = append(checks, gcs("cip", "clusterIP None", 2,
				"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.clusterIP}", "None"))
		}
		if x.affinity != "" {
			checks = append(checks, gcs("aff", fmt.Sprintf("sessionAffinity=%s", x.affinity), 2,
				"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.sessionAffinity}", x.affinity))
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p7headless-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Service combo: %s", x.svc),
			"Headless discovery and sticky sessions combine with named ports.",
			fmt.Sprintf("In namespace %s, create %s named '%s' selecting app=web: named port '%s' with port %s targeting the NAMED port%s.", x.ns, taskHead, x.svc, x.pname, x.sport, taskAff),
			solution, x.ns,
			genHints(
				"Headless = clusterIP: None (stable DNS per Pod).",
				"targetPort accepts the port's NAME.",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------------- triple probes

func genP7TripleProbes() []*models.Question {
	type v struct {
		ns, pod, hp, rp, sp, grace string
	}
	variants := []v{
		{"ckad-p7tp01", "tri-a", "/live", "8080", "boot.done", "10"},
		{"ckad-p7tp02", "tri-b", "/healthz", "9090", "init.ok", "15"},
		{"ckad-p7tp03", "tri-c", "/alive", "7070", "warm.flag", "20"},
		{"ckad-p7tp04", "tri-d", "/ping", "6060", "ready.now", "25"},
		{"ckad-p7tp05", "tri-e", "/up", "5050", "start.done", "30"},
		{"ckad-p7tp06", "tri-f", "/ok", "4040", "settle.ok", "12"},
		{"ckad-p7tp07", "tri-g", "/well", "3030", "gate.open", "18"},
		{"ckad-p7tp08", "tri-h", "/fine", "2020", "primed.yes", "22"},
		{"ckad-p7tp09", "tri-i", "/good", "1010", "armed.go", "14"},
		{"ckad-p7tp10", "tri-j", "/fit", "8081", "steady.on", "16"},
		{"ckad-p7tp11", "tri-k", "/sound", "8082", "cooked.done", "11"},
		{"ckad-p7tp12", "tri-l", "/valid", "8083", "baked.flag", "13"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  terminationGracePeriodSeconds: %s
  containers:
  - name: app
    image: nginx:1.25
    livenessProbe:
      httpGet: {path: %s, port: 80}
    readinessProbe:
      tcpSocket: {port: %s}
    startupProbe:
      exec:
        command: ['test', '-f', '/tmp/%s']`, x.pod, x.ns, x.grace, x.hp, x.rp, x.sp)
		out = append(out, gq(
			fmt.Sprintf("qg-p7triple-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Three-probe stack on %s", x.pod),
			"liveness, readiness and startup probes each answer a different question.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) with ALL THREE probes: liveness httpGet %s:80, readiness tcpSocket %s, startup exec 'test -f /tmp/%s'; also set terminationGracePeriodSeconds=%s.", x.ns, x.pod, x.hp, x.rp, x.sp, x.grace),
			solution, x.ns,
			genHints(
				"startupProbe gates the other two until it succeeds.",
				"All three live under the container alongside each other.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("live", fmt.Sprintf("Liveness path %s", x.hp), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].livenessProbe.httpGet.path}", x.hp),
				gcr("ready", fmt.Sprintf("Readiness tcp %s", x.rp), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].readinessProbe.tcpSocket.port}", "^"+x.rp+"$"),
				gcs("startup", fmt.Sprintf("Startup exec touches %s", x.sp), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].startupProbe.exec.command[2]}", "/tmp/"+x.sp),
				gcr("grace", fmt.Sprintf("grace=%s", x.grace), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.terminationGracePeriodSeconds}", "^"+x.grace+"$"),
			},
		))
	}
	return out
}

// ------------------------------------------------ multi-source env

func genP7EnvMix() []*models.Question {
	type v struct {
		ns, pod, cm, sec, litKey, litVal string
	}
	variants := []v{
		{"ckad-p7em01", "em-a", "mix-cfg-a", "mix-sec-a", "MODE", "fast"},
		{"ckad-p7em02", "em-b", "mix-cfg-b", "mix-sec-b", "LEVEL", "debug"},
		{"ckad-p7em03", "em-c", "mix-cfg-c", "mix-sec-c", "REGION", "east"},
		{"ckad-p7em04", "em-d", "mix-cfg-d", "mix-sec-d", "POOL", "16"},
		{"ckad-p7em05", "em-e", "mix-cfg-e", "mix-sec-e", "LANG", "go"},
		{"ckad-p7em06", "em-f", "mix-cfg-f", "mix-sec-f", "RETRIES", "5"},
		{"ckad-p7em07", "em-g", "mix-cfg-g", "mix-sec-g", "TIMEOUT", "30"},
		{"ckad-p7em08", "em-h", "mix-cfg-h", "mix-sec-h", "DRYRUN", "false"},
		{"ckad-p7em09", "em-i", "mix-cfg-i", "mix-sec-i", "SHARD", "beta"},
		{"ckad-p7em10", "em-j", "mix-cfg-j", "mix-sec-j", "FLAVOR", "slim"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{Name: "create configmap", CommandArgs: fmt.Sprintf("create configmap %s --from-literal=endpoint=svc.local -n %s", x.cm, x.ns)},
			{Name: "create secret", CommandArgs: fmt.Sprintf("create secret generic %s --from-literal=password=hunter2 -n %s", x.sec, x.ns)},
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']
    env:
    - {name: %s, value: %s}
    - name: NODE_NAME
      valueFrom: {fieldRef: {fieldPath: spec.nodeName}}
    - name: ENDPOINT
      valueFrom: {configMapKeyRef: {name: %s, key: endpoint}}
    - name: PASSWORD
      valueFrom: {secretKeyRef: {name: %s, key: password}}`, x.pod, x.ns, x.litKey, x.litVal, x.cm, x.sec)
		out = append(out, gqp(
			fmt.Sprintf("qg-p7envmix-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Four-way env sourcing in %s", x.pod),
			"One container can mix literal values, fieldRef, ConfigMap and Secret refs.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) with FOUR env vars in this order: %s=%s (literal), NODE_NAME from fieldRef spec.nodeName, ENDPOINT from ConfigMap '%s' key endpoint, PASSWORD from Secret '%s' key password.", x.ns, x.pod, x.litKey, x.litVal, x.cm, x.sec),
			solution, x.ns, prepare,
			genHints(
				"Order matters here — graders check indexes 0..3.",
				"valueFrom takes exactly one source kind.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("e0", fmt.Sprintf("env[0]=%s", x.litKey), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].name}", x.litKey),
				gcs("e1", "env[1]=NODE_NAME via fieldRef", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[1].valueFrom.fieldRef.fieldPath}", "spec.nodeName"),
				gcs("e2", fmt.Sprintf("env[2] from CM %s", x.cm), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[2].valueFrom.configMapKeyRef.name}", x.cm),
				gcs("e3", fmt.Sprintf("env[3] from Secret %s", x.sec), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[3].valueFrom.secretKeyRef.name}", x.sec),
			},
		))
	}
	return out
}

// ------------------------------------------------ paused rollouts

func genP7PausedRollout() []*models.Question {
	type v struct {
		ns, dep string
	}
	variants := []v{
		{"ckad-p7pr01", "pr-checkout"},
		{"ckad-p7pr02", "pr-inventory"},
		{"ckad-p7pr03", "pr-pricing"},
		{"ckad-p7pr04", "pr-search"},
		{"ckad-p7pr05", "pr-recommend"},
		{"ckad-p7pr06", "pr-shipping"},
		{"ckad-p7pr07", "pr-tax"},
		{"ckad-p7pr08", "pr-fraud"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create running deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=3", x.dep, x.ns),
		}}
		solution := fmt.Sprintf(`kubectl rollout pause deployment/%s -n %s
# ... make changes ...
kubectl rollout resume deployment/%s -n %s`, x.dep, x.ns, x.dep, x.ns)
		out = append(out, gqp(
			fmt.Sprintf("qg-p7pause-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Pause & resume %s", x.dep),
			"Pausing a Deployment lets you stack multiple changes into one rollout.",
			fmt.Sprintf("In namespace %s, PAUSE Deployment '%s', then RESUME it again (final state: paused=false, rollout resumed).", x.ns, x.dep),
			solution, x.ns, prepare,
			genHints(
				"'kubectl rollout pause/resume deployment/NAME'.",
				"Resume flips spec.paused back to false.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("resumed", "Deployment is NOT paused", 3,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.paused}", ""),
			},
		))
	}
	return out
}

// ------------------------------------- multi-port typed services

func genP7SvcMultiportType() []*models.Question {
	type v struct {
		ns, svc, stype, p1, t1, p2, t2, aff string
	}
	variants := []v{
		{"ckad-p7sm01", "sm-web", "NodePort", "80", "http", "443", "https", ""},
		{"ckad-p7sm02", "sm-api", "LoadBalancer", "8080", "api", "9090", "admin", "ClientIP"},
		{"ckad-p7sm03", "sm-grpc", "ClusterIP", "50051", "grpc", "50052", "grpc-health", ""},
		{"ckad-p7sm04", "sm-db", "NodePort", "5432", "pg", "9187", "metrics", ""},
		{"ckad-p7sm05", "sm-mq", "LoadBalancer", "5672", "amqp", "15672", "manage", "ClientIP"},
		{"ckad-p7sm06", "sm-cache", "ClusterIP", "6379", "redis", "16379", "cluster", ""},
		{"ckad-p7sm07", "sm-search", "NodePort", "9200", "rest", "9300", "nodes", ""},
		{"ckad-p7sm08", "sm-log", "LoadBalancer", "24224", "fwd", "24220", "health", "ClientIP"},
		{"ckad-p7sm09", "sm-mail", "ClusterIP", "25", "smtp", "587", "submit", ""},
		{"ckad-p7sm10", "sm-dns", "NodePort", "53", "dns-tcp", "53u", "dns-udp", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		affYAML := ""
		taskAff := ""
		if x.aff != "" {
			affYAML = fmt.Sprintf("  sessionAffinity: %s\n", x.aff)
			taskAff = fmt.Sprintf(", sessionAffinity=%s", x.aff)
		}
		t2 := x.t2
		port2 := x.p2
		if x.t2 == "dns-udp" {
			port2 = "53"
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  type: %s
%s%s  selector:
    app: web
  ports:
  - {name: %s, port: %s, targetPort: %s}
  - {name: %s, port: %s, targetPort: %s}`, x.svc, x.ns, x.stype, affYAML, "", x.t1, x.p1, x.t1, t2, port2, t2)
		out = append(out, gq(
			fmt.Sprintf("qg-p7svctype-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Typed multi-port Service %s", x.svc),
			"Multi-port Services require explicit names and can still be sticky.",
			fmt.Sprintf("In namespace %s, create a %s Service '%s' selecting app=web with TWO NAMED ports: '%s' %s->%s and '%s' %s->%s%s.", x.ns, x.stype, x.svc, x.t1, x.p1, x.t1, t2, port2, t2, taskAff),
			solution, x.ns,
			genHints(
				"Multi-port Services REQUIRE names on every port.",
				"type and sessionAffinity sit directly under spec.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcs("type", fmt.Sprintf("type=%s", x.stype), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.type}", x.stype),
				gcs("n1", fmt.Sprintf("Port1 named %s", x.t1), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].name}", x.t1),
				gcs("n2", fmt.Sprintf("Port2 named %s", t2), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[1].name}", t2),
				gcr("p1", fmt.Sprintf("Port1=%s", x.p1), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", "^"+x.p1+"$"),
			},
		))
	}
	return out
}

// ---------------------------------------------- probe+lifecycle

func genP7ProbeLifecycle() []*models.Question {
	type v struct {
		ns, pod, hook, cmd, grace, delay string
	}
	variants := []v{
		{"ckad-p7plc01", "plc-a", "postStart", "touch /tmp/began", "20", "5"},
		{"ckad-p7plc02", "plc-b", "preStop", "rm -f /tmp/lock", "40", "8"},
		{"ckad-p7plc03", "plc-c", "postStart", "mkdir -p /work", "30", "10"},
		{"ckad-p7plc04", "plc-d", "preStop", "sleep 5", "60", "12"},
		{"ckad-p7plc05", "plc-e", "postStart", "echo started > /tmp/log", "25", "6"},
		{"ckad-p7plc06", "plc-f", "preStop", "curl -X POST localhost:8080/drain", "50", "9"},
		{"ckad-p7plc07", "plc-g", "postStart", "cp /etc/motd /tmp/motd", "35", "7"},
		{"ckad-p7plc08", "plc-h", "preStop", "kill -TERM 1", "45", "11"},
		{"ckad-p7plc09", "plc-i", "postStart", "date > /tmp/boot", "55", "4"},
		{"ckad-p7plc10", "plc-j", "preStop", "touch /tmp/stopping", "65", "13"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		cmdParts := strings.SplitN(x.cmd, " ", 2)
		arg2 := ""
		if len(cmdParts) == 2 {
			arg2 = "', '" + cmdParts[1]
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  terminationGracePeriodSeconds: %s
  containers:
  - name: app
    image: nginx:1.25
    lifecycle:
      %s:
        exec:
          command: ['%s'%s']
    livenessProbe:
      httpGet: {path: /, port: 80}
      initialDelaySeconds: %s`, x.pod, x.ns, x.grace, x.hook, cmdParts[0], arg2, x.delay)
		out = append(out, gq(
			fmt.Sprintf("qg-p7probelc-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Hook + probe on %s", x.pod),
			"Lifecycle hooks and probes coexist under one container.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25): %s handler executing '%s', a liveness httpGet (/ :80) with initialDelaySeconds=%s, and terminationGracePeriodSeconds=%s.", x.ns, x.pod, x.hook, x.cmd, x.delay, x.grace),
			solution, x.ns,
			genHints(
				"lifecycle.<hook>.exec.command is argv-style.",
				"initialDelaySeconds delays the FIRST probe only.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("hook", fmt.Sprintf("%s handler present", x.hook), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].lifecycle."+x.hook+"}", ""),
				gcs("probe", "Liveness probe present", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].livenessProbe.httpGet.path}", "/"),
				gcr("delay", fmt.Sprintf("initialDelaySeconds=%s", x.delay), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].livenessProbe.initialDelaySeconds}", "^"+x.delay+"$"),
				gcr("grace", fmt.Sprintf("grace=%s", x.grace), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.terminationGracePeriodSeconds}", "^"+x.grace+"$"),
			},
		))
	}
	return out
}

// ------------------------------------------------ full hardening

func genP7FullHarden() []*models.Question {
	type v struct {
		ns, pod, uid, cap string
	}
	variants := []v{
		{"ckad-p7fh01", "fh-vault", "1500", "NET_ADMIN"},
		{"ckad-p7fh02", "fh-proxy", "2000", "SYS_TIME"},
		{"ckad-p7fh03", "fh-firewall", "2500", "NET_RAW"},
		{"ckad-p7fh04", "fh-monitor", "3000", "IPC_LOCK"},
		{"ckad-p7fh05", "fh-scanner", "3500", "CHOWN"},
		{"ckad-p7fh06", "fh-backup", "4000", "DAC_READ_SEARCH"},
		{"ckad-p7fh07", "fh-router", "4500", "NET_BIND_SERVICE"},
		{"ckad-p7fh08", "fh-clock", "5000", "SYS_TIME"},
		{"ckad-p7fh09", "fh-guard", "5500", "AUDIT_WRITE"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  securityContext:
    runAsUser: %s
    runAsNonRoot: true
    seccompProfile: {type: RuntimeDefault}
  containers:
  - name: app
    image: nginx:1.25
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop: [ALL]
        add: [%s]`, x.pod, x.ns, x.uid, x.cap)
		out = append(out, gq(
			fmt.Sprintf("qg-p7harden-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Maximum hardening: %s", x.pod),
			"Defense in depth stacks UID, non-root, seccomp, caps and read-only FS.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) with: Pod-level runAsUser=%s + runAsNonRoot=true + seccomp RuntimeDefault; container-level allowPrivilegeEscalation=false + readOnlyRootFilesystem=true + drop ALL caps but ADD %s.", x.ns, x.pod, x.uid, x.cap),
			solution, x.ns,
			genHints(
				"drop ALL then add back only what's needed.",
				"runAsNonRoot fails the Pod if the image runs as root.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("uid", fmt.Sprintf("runAsUser=%s", x.uid), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.securityContext.runAsUser}", "^"+x.uid+"$"),
				gcs("nonroot", "runAsNonRoot=true", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.securityContext.runAsNonRoot}", "true"),
				gcs("noesc", "allowPrivilegeEscalation=false", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.allowPrivilegeEscalation}", "false"),
				gcs("drop", "caps drop ALL", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.capabilities.drop[*]}", "ALL"),
				gcs("add", fmt.Sprintf("caps add %s", x.cap), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.capabilities.add[*]}", x.cap),
			},
		))
	}
	return out
}

// ------------------------------------------- three-container pods

func genP7ThreeContainers() []*models.Question {
	type v struct {
		ns, pod, c2, c3, dir string
	}
	variants := []v{
		{"ckad-p7tc01", "tc-pipeline", "producer", "consumer", "/pipe"},
		{"ckad-p7tc02", "tc-stream", "writer", "reader", "/stream"},
		{"ckad-p7tc03", "tc-etl", "extractor", "loader", "/stage"},
		{"ckad-p7tc04", "tc-relay", "receiver", "forwarder", "/buffer"},
		{"ckad-p7tc05", "tc-transform", "encoder", "uploader", "/scratch"},
		{"ckad-p7tc06", "tc-aggregate", "collector", "compressor", "/spool"},
		{"ckad-p7tc07", "tc-chain", "generator", "signer", "/queue"},
		{"ckad-p7tc08", "tc-fanout", "dispatcher", "archiver", "/drops"},
		{"ckad-p7tc09", "tc-mirror", "source", "replica", "/mirror"},
		{"ckad-p7tc10", "tc-gateway", "authenticator", "router", "/tokens"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: main
    image: nginx:1.25
    volumeMounts:
    - {name: share, mountPath: %s}
  - name: %s
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']
    volumeMounts:
    - {name: share, mountPath: %s}
  - name: %s
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']
    volumeMounts:
    - {name: share, mountPath: %s}
  volumes:
  - name: share
    emptyDir: {}`, x.pod, x.ns, x.dir, x.c2, x.dir, x.c3, x.dir)
		out = append(out, gq(
			fmt.Sprintf("qg-p7three-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Three-container pipeline %s", x.pod),
			"Multi-container Pods coordinate through shared volumes.",
			fmt.Sprintf("In namespace %s, create Pod '%s' with THREE containers — main (nginx:1.25), '%s' and '%s' (both busybox:1.36 sleeping forever) — all mounting emptyDir 'share' at %s.", x.ns, x.pod, x.c2, x.c3, x.dir),
			solution, x.ns,
			genHints(
				"Every container needs its own volumeMount.",
				"emptyDir is shared across all containers in the Pod.",
			),
			[]models.Check{
				gcr("names", "Has main, producer-side and consumer-side containers", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}",
					`(^| )main( |.* )`+x.c2+`( |$)|(^| )`+x.c2+`( |.* )main( |$)`),
				gcr("c3", fmt.Sprintf("Container %s present", x.c3), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )`+x.c3+`( |$)`),
				gcs("vol", "Shared volume defined", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[*].name}", "share"),
				gcs("m2", fmt.Sprintf("%s mounts %s", x.c2, x.dir), 1,
					fmt.Sprintf("get pod %s -n %s -o jsonpath={.spec.containers[?(@.name==\"%s\")].volumeMounts[0].mountPath}", x.pod, x.ns, x.c2), x.dir),
			},
		))
	}
	return out
}

// ------------------------------------------- ingress annotations

func genP7IngressAnnot() []*models.Question {
	type v struct {
		ns, ing, ak, av string
	}
	variants := []v{
		{"ckad-p7ia01", "ia-rewrite", "nginx.ingress.kubernetes.io/rewrite-target", "/$2"},
		{"ckad-p7ia02", "ia-body", "nginx.ingress.kubernetes.io/proxy-body-size", "8m"},
		{"ckad-p7ia03", "ia-timeout", "nginx.ingress.kubernetes.io/proxy-read-timeout", "120"},
		{"ckad-p7ia04", "ia-ssl", "nginx.ingress.kubernetes.io/ssl-redirect", "false"},
		{"ckad-p7ia05", "ia-backend", "nginx.ingress.kubernetes.io/backend-protocol", "HTTPS"},
		{"ckad-p7ia06", "ia-limit", "nginx.ingress.kubernetes.io/limit-rps", "10"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create backing service",
			CommandArgs: fmt.Sprintf("create deployment web --image=nginx:1.25 -n %s && expose deployment web --port=80 --name=web-svc -n %s", x.ns, x.ns),
		}}
		solution := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s
  namespace: %s
  annotations:
    %s: "%s"
spec:
  rules:
  - host: app.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: web-svc
            port:
              number: 80`, x.ing, x.ns, x.ak, x.av)
		keyPath := ".metadata.annotations." + x.ak
		if strings.Contains(x.ak, ".") {
			keyPath = fmt.Sprintf(`.metadata.annotations['%s']`, x.ak)
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-p7ingannot-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Annotated Ingress %s", x.ing),
			"Ingress controllers are configured almost entirely through annotations.",
			fmt.Sprintf("In namespace %s (Service 'web-svc' exists), create Ingress '%s' routing host app.example.com (/ Prefix) to web-svc:80, carrying annotation %s=\"%s\".", x.ns, x.ing, x.ak, x.av),
			solution, x.ns, prepare,
			genHints(
				"Annotations live under metadata, not spec.",
				"Values are always strings — quote numbers and booleans.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Ingress %s exists", x.ing), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o name", "ingress.networking.k8s.io/"+x.ing),
				gcs("annot", fmt.Sprintf("Annotation %s", x.ak), 3,
					fmt.Sprintf("get ingress %s -n %s -o jsonpath=%s", x.ing, x.ns, keyPath), x.av),
				gcs("route", "Routes to web-svc", 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[0].backend.service.name}", "web-svc"),
			},
		))
	}
	return out
}

// ------------------------------------------- command + args both

func genP7CmdArgsBoth() []*models.Question {
	type v struct {
		ns, pod, cmd, arg string
	}
	variants := []v{
		{"ckad-p7ca01", "ca-server", "python", "-m http.server 8000"},
		{"ckad-p7ca02", "ca-daemon", "redis-server", "--appendonly yes"},
		{"ckad-p7ca03", "ca-agent", "fluent-bit", "-c /conf/fluent.conf"},
		{"ckad-p7ca04", "ca-worker", "java", "-jar app.jar"},
		{"ckad-p7ca05", "ca-bot", "git", "clone https://repo.dev/x.git"},
		{"ckad-p7ca06", "ca-tool", "tar", "-czf /backup/a.tgz /data"},
		{"ckad-p7ca07", "ca-shell", "bash", "-c echo hi"},
		{"ckad-p7ca08", "ca-probe", "curl", "-s http://localhost:8080/health"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ['%s']
    args: ['%s']`, x.pod, x.ns, x.cmd, x.arg)
		out = append(out, gq(
			fmt.Sprintf("qg-p7cmdargs-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("command + args: %s", x.pod),
			"command overrides ENTRYPOINT; args override CMD.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) running command '%s' with argument '%s'.", x.ns, x.pod, x.cmd, x.arg),
			solution, x.ns,
			genHints(
				"command maps to ENTRYPOINT, args maps to CMD.",
				"Both are lists of strings.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("cmd", fmt.Sprintf("Runs %s", x.cmd), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].command[0]}", x.cmd),
				gcs("arg", fmt.Sprintf("Arg '%s'", x.arg), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].args[0]}", x.arg),
			},
		))
	}
	return out
}

// ------------------------------------------- secret types mix

func genP7SecretTypes() []*models.Question {
	type v struct {
		ns, sec, typ string
	}
	variants := []v{
		{"ckad-p7st01", "dockercfg-a", "docker-registry"},
		{"ckad-p7st02", "basicauth-a", "kubernetes.io/basic-auth"},
		{"ckad-p7st03", "sshauth-a", "kubernetes.io/ssh-auth"},
		{"ckad-p7st04", "tokenauth-a", "kubernetes.io/service-account-token"},
		{"ckad-p7st05", "dockercfg-b", "docker-registry"},
		{"ckad-p7st06", "basicauth-b", "kubernetes.io/basic-auth"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var solution, task string
		switch x.typ {
		case "docker-registry":
			task = fmt.Sprintf("In namespace %s, create a docker-registry Secret named '%s' for server registry.example.com with user 'bot' and password 'tok123'.", x.ns, x.sec)
			solution = fmt.Sprintf("kubectl create secret docker-registry %s --docker-server=registry.example.com --docker-username=bot --docker-password=tok123 -n %s", x.sec, x.ns)
		case "kubernetes.io/basic-auth":
			task = fmt.Sprintf("In namespace %s, create a Secret named '%s' of type kubernetes.io/basic-auth with username admin and password s3cret (stringData).", x.ns, x.sec)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: kubernetes.io/basic-auth
stringData:
  username: admin
  password: s3cret`, x.sec, x.ns)
		case "kubernetes.io/ssh-auth":
			task = fmt.Sprintf("In namespace %s, create a Secret named '%s' of type kubernetes.io/ssh-auth holding key ssh-privatekey (any placeholder value).", x.ns, x.sec)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: kubernetes.io/ssh-auth
stringData:
  ssh-privatekey: fake-key-data`, x.sec, x.ns)
		default:
			task = fmt.Sprintf("In namespace %s, create a Secret named '%s' of type kubernetes.io/service-account-token annotated with kubernetes.io/service-account.name=default.", x.ns, x.sec)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
  annotations:
    kubernetes.io/service-account.name: default
type: kubernetes.io/service-account-token`, x.sec, x.ns)
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p7sectype-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Typed Secret %s", x.sec),
			"Special-purpose Secret types validate their required keys.",
			task, solution, x.ns,
			genHints(
				"basic-auth expects username/password keys.",
				"SA token secrets point at an account via annotation.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Secret %s exists", x.sec), 1,
					"get secret "+x.sec+" -n "+x.ns+" -o name", "secret/"+x.sec),
				gcs("type", fmt.Sprintf("type=%s", x.typ), 3,
					"get secret "+x.sec+" -n "+x.ns+" -o jsonpath={.type}", x.typ),
			},
		))
	}
	return out
}

// --------------------------------- matchExpression deployments

func genP7MatchExprDeploy() []*models.Question {
	type v struct {
		ns, dep, key, op, val string
	}
	variants := []v{
		{"ckad-p7med01", "med-a", "release", "In", "stable"},
		{"ckad-p7med02", "med-b", "channel", "NotIn", "canary"},
		{"ckad-p7med03", "med-c", "hotfix", "Exists", ""},
		{"ckad-p7med04", "med-d", "legacy", "DoesNotExist", ""},
		{"ckad-p7med05", "med-e", "track", "In", "beta"},
		{"ckad-p7med06", "med-f", "sla", "In", "gold"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		valYAML := ""
		valDesc := fmt.Sprintf("operator %s", x.op)
		if x.op == "In" || x.op == "NotIn" {
			valYAML = fmt.Sprintf("\n      values: [%s]", x.val)
			valDesc = fmt.Sprintf("%s [%s]", x.op, x.val)
		}
		labelYAML := ""
		if x.op != "DoesNotExist" {
			labelYAML = fmt.Sprintf("\n        %s: ok", x.key)
		}
		solution := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 2
  selector:
    matchExpressions:
    - key: %s
      operator: %s%s
  template:
    metadata:
      labels:
        %s: ok
    spec:
      containers:
      - name: web
        image: nginx:1.25`, x.dep, x.ns, x.key, x.op, valYAML, x.key)
		_ = labelYAML
		out = append(out, gq(
			fmt.Sprintf("qg-p7matchdep-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Set-based selector: %s", x.dep),
			"Deployment selectors support matchExpressions too.",
			fmt.Sprintf("In namespace %s, create Deployment '%s' (nginx:1.25, 2 replicas) whose SELECTOR uses matchExpressions: key=%s, %s. Template labels must satisfy it.", x.ns, x.dep, x.key, valDesc),
			solution, x.ns,
			genHints(
				"In/NotIn carry a values list; Exists/DoesNotExist don't.",
				"The template labels must satisfy the selector or creation fails.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("key", fmt.Sprintf("Selector key %s", x.key), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.selector.matchExpressions[0].key}", x.key),
				gcs("op", fmt.Sprintf("Operator %s", x.op), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.selector.matchExpressions[0].operator}", x.op),
			},
		))
	}
	return out
}
