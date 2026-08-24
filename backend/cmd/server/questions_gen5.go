package main

import (
	"fmt"
	"strings"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// Bulk expansion pack #1 (~200 questions): probe tuning, resource unit
// drills, envFrom, subPath/emptyDir details, advanced PVCs, Deployment
// lifecycle fields, StatefulSets/DaemonSets, cron schedules, named-port
// services, ipBlock NetworkPolicies, SA binding, matchExpressions,
// node affinity, downward API, lifecycle hooks, pull policies,
// restart policies.

// ---------------------------------------------------- tcp probes

func genP5TcpProbes() []*models.Question {
	type v struct {
		ns, pod, kind, port, period string
	}
	variants := []v{
		{"ckad-p5tcp01", "tcp-live", "livenessProbe", "8080", "10"},
		{"ckad-p5tcp02", "tcp-ready", "readinessProbe", "9090", "15"},
		{"ckad-p5tcp03", "tcp-db", "livenessProbe", "5432", "20"},
		{"ckad-p5tcp04", "tcp-cache", "readinessProbe", "6379", "5"},
		{"ckad-p5tcp05", "tcp-mq", "livenessProbe", "5672", "30"},
		{"ckad-p5tcp06", "tcp-search", "readinessProbe", "9200", "10"},
		{"ckad-p5tcp07", "tcp-mail", "livenessProbe", "25", "60"},
		{"ckad-p5tcp08", "tcp-dns", "readinessProbe", "53", "10"},
		{"ckad-p5tcp09", "tcp-ui", "livenessProbe", "3000", "12"},
		{"ckad-p5tcp10", "tcp-edge", "readinessProbe", "10000", "8"},
		{"ckad-p5tcp11", "tcp-metrics", "livenessProbe", "8888", "25"},
		{"ckad-p5tcp12", "tcp-grpc", "readinessProbe", "50051", "6"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		kindName := "Liveness"
		if x.kind == "readinessProbe" {
			kindName = "Readiness"
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
    %s:
      tcpSocket:
        port: %s
      periodSeconds: %s`, x.pod, x.ns, x.kind, x.port, x.period)
		out = append(out, gq(
			fmt.Sprintf("qg-p5tcp-%02d", i+1), models.DomainApplicationObservability, models.DifficultyMedium,
			fmt.Sprintf("TCP %s on %s", kindName, x.pod),
			"tcpSocket probes verify a listener without an HTTP roundtrip.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) with a %s of type tcpSocket on port %s and periodSeconds=%s.", x.ns, x.pod, kindName, x.port, x.period),
			solution, x.ns,
			genHints(
				"tcpSocket takes just a port (number or name).",
				"periodSeconds controls how often the probe runs.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("port", fmt.Sprintf("Probe port is %s", x.port), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0]."+x.kind+".tcpSocket.port}", "^"+x.port+"$"),
				gcr("period", fmt.Sprintf("periodSeconds=%s", x.period), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0]."+x.kind+".periodSeconds}", "^"+x.period+"$"),
			},
		))
	}
	return out
}

// --------------------------------------------- http probe tuning

func genP5HttpProbeTuning() []*models.Question {
	type v struct {
		ns, dep, path, port, delay, period, failure, success, timeout string
	}
	variants := []v{
		{"ckad-p5hpt01", "tune-a", "/healthz", "80", "10", "5", "3", "1", "1"},
		{"ckad-p5hpt02", "tune-b", "/ready", "8080", "5", "10", "6", "2", "2"},
		{"ckad-p5hpt03", "tune-c", "/status", "80", "15", "20", "2", "1", "3"},
		{"ckad-p5hpt04", "tune-d", "/ping", "9090", "0", "4", "4", "1", "1"},
		{"ckad-p5hpt05", "tune-e", "/live", "80", "20", "30", "1", "1", "5"},
		{"ckad-p5hpt06", "tune-f", "/deep-check", "8443", "8", "6", "5", "2", "2"},
		{"ckad-p5hpt07", "tune-g", "/health", "80", "12", "9", "3", "3", "1"},
		{"ckad-p5hpt08", "tune-h", "/up", "7070", "6", "7", "2", "1", "4"},
		{"ckad-p5hpt09", "tune-i", "/ok", "80", "3", "3", "9", "1", "1"},
		{"ckad-p5hpt10", "tune-j", "/warm", "8080", "18", "12", "4", "2", "3"},
		{"ckad-p5hpt11", "tune-k", "/alive", "80", "9", "8", "3", "1", "2"},
		{"ckad-p5hpt12", "tune-l", "/boot", "9090", "25", "15", "2", "2", "5"},
		{"ckad-p5hpt13", "tune-m", "/meta/health", "80", "7", "11", "6", "1", "1"},
		{"ckad-p5hpt14", "tune-n", "/api/ping", "8000", "4", "5", "3", "2", "2"},
		{"ckad-p5hpt15", "tune-o", "/srv/status", "80", "11", "13", "5", "1", "3"},
		{"ckad-p5hpt16", "tune-p", "/probe", "3000", "14", "9", "4", "3", "2"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.dep, x.ns),
		}}
		solution := fmt.Sprintf(`# edit spec.template.spec.containers[0] then apply
livenessProbe:
  httpGet:
    path: %s
    port: %s
  initialDelaySeconds: %s
  periodSeconds: %s
  failureThreshold: %s
  successThreshold: %s
  timeoutSeconds: %s`, x.path, x.port, x.delay, x.period, x.failure, x.success, x.timeout)
		task := fmt.Sprintf("In namespace %s, add an liveness httpGet probe to Deployment '%s': path %s, port %s, initialDelaySeconds=%s, periodSeconds=%s, failureThreshold=%s, successThreshold=%s, timeoutSeconds=%s.",
			x.ns, x.dep, x.path, x.port, x.delay, x.period, x.failure, x.success, x.timeout)
		base := ".spec.template.spec.containers[0].livenessProbe."
		out = append(out, gqp(
			fmt.Sprintf("qg-p5hpt-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Probe tuning for %s", x.dep),
			"failure/success thresholds and timeouts define how flaky probes behave.",
			task, solution, x.ns, prepare,
			genHints(
				"All knobs live next to httpGet under the probe.",
				"kubectl edit deploy works well for this.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("path", fmt.Sprintf("Probe path %s", x.path), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath="+base+"httpGet.path", x.path),
				gcr("port", fmt.Sprintf("Probe port %s", x.port), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath="+base+"httpGet.port", "^"+x.port+"$"),
				gcr("delay", fmt.Sprintf("initialDelaySeconds=%s", x.delay), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath="+base+"initialDelaySeconds", "^"+x.delay+"$"),
				gcr("period", fmt.Sprintf("periodSeconds=%s", x.period), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath="+base+"periodSeconds", "^"+x.period+"$"),
				gcr("fail", fmt.Sprintf("failureThreshold=%s", x.failure), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath="+base+"failureThreshold", "^"+x.failure+"$"),
				gcr("succ", fmt.Sprintf("successThreshold=%s", x.success), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath="+base+"successThreshold", "^"+x.success+"$"),
				gcr("timeout", fmt.Sprintf("timeoutSeconds=%s", x.timeout), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath="+base+"timeoutSeconds", "^"+x.timeout+"$"),
			},
		))
	}
	return out
}

// ------------------------------------------------ resource combos

func genP5ResourceCombos() []*models.Question {
	type v struct {
		ns, pod, cpuReq, cpuLim, memReq, memLim string
	}
	variants := []v{
		{"ckad-p5res01", "res-a", "150m", "", "", ""},
		{"ckad-p5res02", "res-b", "", "250m", "", ""},
		{"ckad-p5res03", "res-c", "", "", "128Mi", ""},
		{"ckad-p5res04", "res-d", "", "", "", "512Mi"},
		{"ckad-p5res05", "res-e", "0.5", "1", "", ""},
		{"ckad-p5res06", "res-f", "", "", "0.5Gi", "1Gi"},
		{"ckad-p5res07", "res-g", "2000m", "3000m", "1Gi", "2Gi"},
		{"ckad-p5res08", "res-h", "50m", "150m", "32Mi", "96Mi"},
		{"ckad-p5res09", "res-i", "750m", "", "256Mi", ""},
		{"ckad-p5res10", "res-j", "", "400m", "", "384Mi"},
		{"ckad-p5res11", "res-k", "125m", "625m", "64Mi", "320Mi"},
		{"ckad-p5res12", "res-l", "1", "2", "1Gi", "3Gi"},
		{"ckad-p5res13", "res-m", "300m", "300m", "192Mi", "192Mi"},
		{"ckad-p5res14", "res-n", "25m", "75m", "16Mi", "48Mi"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		resLines := ""
		reqPart, limPart := "", ""
		if x.cpuReq != "" || x.memReq != "" {
			if x.cpuReq != "" && x.memReq != "" {
				reqPart = fmt.Sprintf("cpu: %s\n        memory: %s", x.cpuReq, x.memReq)
			} else if x.cpuReq != "" {
				reqPart = fmt.Sprintf("cpu: %s", x.cpuReq)
			} else {
				reqPart = fmt.Sprintf("memory: %s", x.memReq)
			}
		}
		if x.cpuLim != "" || x.memLim != "" {
			if x.cpuLim != "" && x.memLim != "" {
				limPart = fmt.Sprintf("cpu: %s\n        memory: %s", x.cpuLim, x.memLim)
			} else if x.cpuLim != "" {
				limPart = fmt.Sprintf("cpu: %s", x.cpuLim)
			} else {
				limPart = fmt.Sprintf("memory: %s", x.memLim)
			}
		}
		if reqPart != "" {
			resLines += "      requests:\n        " + reqPart + "\n"
		}
		if limPart != "" {
			resLines += "      limits:\n        " + limPart + "\n"
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
    resources:
%s`, x.pod, x.ns, resLines)

		var parts []string
		if x.cpuReq != "" {
			parts = append(parts, "requests.cpu="+x.cpuReq)
		}
		if x.memReq != "" {
			parts = append(parts, "requests.memory="+x.memReq)
		}
		if x.cpuLim != "" {
			parts = append(parts, "limits.cpu="+x.cpuLim)
		}
		if x.memLim != "" {
			parts = append(parts, "limits.memory="+x.memLim)
		}
		task := fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) setting exactly: %s.", x.ns, x.pod, strings.Join(parts, ", "))

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
		}
		w := 1
		if x.cpuReq != "" {
			checks = append(checks, gcr("cpureq", "requests.cpu="+x.cpuReq, 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].resources.requests.cpu}", "^"+x.cpuReq+"$"))
			w += 2
		}
		if x.memReq != "" {
			checks = append(checks, gcr("memreq", "requests.memory="+x.memReq, 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].resources.requests.memory}", "^"+x.memReq+"$"))
			w++
		}
		if x.cpuLim != "" {
			checks = append(checks, gcr("cpulim", "limits.cpu="+x.cpuLim, 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].resources.limits.cpu}", "^"+x.cpuLim+"$"))
			w++
		}
		if x.memLim != "" {
			checks = append(checks, gcr("memlim", "limits.memory="+x.memLim, 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].resources.limits.memory}", "^"+x.memLim+"$"))
			w++
		}

		diff := models.DifficultyHard
		if len(parts) == 1 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p5res-%02d", i+1), models.DomainApplicationEnvironment, diff,
			fmt.Sprintf("Resource drill: %s", x.pod),
			"CPU/memory quantities come in millicores, cores, Mi and Gi — precision matters.",
			task, solution, x.ns,
			genHints(
				"1500m means 1.5 cores; 0.5Gi equals 512Mi.",
				"Only set the fields the task asks for.",
			),
			checks,
		))
		_ = w
	}
	return out
}

// ------------------------------------------------------- envFrom

func genP5EnvFrom() []*models.Question {
	type v struct {
		ns, pod, src, kind string
	}
	variants := []v{
		{"ckad-p5env01", "env-cm-a", "bulk-config-a", "configmap"},
		{"ckad-p5env02", "env-cm-b", "bulk-config-b", "configmap"},
		{"ckad-p5env03", "env-cm-c", "settings-c", "configmap"},
		{"ckad-p5env04", "env-cm-d", "settings-d", "configmap"},
		{"ckad-p5env05", "env-cm-e", "flags-e", "configmap"},
		{"ckad-p5env06", "env-cm-f", "flags-f", "configmap"},
		{"ckad-p5sec01", "env-sec-a", "bulk-secrets-a", "secret"},
		{"ckad-p5sec02", "env-sec-b", "bulk-secrets-b", "secret"},
		{"ckad-p5sec03", "env-sec-c", "creds-c", "secret"},
		{"ckad-p5sec04", "env-sec-d", "creds-d", "secret"},
		{"ckad-p5sec05", "env-sec-e", "tokens-e", "secret"},
		{"ckad-p5sec06", "env-sec-f", "tokens-f", "secret"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		refField := "configMapRef"
		srcKind := "ConfigMap"
		prepCmd := fmt.Sprintf("create configmap %s --from-literal=k=v -n %s", x.src, x.ns)
		if x.kind == "secret" {
			refField = "secretRef"
			srcKind = "Secret"
			prepCmd = fmt.Sprintf("create secret generic %s --from-literal=k=v -n %s", x.src, x.ns)
		}
		prepare := []models.SetupStep{{Name: "create source " + srcKind, CommandArgs: prepCmd}}
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
    envFrom:
    - %s:
        name: %s`, x.pod, x.ns, refField, x.src)
		out = append(out, gqp(
			fmt.Sprintf("qg-p5envfrom-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyMedium,
			fmt.Sprintf("Import all keys from %s", x.src),
			"envFrom injects every key of a ConfigMap/Secret as environment variables.",
			fmt.Sprintf("In namespace %s (%s '%s' already exists), create a Pod named '%s' (busybox:1.36) that imports ALL its keys via envFrom.", x.ns, srcKind, x.src, x.pod),
			solution, x.ns, prepare,
			genHints(
				"envFrom entries use configMapRef or secretRef.",
				"Unlike env[], you don't list individual keys.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("ref", fmt.Sprintf("envFrom references %s", x.src), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].envFrom[0]."+refField+".name}", x.src),
			},
		))
	}
	return out
}

// ------------------------------------------------------- subPath

func genP5SubPath() []*models.Question {
	type v struct {
		ns, pod, vol, sub, mnt string
	}
	variants := []v{
		{"ckad-p5sub01", "sub-app", "data", "db-files", "/var/lib/db"},
		{"ckad-p5sub02", "sub-web", "html", "site", "/usr/share/nginx/html"},
		{"ckad-p5sub03", "sub-log", "logs", "app.log", "/var/log/app"},
		{"ckad-p5sub04", "sub-cache", "cache", "entries", "/cache"},
		{"ckad-p5sub05", "sub-tmp", "scratch", "workdir", "/tmp/work"},
		{"ckad-p5sub06", "sub-cfg", "conf", "app.conf", "/etc/app/app.conf"},
		{"ckad-p5sub07", "sub-up", "uploads", "incoming", "/uploads"},
		{"ckad-p5sub08", "sub-art", "artifacts", "build-42", "/artifacts"},
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
    command: ['sh','-c','sleep 3600']
    volumeMounts:
    - {name: %s, mountPath: %s, subPath: %s}
  volumes:
  - name: %s
    emptyDir: {}`, x.pod, x.ns, x.vol, x.mnt, x.sub, x.vol)
		out = append(out, gq(
			fmt.Sprintf("qg-p5sub-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("subPath mount in %s", x.pod),
			"subPath mounts a single file or folder of a volume instead of the whole thing.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) with an emptyDir volume '%s' mounted at %s using subPath '%s'.", x.ns, x.pod, x.vol, x.mnt, x.sub),
			solution, x.ns,
			genHints(
				"subPath goes on the volumeMount, not the volume.",
				"It's handy to mount one file into an existing directory.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("mnt", fmt.Sprintf("Mounted at %s", x.mnt), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].mountPath}", x.mnt),
				gcs("sub", fmt.Sprintf("subPath is %s", x.sub), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].subPath}", x.sub),
			},
		))
	}
	return out
}

// -------------------------------------------------- emptyDir kinds

func genP5EmptyDirVariants() []*models.Question {
	type v struct {
		ns, pod, vol, medium, limit string
	}
	variants := []v{
		{"ckad-p5ed01", "ed-mem", "ram", "Memory", ""},
		{"ckad-p5ed02", "ed-disk", "disk", "", "1Gi"},
		{"ckad-p5ed03", "ed-small", "tiny", "", "64Mi"},
		{"ckad-p5ed04", "ed-big", "huge", "", "8Gi"},
		{"ckad-p5ed05", "ed-ramlim", "fast", "Memory", "256Mi"},
		{"ckad-p5ed06", "ed-mid", "mid", "", "512Mi"},
		{"ckad-p5ed07", "ed-tinyram", "spark", "Memory", "32Mi"},
		{"ckad-p5ed08", "ed-wide", "wide", "", "4Gi"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		edBody := ""
		if x.medium != "" {
			edBody += "\n    medium: " + x.medium
		}
		if x.limit != "" {
			edBody += "\n    sizeLimit: " + x.limit
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
    - {name: %s, mountPath: /scratch}
  volumes:
  - name: %s
    emptyDir:{}%s`, x.pod, x.ns, x.vol, x.vol, edBody)

		task := fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) with an emptyDir volume '%s' mounted at /scratch", x.ns, x.pod, x.vol)
		if x.medium != "" {
			task += fmt.Sprintf(" with medium=%s", x.medium)
		}
		if x.limit != "" {
			task += fmt.Sprintf(" and sizeLimit=%s", x.limit)
		}
		task += "."

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
			gcs("vol", fmt.Sprintf("Volume %s defined", x.vol), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[*].name}", x.vol),
		}
		if x.medium != "" {
			checks = append(checks, gcs("medium", "medium="+x.medium, 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].emptyDir.medium}", x.medium))
		}
		if x.limit != "" {
			checks = append(checks, gcr("limit", "sizeLimit="+x.limit, 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].emptyDir.sizeLimit}", "^"+x.limit+"$"))
		}

		out = append(out, gq(
			fmt.Sprintf("qg-p5ed-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("emptyDir flavor: %s", x.pod),
			"emptyDir can live in RAM (medium: Memory) and carry a size limit.",
			task, solution, x.ns,
			genHints(
				"medium: Memory counts against the container memory usage.",
				"sizeLimit protects nodes from runaway writes.",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------------- advanced PVCs

func genP5PVCAdvanced() []*models.Question {
	type v struct {
		ns, pvc, size, mode, sc, vmode string
	}
	variants := []v{
		{"ckad-p5pvc01", "fast-claim", "5Gi", "ReadWriteOnce", "standard", "Filesystem"},
		{"ckad-p5pvc02", "shared-claim", "2Gi", "ReadWriteMany", "standard", "Filesystem"},
		{"ckad-p5pvc03", "rox-claim", "1Gi", "ReadOnlyMany", "slow", "Filesystem"},
		{"ckad-p5pvc04", "block-claim", "3Gi", "ReadWriteOnce", "standard", "Block"},
		{"ckad-p5pvc05", "big-block", "20Gi", "ReadWriteOnce", "fast", "Block"},
		{"ckad-p5pvc06", "tiny-claim", "100Mi", "ReadWriteOnce", "standard", "Filesystem"},
		{"ckad-p5pvc07", "rwop-claim", "1Gi", "ReadWriteOncePod", "standard", "Filesystem"},
		{"ckad-p5pvc08", "archive-claim", "50Gi", "ReadWriteOnce", "slow", "Filesystem"},
		{"ckad-p5pvc09", "dev-claim", "500Mi", "ReadWriteOnce", "fast", "Filesystem"},
		{"ckad-p5pvc10", "multi-claim", "7Gi", "ReadWriteMany", "fast", "Filesystem"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
spec:
  accessModes: ['%s']
  storageClassName: %s
  volumeMode: %s
  resources:
    requests:
      storage: %s`, x.pvc, x.ns, x.mode, x.sc, x.vmode, x.size)
		out = append(out, gq(
			fmt.Sprintf("qg-p5pvcadv-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Advanced claim %s", x.pvc),
			"PVCs support storage classes and raw block volumes.",
			fmt.Sprintf("In namespace %s, create a PVC named '%s': size %s, accessMode %s, storageClassName %s, volumeMode %s.", x.ns, x.pvc, x.size, x.mode, x.sc, x.vmode),
			solution, x.ns,
			genHints(
				"storageClassName and volumeMode sit directly under spec.",
				"volumeMode Block skips the filesystem layer.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("PVC %s exists", x.pvc), 1,
					"get pvc "+x.pvc+" -n "+x.ns+" -o name", "persistentvolumeclaim/"+x.pvc),
				gcs("size", fmt.Sprintf("Requests %s", x.size), 1,
					"get pvc "+x.pvc+" -n "+x.ns+" -o jsonpath={.spec.resources.requests.storage}", x.size),
				gcs("mode", fmt.Sprintf("Access mode %s", x.mode), 1,
					"get pvc "+x.pvc+" -n "+x.ns+" -o jsonpath={.spec.accessModes[0]}", x.mode),
				gcs("sc", fmt.Sprintf("storageClassName %s", x.sc), 2,
					"get pvc "+x.pvc+" -n "+x.ns+" -o jsonpath={.spec.storageClassName}", x.sc),
				gcs("vmode", fmt.Sprintf("volumeMode %s", x.vmode), 2,
					"get pvc "+x.pvc+" -n "+x.ns+" -o jsonpath={.spec.volumeMode}", x.vmode),
			},
		))
	}
	return out
}

// ------------------------------------- deployment lifecycle fields

func genP5DepAdvancedFields() []*models.Question {
	type v struct {
		ns, dep, hist, deadline, ready string
	}
	variants := []v{
		{"ckad-p5dep01", "hist-app", "1", "", ""},
		{"ckad-p5dep02", "dl-app", "", "600", ""},
		{"ckad-p5dep03", "ready-app", "", "", "45"},
		{"ckad-p5dep04", "combo-a", "5", "300", "30"},
		{"ckad-p5dep05", "combo-b", "2", "120", "10"},
		{"ckad-p5dep06", "combo-c", "10", "900", "60"},
		{"ckad-p5dep07", "combo-d", "0", "60", "5"},
		{"ckad-p5dep08", "combo-e", "3", "450", "20"},
		{"ckad-p5dep09", "combo-f", "4", "240", "15"},
		{"ckad-p5dep10", "combo-g", "6", "180", "25"},
		{"ckad-p5dep11", "combo-h", "8", "360", "40"},
		{"ckad-p5dep12", "combo-i", "7", "720", "50"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=3", x.dep, x.ns),
		}}
		fields := ""
		taskParts := []string{}
		if x.hist != "" {
			fields += fmt.Sprintf("  revisionHistoryLimit: %s\n", x.hist)
			taskParts = append(taskParts, "revisionHistoryLimit="+x.hist)
		}
		if x.deadline != "" {
			fields += fmt.Sprintf("  progressDeadlineSeconds: %s\n", x.deadline)
			taskParts = append(taskParts, "progressDeadlineSeconds="+x.deadline)
		}
		if x.ready != "" {
			fields += fmt.Sprintf("  minReadySeconds: %s\n", x.ready)
			taskParts = append(taskParts, "minReadySeconds="+x.ready)
		}
		solution := fmt.Sprintf(`# patch or edit spec of the existing Deployment
spec:
%s  replicas: 3
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
        image: nginx:1.25`, fields, x.dep, x.dep)
		task := fmt.Sprintf("In namespace %s, update Deployment '%s' to set %s.", x.ns, x.dep, strings.Join(taskParts, ", "))
		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
				"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
		}
		if x.hist != "" {
			checks = append(checks, gcr("hist", "revisionHistoryLimit="+x.hist, 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.revisionHistoryLimit}", "^"+x.hist+"$"))
		}
		if x.deadline != "" {
			checks = append(checks, gcr("deadline", "progressDeadlineSeconds="+x.deadline, 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.progressDeadlineSeconds}", "^"+x.deadline+"$"))
		}
		if x.ready != "" {
			checks = append(checks, gcr("ready", "minReadySeconds="+x.ready, 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.minReadySeconds}", "^"+x.ready+"$"))
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-p5depfld-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Lifecycle fields on %s", x.dep),
			"Deployments expose history, progress-deadline and min-ready tuning.",
			task, solution, x.ns, prepare,
			genHints(
				"These fields sit directly under spec, sibling to replicas.",
				"'kubectl patch deploy --type=merge -p' also works.",
			),
			checks,
		))
	}
	return out
}

// --------------------------------------------------- statefulsets

func genP5StatefulSets() []*models.Question {
	type v struct {
		ns, sts, svc, img string
		replicas          int
	}
	variants := []v{
		{"ckad-p5sts01", "zk", "zk-hd", "zookeeper:3.9", 3},
		{"ckad-p5sts02", "kafka", "kafka-hd", "kafka:3.7", 3},
		{"ckad-p5sts03", "cass", "cass-hd", "cassandra:5.0", 4},
		{"ckad-p5sts04", "mongo", "mongo-hd", "mongo:7.0", 3},
		{"ckad-p5sts05", "etcd", "etcd-hd", "quay.io/coreos/etcd:v3.5", 5},
		{"ckad-p5sts06", "cockroach", "crdb-hd", "cockroachdb/cockroach:v24.1", 3},
		{"ckad-p5sts07", "rabbit", "rabbit-hd", "rabbitmq:3.13", 3},
		{"ckad-p5sts08", "mysql", "mysql-hd", "mysql:8.4", 2},
		{"ckad-p5sts09", "pg", "pg-hd", "postgres:16.2", 2},
		{"ckad-p5sts10", "redis-cluster", "redis-hd", "redis:7.2", 6},
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
        image: %s`, x.sts, x.ns, x.svc, x.replicas, x.sts, x.sts, x.img)
		out = append(out, gq(
			fmt.Sprintf("qg-p5sts-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("StatefulSet %s (%d replicas)", x.sts, x.replicas),
			"StatefulSets give Pods stable names and ordered rollout via a headless Service.",
			fmt.Sprintf("In namespace %s, create a StatefulSet named '%s' with %d replicas running image '%s', linked to headless Service '%s' via serviceName. Label pods app=%s.", x.ns, x.sts, x.replicas, x.img, x.svc, x.sts),
			solution, x.ns,
			genHints(
				"serviceName must point at a headless Service (clusterIP: None).",
				"Pods get DNS like <pod>.<svc>.<ns>.svc.cluster.local.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("StatefulSet %s exists", x.sts), 2,
					"get statefulset "+x.sts+" -n "+x.ns+" -o name", "statefulset.apps/"+x.sts),
				gcr("replicas", fmt.Sprintf("replicas=%d", x.replicas), 2,
					"get statefulset "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.replicas}", fmt.Sprintf("^%d$", x.replicas)),
				gcs("svc", fmt.Sprintf("serviceName=%s", x.svc), 2,
					"get statefulset "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.serviceName}", x.svc),
				gcs("img", fmt.Sprintf("Image %s", x.img), 1,
					"get statefulset "+x.sts+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.img),
			},
		))
	}
	return out
}

// ------------------------------------------------------ daemonsets

func genP5DaemonSets() []*models.Question {
	type v struct {
		ns, ds, img, key, val string
	}
	variants := []v{
		{"ckad-p5ds01", "log-agent", "fluentd:1.16", "role", "logging"},
		{"ckad-p5ds02", "mon-agent", "node-exporter:v1.8", "role", "monitoring"},
		{"ckad-p5ds03", "net-agent", "cilium:v1.15", "role", "network"},
		{"ckad-p5ds04", "sto-agent", "csi-node:v2", "role", "storage"},
		{"ckad-p5ds05", "sec-agent", "falco:0.38", "role", "security"},
		{"ckad-p5ds06", "gpu-agent", "device-plugin:v0.17", "role", "gpu"},
		{"ckad-p5ds07", "backup-agent", "velero:v1.13", "role", "backup"},
		{"ckad-p5ds08", "trace-agent", "otel-collector:0.99", "role", "tracing"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    matchLabels:
      %s: %s
  template:
    metadata:
      labels:
        %s: %s
    spec:
      containers:
      - name: agent
        image: %s`, x.ds, x.ns, x.key, x.val, x.key, x.val, x.img)
		out = append(out, gq(
			fmt.Sprintf("qg-p5ds-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("DaemonSet %s", x.ds),
			"DaemonSets run one Pod per node — ideal for agents.",
			fmt.Sprintf("In namespace %s, create a DaemonSet named '%s' running image '%s' with pod label %s=%s.", x.ns, x.ds, x.img, x.key, x.val),
			solution, x.ns,
			genHints(
				"No replicas field — scheduling is per-node.",
				"The selector must match the template labels exactly.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("DaemonSet %s exists", x.ds), 2,
					"get daemonset "+x.ds+" -n "+x.ns+" -o name", "daemonset.apps/"+x.ds),
				gcs("img", fmt.Sprintf("Image %s", x.img), 2,
					"get daemonset "+x.ds+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.img),
				gcs("label", fmt.Sprintf("Pod label %s=%s", x.key, x.val), 1,
					"get daemonset "+x.ds+" -n "+x.ns+" -o jsonpath={.spec.template.metadata.labels."+x.key+"}", x.val),
			},
		))
	}
	return out
}

// ------------------------------------------------- cron schedules

func genP5CronSchedules() []*models.Question {
	type v struct {
		ns, name, schedule, desc string
	}
	variants := []v{
		{"ckad-p5cron01", "every-second-min", "*/1 * * * *", "every minute"},
		{"ckad-p5cron02", "every-five", "*/5 * * * *", "every 5 minutes"},
		{"ckad-p5cron03", "hourly-sharp", "0 * * * *", "at minute 0 hourly"},
		{"ckad-p5cron04", "daily-330am", "30 3 * * *", "daily at 03:30"},
		{"ckad-p5cron05", "weekly-sun", "0 1 * * 0", "Sundays 01:00"},
		{"ckad-p5cron06", "monthly-first", "0 0 1 * *", "first of month"},
		{"ckad-p5cron07", "yearly", "0 0 1 1 *", "Jan 1st"},
		{"ckad-p5cron08", "weekdays-noon", "0 12 * * 1-5", "weekdays noon"},
		{"ckad-p5cron09", "quarter-hour", "0,15,30,45 * * * *", "every quarter hour"},
		{"ckad-p5cron10", "business-hours", "0 9-17 * * *", "hourly 09:00-17:00"},
		{"ckad-p5cron11", "shorthand-daily", "@daily", "@daily shorthand"},
		{"ckad-p5cron12", "shorthand-weekly", "@weekly", "@weekly shorthand"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf("kubectl create cronjob %s --image=busybox:1.36 --schedule='%s' -n %s -- /bin/sh -c 'date'",
			x.name, x.schedule, x.ns)
		out = append(out, gq(
			fmt.Sprintf("qg-p5cron-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("Schedule drill: %s", x.desc),
			"Cron syntax fluency: minutes, hours, days, months, weekdays.",
			fmt.Sprintf("In namespace %s, create a CronJob named '%s' (busybox:1.36) with schedule '%s' (%s) running '/bin/sh -c date'.", x.ns, x.name, x.schedule, x.desc),
			solution, x.ns,
			genHints(
				"Order: minute hour day-of-month month day-of-week.",
				"Ranges use '-', steps use '*/', lists use ','.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("CronJob %s exists", x.name), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o name", "cronjob.batch/"+x.name),
				gcs("schedule", fmt.Sprintf("schedule='%s'", x.schedule), 3,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.schedule}", x.schedule),
			},
		))
	}
	return out
}

// -------------------------------------------- named-port services

func genP5NamedPortServices() []*models.Question {
	type v struct {
		ns, pod, svc, portName, cport, sport string
	}
	variants := []v{
		{"ckad-p5np01", "np-web", "np-web-svc", "web", "8080", "80"},
		{"ckad-p5np02", "np-api", "np-api-svc", "api", "9000", "8080"},
		{"ckad-p5np03", "np-grpc", "np-grpc-svc", "grpc", "50051", "50051"},
		{"ckad-p5np04", "np-admin", "np-admin-svc", "admin", "9090", "90"},
		{"ckad-p5np05", "np-metrics", "np-metrics-svc", "metrics", "8888", "88"},
		{"ckad-p5np06", "np-health", "np-health-svc", "health", "7000", "70"},
		{"ckad-p5np07", "np-rpc", "np-rpc-svc", "rpc", "6000", "60"},
		{"ckad-p5np08", "np-ws", "np-ws-svc", "ws", "4000", "40"},
		{"ckad-p5np09", "np-file", "np-file-svc", "files", "2121", "21"},
		{"ckad-p5np10", "np-smtp", "np-smtp-svc", "smtp", "1025", "25"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name: "create pod with named port",
			YAML: fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  labels:
    app: %s
spec:
  containers:
  - name: app
    image: nginx:1.25
    ports:
    - {name: %s, containerPort: %s}`, x.pod, x.pod, x.portName, x.cport),
			Namespace: x.ns,
		}}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: %s
  ports:
  - port: %s
    targetPort: %s`, x.svc, x.ns, x.pod, x.sport, x.portName)
		out = append(out, gqp(
			fmt.Sprintf("qg-p5namedport-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Service via named port %s", x.portName),
			"Services can target container ports BY NAME instead of number.",
			fmt.Sprintf("In namespace %s, Pod '%s' already declares named port '%s' (containerPort %s). Create Service '%s' selecting app=%s with port %s targeting the NAMED port '%s'.", x.ns, x.pod, x.portName, x.cport, x.svc, x.pod, x.sport, x.portName),
			solution, x.ns, prepare,
			genHints(
				"targetPort accepts a string port name.",
				"The name must match the containerPort's name exactly.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcr("sport", fmt.Sprintf("Service port %s", x.sport), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", "^"+x.sport+"$"),
				gcs("tport", fmt.Sprintf("targetPort is name '%s'", x.portName), 3,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].targetPort}", x.portName),
			},
		))
	}
	return out
}

// ------------------------------------------------ ipBlock netpol

func genP5IpBlockNetpol() []*models.Question {
	type v struct {
		ns, np, selKey, selVal, cidr, except string
		hasExcept                            bool
	}
	variants := []v{
		{"ckad-p5ip01", "office-only", "app", "internal", "203.0.113.0/24", "", false},
		{"ckad-p5ip02", "vpn-ingress", "tier", "api", "198.51.100.0/24", "", false},
		{"ckad-p5ip03", "no-lab", "app", "prod", "192.168.0.0/16", "192.168.7.0/24", true},
		{"ckad-p5ip04", "partner-net", "app", "b2b", "10.1.0.0/16", "", false},
		{"ckad-p5ip05", "carve-out", "tier", "edge", "172.16.0.0/12", "172.20.0.0/16", true},
		{"ckad-p5ip06", "corp-range", "role", "gateway", "100.64.0.0/10", "", false},
		{"ckad-p5ip07", "skip-audit", "app", "public", "203.0.114.0/24", "203.0.114.128/25", true},
		{"ckad-p5ip08", "dc-only", "app", "core", "198.18.0.0/15", "", false},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		exceptYAML := ""
		if x.hasExcept {
			exceptYAML = fmt.Sprintf("\n        except:\n        - %s", x.except)
		}
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
    - ipBlock:
        cidr: %s%s`, x.np, x.ns, x.selKey, x.selVal, x.cidr, exceptYAML)

		task := fmt.Sprintf("In namespace %s, create a NetworkPolicy named '%s' allowing ingress to Pods labeled %s=%s ONLY from CIDR %s", x.ns, x.np, x.selKey, x.selVal, x.cidr)
		if x.hasExcept {
			task += fmt.Sprintf(", excluding %s", x.except)
		}
		task += ". Use an ipBlock peer."

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("NetworkPolicy %s exists", x.np), 1,
				"get networkpolicy "+x.np+" -n "+x.ns+" -o name", "networkpolicy.networking.k8s.io/"+x.np),
			gcs("sel", fmt.Sprintf("Selects %s=%s", x.selKey, x.selVal), 1,
				"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.podSelector.matchLabels."+x.selKey+"}", x.selVal),
			gcs("cidr", fmt.Sprintf("ipBlock cidr %s", x.cidr), 3,
				"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.ingress[0].from[0].ipBlock.cidr}", x.cidr),
		}
		if x.hasExcept {
			checks = append(checks, gcs("except", fmt.Sprintf("except %s", x.except), 2,
				"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.ingress[0].from[0].ipBlock.except[0]}", x.except))
		}

		out = append(out, gq(
			fmt.Sprintf("qg-p5ipblock-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("ipBlock policy %s", x.np),
			"NetworkPolicies can filter by source IP ranges with optional carve-outs.",
			task, solution, x.ns,
			genHints(
				"ipBlock replaces podSelector inside the from entry.",
				"except removes sub-ranges from the allowed CIDR.",
			),
			checks,
		))
	}
	return out
}

// --------------------------------------------------- SA binding

func genP5SaBinding() []*models.Question {
	type v struct {
		ns, sa, pod string
	}
	variants := []v{
		{"ckad-p5sa01", "ci-bot", "ci-runner"},
		{"ckad-p5sa02", "deploy-bot", "deployer"},
		{"ckad-p5sa03", "monitor-bot", "monitor-pod"},
		{"ckad-p5sa04", "backup-bot", "backup-pod"},
		{"ckad-p5sa05", "scan-bot", "scanner"},
		{"ckad-p5sa06", "log-bot", "logshipper"},
		{"ckad-p5sa07", "mesh-bot", "mesh-proxy"},
		{"ckad-p5sa08", "audit-bot", "auditor"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create serviceaccount",
			CommandArgs: fmt.Sprintf("create serviceaccount %s -n %s", x.sa, x.ns),
		}}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  serviceAccountName: %s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.sa)
		out = append(out, gqp(
			fmt.Sprintf("qg-p5sabind-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyMedium,
			fmt.Sprintf("Run %s as %s", x.pod, x.sa),
			"Pods adopt an identity through serviceAccountName.",
			fmt.Sprintf("In namespace %s (ServiceAccount '%s' exists), create a Pod named '%s' (busybox:1.36) that runs AS that ServiceAccount.", x.ns, x.sa, x.pod),
			solution, x.ns, prepare,
			genHints(
				"serviceAccountName sits directly under spec.",
				"The default is 'default' when omitted.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("sa", fmt.Sprintf("Runs as %s", x.sa), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.serviceAccountName}", x.sa),
			},
		))
	}
	return out
}

// ---------------------------------------------- matchExpressions

func genP5MatchExpressions() []*models.Question {
	type v struct {
		ns, res, kind, key, op, val string
	}
	variants := []v{
		{"ckad-p5me01", "me-dep-a", "deployment", "tier", "In", "gold"},
		{"ckad-p5me02", "me-dep-b", "deployment", "env", "NotIn", "dev"},
		{"ckad-p5me03", "me-dep-c", "deployment", "canary", "DoesNotExist", ""},
		{"ckad-p5me04", "me-dep-d", "deployment", "region", "In", "eu"},
		{"ckad-p5me05", "me-svc-a", "service", "app", "In", "web"},
		{"ckad-p5me06", "me-svc-b", "service", "tier", "Exists", ""},
		{"ckad-p5me07", "me-svc-c", "service", "role", "In", "api"},
		{"ckad-p5me08", "me-svc-d", "service", "stage", "NotIn", "test"},
		{"ckad-p5me09", "me-sts-a", "statefulset", "shard", "In", "alpha"},
		{"ckad-p5me10", "me-sts-b", "statefulset", "legacy", "DoesNotExist", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		valYAML := ""
		valDesc := fmt.Sprintf("operator %s", x.op)
		if x.op == "In" || x.op == "NotIn" {
			valYAML = fmt.Sprintf("\n        values: [%s]", x.val)
			valDesc = fmt.Sprintf("%s [%s]", x.op, x.val)
		}
		var head, tmplSel string
		switch x.kind {
		case "deployment":
			head = fmt.Sprintf(`apiVersion: apps/v1
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
        %s: matched
    spec:
      containers:
      - name: web
        image: nginx:1.25`, x.res, x.ns, x.key, x.op, valYAML, x.key)
			tmplSel = "selector"
		case "service":
			head = fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    matchExpressions:
    - key: %s
      operator: %s%s
  ports:
  - port: 80`, x.res, x.ns, x.key, x.op, valYAML)
			tmplSel = "selector"
		default:
			head = fmt.Sprintf(`apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: %s
  namespace: %s
spec:
  serviceName: %s-hd
  replicas: 1
  selector:
    matchExpressions:
    - key: %s
      operator: %s%s
  template:
    metadata:
      labels:
        %s: matched
    spec:
      containers:
      - name: web
        image: nginx:1.25`, x.res, x.ns, x.res, x.key, x.op, valYAML, x.key)
			tmplSel = "selector"
		}
		_ = tmplSel
		kindPath := map[string]string{"deployment": "deploy", "service": "svc", "statefulset": "sts"}[x.kind]
		kindFull := map[string]string{"deployment": "deployment.apps/", "service": "service/", "statefulset": "statefulset.apps/"}[x.kind]
		out = append(out, gq(
			fmt.Sprintf("qg-p5matchexp-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("%s selector via matchExpressions", strings.Title(x.kind)),
			"set-based selectors (matchExpressions) go beyond simple equality.",
			fmt.Sprintf("In namespace %s, create a %s named '%s' whose selector uses matchExpressions: key=%s, %s. Give the workload/template matching labels so it is valid.", x.ns, x.kind, x.res, x.key, valDesc),
			head, x.ns,
			genHints(
				"In/NotIn require a values list; Exists/DoesNotExist don't.",
				"For Deployments/STS the template labels must satisfy the selector.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("%s %s exists", x.kind, x.res), 1,
					"get "+kindPath+" "+x.res+" -n "+x.ns+" -o name", kindFull+x.res),
				gcs("key", fmt.Sprintf("Selector key %s", x.key), 2,
					fmt.Sprintf("get %s %s -n %s -o jsonpath={.spec.selector.matchExpressions[0].key}", kindPath, x.res, x.ns), x.key),
				gcs("op", fmt.Sprintf("Operator %s", x.op), 2,
					fmt.Sprintf("get %s %s -n %s -o jsonpath={.spec.selector.matchExpressions[0].operator}", kindPath, x.res, x.ns), x.op),
			},
		))
	}
	return out
}

// ------------------------------------------------- node affinity

func genP5NodeAffinity() []*models.Question {
	type v struct {
		ns, pod, key, op, val string
	}
	variants := []v{
		{"ckad-p5na01", "aff-gpu", "accelerator", "In", "nvidia"},
		{"ckad-p5na02", "aff-ssd", "disktype", "In", "ssd"},
		{"ckad-p5na03", "aff-arm", "kubernetes.io/arch", "In", "arm64"},
		{"ckad-p5na04", "aff-linux", "kubernetes.io/os", "In", "linux"},
		{"ckad-p5na05", "aff-zone", "topology.kubernetes.io/zone", "In", "west"},
		{"ckad-p5na06", "aff-spot", "nodepool", "In", "spot"},
		{"ckad-p5na07", "aff-nogpu", "accelerator", "NotIn", "nvidia"},
		{"ckad-p5na08", "aff-big", "node-size", "In", "xlarge"},
		{"ckad-p5na09", "aff-edge", "location", "In", "edge-1"},
		{"ckad-p5na10", "aff-bare", "instance-type", "In", "bare-metal"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
        - matchExpressions:
          - key: %s
            operator: %s
            values: [%s]
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.key, x.op, x.val)
		out = append(out, gq(
			fmt.Sprintf("qg-p5nodeaff-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Node affinity for %s", x.pod),
			"required nodeAffinity expresses rich node constraints.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) with REQUIRED node affinity: key=%s, operator=%s, values=[%s].", x.ns, x.pod, x.key, x.op, x.val),
			solution, x.ns,
			genHints(
				"The long path is spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.",
				"values is always a list, even with one entry.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("key", fmt.Sprintf("Affinity key %s", x.key), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].key}", x.key),
				gcs("op", fmt.Sprintf("Operator %s", x.op), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].operator}", x.op),
				gcs("val", fmt.Sprintf("Value %s", x.val), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[0].matchExpressions[0].values[0]}", x.val),
			},
		))
	}
	return out
}

// ------------------------------------------------- downward API

func genP5DownwardAPI() []*models.Question {
	type v struct {
		ns, pod, envKey, fieldPath string
	}
	variants := []v{
		{"ckad-p5dw01", "dw-name", "POD_NAME", "metadata.name"},
		{"ckad-p5dw02", "dw-ns", "POD_NAMESPACE", "metadata.namespace"},
		{"ckad-p5dw03", "dw-ip", "POD_IP", "status.podIP"},
		{"ckad-p5dw04", "dw-node", "NODE_NAME", "spec.nodeName"},
		{"ckad-p5dw05", "dw-sa", "SERVICE_ACCOUNT", "spec.serviceAccountName"},
		{"ckad-p5dw06", "dw-host", "HOST_IP", "status.hostIP"},
		{"ckad-p5dw07", "dw-label", "APP_LABEL", "metadata.labels['app']"},
		{"ckad-p5dw08", "dw-annot", "BUILD_ANNOT", "metadata.annotations['build']"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		extra := ""
		if strings.Contains(x.fieldPath, "[") {
			if strings.Contains(x.fieldPath, "labels") {
				extra = "\n  labels:\n    app: demo"
			} else {
				extra = "\n  annotations:\n    build: \"42\""
			}
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s%s
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']
    env:
    - name: %s
      valueFrom:
        fieldRef:
          fieldPath: %s`, x.pod, x.ns, extra, x.envKey, x.fieldPath)
		out = append(out, gq(
			fmt.Sprintf("qg-p5downward-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Expose %s via downward API", x.envKey),
			"The downward API projects Pod metadata into the container.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) exposing field '%s' as environment variable %s via valueFrom.fieldRef.", x.ns, x.pod, x.fieldPath, x.envKey),
			solution, x.ns,
			genHints(
				"valueFrom.fieldRef.fieldPath carries the path.",
				"Label/annotation paths use metadata.labels['key'] syntax.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("env", fmt.Sprintf("Env %s defined", x.envKey), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].name}", x.envKey),
				gcs("field", fmt.Sprintf("fieldPath %s", x.fieldPath), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].valueFrom.fieldRef.fieldPath}", x.fieldPath),
			},
		))
	}
	return out
}

// ----------------------------------------------- lifecycle hooks

func genP5LifecycleHooks() []*models.Question {
	type v struct {
		ns, pod, hook, htype, arg1, arg2 string
	}
	variants := []v{
		{"ckad-p5lc01", "lc-post-exec", "postStart", "exec", "touch", "/tmp/started"},
		{"ckad-p5lc02", "lc-pre-exec", "preStop", "exec", "rm", "-f /tmp/lock"},
		{"ckad-p5lc03", "lc-post-http", "postStart", "http", "/warmup", "8080"},
		{"ckad-p5lc04", "lc-pre-http", "preStop", "http", "/drain", "8080"},
		{"ckad-p5lc05", "lc-post-mk", "postStart", "exec", "mkdir", "-p /work"},
		{"ckad-p5lc06", "lc-pre-sleep", "preStop", "exec", "sleep", "10"},
		{"ckad-p5lc07", "lc-post-flag", "postStart", "exec", "echo", "hi > /tmp/x"},
		{"ckad-p5lc08", "lc-pre-url", "preStop", "http", "/shutdown", "9090"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		hookBody := ""
		hookDesc := ""
		if x.htype == "exec" {
			hookBody = fmt.Sprintf("      exec:\n        command: ['%s', '%s']", x.arg1, x.arg2)
			hookDesc = fmt.Sprintf("exec '%s %s'", x.arg1, x.arg2)
		} else {
			hookBody = fmt.Sprintf("      httpGet:\n        path: %s\n        port: %s", x.arg1, x.arg2)
			hookDesc = fmt.Sprintf("httpGet %s:%s", x.arg1, x.arg2)
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
    lifecycle:
      %s:
%s`, x.pod, x.ns, x.hook, hookBody)
		out = append(out, gq(
			fmt.Sprintf("qg-p5lifecycle-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("%s hook on %s", x.hook, x.pod),
			"Lifecycle hooks fire right after start or before termination.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) with a %s handler of type %s (%s).", x.ns, x.pod, x.hook, x.htype, hookDesc),
			solution, x.ns,
			genHints(
				"lifecycle.<hook> supports exec and httpGet handlers.",
				"preStop runs before SIGTERM grace countdown ends.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("hook", fmt.Sprintf("%s handler present", x.hook), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].lifecycle."+x.hook+"}", ""),
			},
		))
	}
	return out
}

// ------------------------------------------- pull policies/secrets

func genP5PullPolicySecrets() []*models.Question {
	type v struct {
		ns, pod, policy, secret string
	}
	variants := []v{
		{"ckad-p5pp01", "pp-always", "Always", ""},
		{"ckad-p5pp02", "pp-ifnot", "IfNotPresent", ""},
		{"ckad-p5pp03", "pp-never", "Never", ""},
		{"ckad-p5pp04", "pp-always2", "Always", ""},
		{"ckad-p5ips01", "ips-reg", "", "regcred"},
		{"ckad-p5ips02", "ips-gcr", "", "gcr-key"},
		{"ckad-p5ips03", "ips-acr", "", "acr-auth"},
		{"ckad-p5ips04", "ips-ghcr", "", "ghcr-token"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var solution, task string
		var checks []models.Check
		if x.policy != "" {
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) whose container sets imagePullPolicy=%s.", x.ns, x.pod, x.policy)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: nginx:1.25
    imagePullPolicy: %s`, x.pod, x.ns, x.policy)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("policy", fmt.Sprintf("imagePullPolicy=%s", x.policy), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].imagePullPolicy}", x.policy),
			}
		} else {
			prepare := []models.SetupStep{{
				Name:        "create docker-registry secret",
				CommandArgs: fmt.Sprintf("create secret docker-registry %s --docker-server=example.io --docker-username=u --docker-password=p -n %s", x.secret, x.ns),
			}}
			task = fmt.Sprintf("In namespace %s (docker-registry Secret '%s' exists), create a Pod named '%s' (nginx:1.25) that pulls images USING that secret.", x.ns, x.secret, x.pod)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  imagePullSecrets:
  - name: %s
  containers:
  - name: app
    image: nginx:1.25`, x.pod, x.ns, x.secret)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("ips", fmt.Sprintf("Uses imagePullSecret %s", x.secret), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.imagePullSecrets[0].name}", x.secret),
			}
			out = append(out, gqp(fmt.Sprintf("qg-p5pull-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
				fmt.Sprintf("Private registry access for %s", x.pod),
				"imagePullSecrets authenticate private registry pulls.",
				task, solution, x.ns, prepare,
				genHints(
					"imagePullSecrets lives under Pod spec, not the container.",
					"Create it with 'kubectl create secret docker-registry'.",
				),
				checks,
			))
			continue
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p5pull-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyEasy,
			fmt.Sprintf("Pull policy on %s", x.pod),
			"imagePullPolicy decides when kubelet re-pulls an image.",
			task, solution, x.ns,
			genHints(
				"Always re-pulls every time; Never uses local only.",
				"The field sits on the container.",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------------ restart policies

func genP5RestartPolicies() []*models.Question {
	type v struct {
		ns, pod, policy string
	}
	variants := []v{
		{"ckad-p5rp01", "rp-onfail", "OnFailure"},
		{"ckad-p5rp02", "rp-never", "Never"},
		{"ckad-p5rp03", "rp-onfail2", "OnFailure"},
		{"ckad-p5rp04", "rp-never2", "Never"},
		{"ckad-p5rp05", "rp-onfail3", "OnFailure"},
		{"ckad-p5rp06", "rp-always", "Always"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf("kubectl run %s --image=busybox:1.36 -n %s --restart=%s --command -- sleep 3600",
			x.pod, x.ns, x.policy)
		out = append(out, gq(
			fmt.Sprintf("qg-p5restart-%02d", i+1), models.DomainApplicationDesign, models.DifficultyEasy,
			fmt.Sprintf("restartPolicy=%s on %s", x.policy, x.pod),
			"restartPolicy governs what kubelet does when the container exits.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) with restartPolicy=%s.", x.ns, x.pod, x.policy),
			solution, x.ns,
			genHints(
				"'kubectl run --restart=' maps straight to spec.restartPolicy.",
				"Jobs require Never or OnFailure.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("policy", fmt.Sprintf("restartPolicy=%s", x.policy), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.restartPolicy}", x.policy),
			},
		))
	}
	return out
}
