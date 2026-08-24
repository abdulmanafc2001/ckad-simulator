package main

import (
	"fmt"
	"strings"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// Bulk expansion pack #2 (~254 questions): UDP ports, hostPath, init
// chains, arg interpolation, quota scopes, annotations, job patterns,
// probe thresholds, odd units, multi-label workloads, rollout-to-revision,
// expose/set drills, immutable ConfigMaps, stringData secrets, hostAliases,
// DNS settings, grace periods, named ports, shareProcessNamespace, plus
// expanded simple families.

// ---------------------------------------------------- UDP ports

func genP6UdpPorts() []*models.Question {
	type v struct {
		ns, pod, port string
	}
	variants := []v{
		{"ckad-p6udp01", "dns-udp", "53"},
		{"ckad-p6udp02", "syslog-udp", "514"},
		{"ckad-p6udp03", "ntp-udp", "123"},
		{"ckad-p6udp04", "snmp-udp", "161"},
		{"ckad-p6udp05", "dhcp-udp", "67"},
		{"ckad-p6udp06", "tftp-udp", "69"},
		{"ckad-p6udp07", "radius-udp", "1812"},
		{"ckad-p6udp08", "statsd-udp", "8125"},
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
    ports:
    - {containerPort: %s, protocol: UDP}`, x.pod, x.ns, x.port)
		out = append(out, gq(
			fmt.Sprintf("qg-p6udp-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("UDP port %s on %s", x.port, x.pod),
			"Container ports default to TCP; UDP must be declared explicitly.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) exposing containerPort %s with protocol UDP.", x.ns, x.pod, x.port),
			solution, x.ns,
			genHints(
				"protocol: UDP goes inside the ports entry.",
				"Omitting protocol means TCP.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("port", fmt.Sprintf("containerPort %s", x.port), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].containerPort}", "^"+x.port+"$"),
				gcs("proto", "protocol is UDP", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].protocol}", "UDP"),
			},
		))
	}
	return out
}

// ---------------------------------------------------- hostPath

func genP6HostPath() []*models.Question {
	type v struct {
		ns, pod, vol, hpath, htype, mnt string
	}
	variants := []v{
		{"ckad-p6hp01", "hp-log", "hostlogs", "/var/log", "Directory", "/host-log"},
		{"ckad-p6hp02", "hp-time", "hosttime", "/etc/localtime", "File", "/etc/localtime"},
		{"ckad-p6hp03", "hp-data", "hostdata", "/mnt/data", "DirectoryOrCreate", "/data"},
		{"ckad-p6hp04", "hp-cfg", "hostcfg", "/etc/config", "Directory", "/etc/host-cfg"},
		{"ckad-p6hp05", "hp-sock", "hostsock", "/var/run/docker.sock", "Socket", "/var/run/docker.sock"},
		{"ckad-p6hp06", "hp-proc", "hostproc", "/proc", "Directory", "/host-proc"},
		{"ckad-p6hp07", "hp-sys", "hostsys", "/sys", "Directory", "/host-sys"},
		{"ckad-p6hp08", "hp-opt", "hostopt", "/opt/shared", "DirectoryOrCreate", "/shared"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		typeLine := ""
		if x.htype != "" {
			typeLine = "\n    type: " + x.htype
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
    - {name: %s, mountPath: %s}
  volumes:
  - name: %s
    hostPath:
      path: %s%s`, x.pod, x.ns, x.vol, x.mnt, x.vol, x.hpath, typeLine)
		task := fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) mounting hostPath %s at %s", x.ns, x.pod, x.hpath, x.mnt)
		if x.htype != "" {
			task += fmt.Sprintf(" with type %s", x.htype)
		}
		task += "."
		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
			gcs("path", fmt.Sprintf("hostPath %s", x.hpath), 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].hostPath.path}", x.hpath),
			gcs("mnt", fmt.Sprintf("Mounted at %s", x.mnt), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].mountPath}", x.mnt),
		}
		if x.htype != "" {
			checks = append(checks, gcs("type", "type "+x.htype, 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].hostPath.type}", x.htype))
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p6hostpath-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("hostPath volume in %s", x.pod),
			"hostPath mounts node filesystem paths into Pods.",
			task, solution, x.ns,
			genHints(
				"type guards against unexpected node state.",
				"DirectoryOrCreate creates the path if absent.",
			),
			checks,
		))
	}
	return out
}

// --------------------------------------------- init chains

func genP6MultiInitChains() []*models.Question {
	type v struct {
		ns, pod, i1, i2, f1, f2 string
	}
	variants := []v{
		{"ckad-p6mic01", "chain-a", "wait-db", "migrate", "db.ready", "schema.done"},
		{"ckad-p6mic02", "chain-b", "fetch-config", "validate", "config.json", "valid.flag"},
		{"ckad-p6mic03", "chain-c", "warm-cache", "ping-deps", "cache.warm", "deps.ok"},
		{"ckad-p6mic04", "chain-d", "provision-vol", "seed-data", "vol.ready", "seeded"},
		{"ckad-p6mic05", "chain-e", "register-svc", "health-gate", "registered", "healthy"},
		{"ckad-p6mic06", "chain-f", "pull-model", "quantize", "model.bin", "model.q"},
		{"ckad-p6mic07", "chain-g", "gen-cert", "install-cert", "cert.pem", "installed"},
		{"ckad-p6mic08", "chain-h", "lock-acquire", "snapshot", "locked", "snapshot.done"},
		{"ckad-p6mic09", "chain-i", "check-quota", "reserve-slot", "quota.ok", "reserved"},
		{"ckad-p6mic10", "chain-j", "init-mesh", "inject-proxy", "mesh.cfg", "proxy.ok"},
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
    command: ['sh','-c','touch /work/%s']
    volumeMounts:
    - {name: shared, mountPath: /work}
  - name: %s
    image: busybox:1.36
    command: ['sh','-c','touch /work/%s']
    volumeMounts:
    - {name: shared, mountPath: /work}
  containers:
  - name: main
    image: nginx:1.25
    volumeMounts:
    - {name: shared, mountPath: /usr/share/nginx/html}
  volumes:
  - name: shared
    emptyDir: {}`, x.pod, x.ns, x.i1, x.f1, x.i2, x.f2)
		out = append(out, gq(
			fmt.Sprintf("qg-p6initchain-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Init chain for %s", x.pod),
			"Init containers run sequentially in declaration order.",
			fmt.Sprintf("In namespace %s, create Pod '%s' with TWO init containers '%s' then '%s' (both busybox:1.36) that touch /work/%s and /work/%s respectively on shared emptyDir 'shared', plus main container 'main' (nginx:1.25) mounting it at /usr/share/nginx/html.", x.ns, x.pod, x.i1, x.i2, x.f1, x.f2),
			solution, x.ns,
			genHints(
				"Order in initContainers defines execution order.",
				"Each init container needs its own volumeMount.",
			),
			[]models.Check{
				gcr("i1", fmt.Sprintf("First init %s", x.i1), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.initContainers[0].name}", "^"+x.i1+"$"),
				gcr("i2", fmt.Sprintf("Second init %s", x.i2), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.initContainers[1].name}", "^"+x.i2+"$"),
				gcs("main", "Main container main exists", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].name}", "main"),
				gcs("vol", "Shared volume defined", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[*].name}", "shared"),
			},
		))
	}
	return out
}

// ------------------------------------------ arg interpolation

func genP6ArgInterpolation() []*models.Question {
	type v struct {
		ns, pod, envKey, envVal, arg string
	}
	variants := []v{
		{"ckad-p6arg01", "interp-a", "TARGET", "web.local", "$(TARGET)"},
		{"ckad-p6arg02", "interp-b", "PORT_NUM", "9090", "$(PORT_NUM)"},
		{"ckad-p6arg03", "interp-c", "MODE", "batch", "--mode=$(MODE)"},
		{"ckad-p6arg04", "interp-d", "ENDPOINT", "api.svc", "$(ENDPOINT)"},
		{"ckad-p6arg05", "interp-e", "WORKERS", "4", "--workers=$(WORKERS)"},
		{"ckad-p6arg06", "interp-f", "LOGDIR", "/var/log/app", "$(LOGDIR)"},
		{"ckad-p6arg07", "interp-g", "RETRIES", "7", "--retries=$(RETRIES)"},
		{"ckad-p6arg08", "interp-h", "REGION_ID", "eu-central", "$(REGION_ID)"},
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
    env:
    - name: %s
      value: %s
    args: ['%s']`, x.pod, x.ns, x.envKey, x.envVal, x.arg)
		out = append(out, gq(
			fmt.Sprintf("qg-p6argint-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Interpolate $(%s) in %s", x.envKey, x.pod),
			"Container args can reference env vars with $(VAR) syntax.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) defining env %s=%s and an args entry that references it as %s.", x.ns, x.pod, x.envKey, x.envVal, x.arg),
			solution, x.ns,
			genHints(
				"$(VAR) expansion happens at runtime, unlike shell $VAR.",
				"The env var must be declared in the same container.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("envval", fmt.Sprintf("%s=%s", x.envKey, x.envVal), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].value}", x.envVal),
				gcs("argref", fmt.Sprintf("args reference %s", x.arg), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].args[0]}", x.arg),
			},
		))
	}
	return out
}

// ------------------------------------------- quota scopes

func genP6QuotaScopes() []*models.Question {
	type v struct {
		ns, q, scope, extra string
	}
	variants := []v{
		{"ckad-p6qs01", "besteffort-q", "BestEffort", ""},
		{"ckad-p6qs02", "notbesteffort-q", "NotBestEffort", "pods=10"},
		{"ckad-p6qs03", "terminating-q", "Terminating", "pods=5"},
		{"ckad-p6qs04", "nonterminating-q", "NotTerminating", ""},
		{"ckad-p6qs05", "priority-q", "PriorityClass", "pods=8"},
		{"ckad-p6qs06", "crossns-q", "CrossNamespacePodAffinity", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		hardYAML := ""
		if x.extra != "" {
			kv := strings.SplitN(x.extra, "=", 2)
			hardYAML = fmt.Sprintf("  hard:\n    %s: \"%s\"\n", kv[0], kv[1])
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: ResourceQuota
metadata:
  name: %s
  namespace: %s
spec:
  scopes: [%s]
%s`, x.q, x.ns, x.scope, hardYAML)
		out = append(out, gq(
			fmt.Sprintf("qg-p6quotascope-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Scoped quota %s", x.q),
			"ResourceQuota scopes limit which objects the quota applies to.",
			fmt.Sprintf("In namespace %s, create a ResourceQuota named '%s' with scope [%s]%s.", x.ns, x.q, x.scope, extraDesc(x.extra)),
			solution, x.ns,
			genHints(
				"scopes is a list directly under spec.",
				"BestEffort matches Pods without any requests/limits.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("ResourceQuota %s exists", x.q), 1,
					"get resourcequota "+x.q+" -n "+x.ns+" -o name", "resourcequota/"+x.q),
				gcs("scope", fmt.Sprintf("Scope %s", x.scope), 3,
					"get resourcequota "+x.q+" -n "+x.ns+" -o jsonpath={.spec.scopes[0]}", x.scope),
			},
		))
	}
	return out
}

func extraDesc(extra string) string {
	if extra == "" {
		return ""
	}
	kv := strings.SplitN(extra, "=", 2)
	return fmt.Sprintf(" and hard %s=%s", kv[0], kv[1])
}

// --------------------------------------------- pod annotations

func genP6PodAnnotations() []*models.Question {
	type v struct {
		ns, pod, key, val string
	}
	variants := []v{
		{"ckad-p6an01", "an-build", "build.number", "2024"},
		{"ckad-p6an02", "an-owner", "owner.team", "platform"},
		{"ckad-p6an03", "an-link", "docs.url", "https://wiki.internal/app"},
		{"ckad-p6an04", "an-hash", "git.sha", "abc123def"},
		{"ckad-p6an05", "an-tier", "support.tier", "gold"},
		{"ckad-p6an06", "an-alert", "alerting.enabled", "true"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`kubectl run %s --image=busybox:1.36 -n %s --command -- sleep 3600
kubectl annotate pod %s %s=%s -n %s`, x.pod, x.ns, x.pod, x.key, x.val, x.ns)
		out = append(out, gq(
			fmt.Sprintf("qg-p6annot-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("Annotate %s", x.pod),
			"Annotations carry arbitrary non-identifying metadata.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) carrying annotation %s=%s.", x.ns, x.pod, x.key, x.val),
			solution, x.ns,
			genHints(
				"'kubectl annotate' adds metadata without affecting scheduling.",
				"In YAML it lives under metadata.annotations.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("annot", fmt.Sprintf("Annotation %s=%s", x.key, x.val), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.metadata.annotations."+x.key+"}", x.val),
			},
		))
	}
	return out
}

// --------------------------------------------- job patterns

func genP6JobPatterns() []*models.Question {
	type v struct {
		ns, name, img, cmd, comp, par string
	}
	variants := []v{
		{"ckad-p6job01", "render-job", "blender:4.1", "blender -b scene.blend", "", ""},
		{"ckad-p6job02", "migrate-job", "flyway:10", "flyway migrate", "", ""},
		{"ckad-p6job03", "index-job", "search-indexer:2", "index --full", "3", "2"},
		{"ckad-p6job04", "report-job", "python:3.12", "python report.py", "", ""},
		{"ckad-p6job05", "backup-job", "restic:0.16", "restic backup /data", "", ""},
		{"ckad-p6job06", "scan-job", "trivy:0.50", "trivy fs /src", "2", "1"},
		{"ckad-p6job07", "train-job", "pytorch:2.3", "python train.py --epochs 1", "1", "1"},
		{"ckad-p6job08", "etl-job", "spark:3.5", "submit etl.py", "4", "4"},
		{"ckad-p6job09", "notify-job", "curlimages/curl:8.7", "curl -X POST hook", "", ""},
		{"ckad-p6job10", "cleanup-job", "alpine:3.19", "rm -rf /tmp/cache", "", ""},
		{"ckad-p6job11", "verify-job", "golang:1.22", "go test ./...", "2", "2"},
		{"ckad-p6job12", "publish-job", "node:20", "npm publish", "", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		specExtra := ""
		if x.comp != "" {
			specExtra = fmt.Sprintf("  completions: %s\n  parallelism: %s\n", x.comp, x.par)
		}
		solution := fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: %s
spec:
%sextra-tmpl`, x.name, x.ns, specExtra)
		solution = strings.Replace(solution, "extra-tmpl", `  template:
    spec:
      restartPolicy: Never
      containers:
      - name: work
        image: `+x.img+`
        command: ['sh','-c','`+x.cmd+`']`, 1)

		task := fmt.Sprintf("In namespace %s, create a Job named '%s' running image '%s' with command '%s'", x.ns, x.name, x.img, x.cmd)
		if x.comp != "" {
			task += fmt.Sprintf(", completions=%s and parallelism=%s", x.comp, x.par)
		}
		task += "."

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Job %s exists", x.name), 1,
				"get job "+x.name+" -n "+x.ns+" -o name", "job.batch/"+x.name),
			gcs("img", fmt.Sprintf("Image %s", x.img), 2,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.img),
			gcr("restart", "restartPolicy Never or OnFailure", 1,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.restartPolicy}", `Never|OnFailure`),
		}
		if x.comp != "" {
			checks = append(checks, gcr("comp", "completions="+x.comp, 1,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.completions}", "^"+x.comp+"$"))
			checks = append(checks, gcr("par", "parallelism="+x.par, 1,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.parallelism}", "^"+x.par+"$"))
		}
		diff := models.DifficultyMedium
		if x.comp != "" {
			diff = models.DifficultyHard
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p6jobpat-%02d", i+1), models.DomainApplicationDesign, diff,
			fmt.Sprintf("Real-world Job: %s", x.name),
			"Jobs wrap batch workloads of any shape.",
			task, solution, x.ns,
			genHints(
				"'kubectl create job NAME --image=IMG -- CMD...' scaffolds quickly.",
				"Add completions/parallelism by editing the YAML.",
			),
			checks,
		))
	}
	return out
}

// ----------------------------------------- cron concurrency

func genP6CronConcurrency() []*models.Question {
	type v struct {
		ns, name, conc, sched string
	}
	variants := []v{
		{"ckad-p6cc01", "cc-forbid-a", "Forbid", "*/2 * * * *"},
		{"ckad-p6cc02", "cc-allow-a", "Allow", "*/3 * * * *"},
		{"ckad-p6cc03", "cc-replace-a", "Replace", "*/4 * * * *"},
		{"ckad-p6cc04", "cc-forbid-b", "Forbid", "*/6 * * * *"},
		{"ckad-p6cc05", "cc-replace-b", "Replace", "*/7 * * * *"},
		{"ckad-p6cc06", "cc-allow-b", "Allow", "*/8 * * * *"},
		{"ckad-p6cc07", "cc-forbid-c", "Forbid", "@hourly"},
		{"ckad-p6cc08", "cc-replace-c", "Replace", "@daily"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`kubectl create cronjob %s --image=busybox:1.36 --schedule='%s' -n %s -- /bin/sh -c date
# then set spec.concurrencyPolicy: %s`, x.name, x.sched, x.ns, x.conc)
		out = append(out, gq(
			fmt.Sprintf("qg-p6cronconc-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("Concurrency %s on %s", x.conc, x.name),
			"concurrencyPolicy decides whether overlapping runs queue, coexist, or replace.",
			fmt.Sprintf("In namespace %s, create CronJob '%s' (busybox:1.36, schedule '%s') with concurrencyPolicy=%s.", x.ns, x.name, x.sched, x.conc),
			solution, x.ns,
			genHints(
				"Forbid skips, Replace kills the old one, Allow overlaps.",
				"The field sits next to schedule under spec.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("CronJob %s exists", x.name), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o name", "cronjob.batch/"+x.name),
				gcs("sched", fmt.Sprintf("schedule '%s'", x.sched), 1,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.schedule}", x.sched),
				gcs("conc", fmt.Sprintf("concurrencyPolicy=%s", x.conc), 3,
					"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.concurrencyPolicy}", x.conc),
			},
		))
	}
	return out
}

// -------------------------------------------- probe thresholds

func genP6ProbeThresholds() []*models.Question {
	type v struct {
		ns, pod, fail, succ, timeout string
	}
	variants := []v{
		{"ckad-p6pt01", "pt-a", "3", "1", "1"},
		{"ckad-p6pt02", "pt-b", "6", "2", "2"},
		{"ckad-p6pt03", "pt-c", "2", "1", "5"},
		{"ckad-p6pt04", "pt-d", "9", "3", "3"},
		{"ckad-p6pt05", "pt-e", "4", "1", "2"},
		{"ckad-p6pt06", "pt-f", "5", "2", "4"},
		{"ckad-p6pt07", "pt-g", "7", "1", "1"},
		{"ckad-p6pt08", "pt-h", "1", "1", "10"},
		{"ckad-p6pt09", "pt-i", "8", "4", "2"},
		{"ckad-p6pt10", "pt-j", "3", "2", "6"},
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
    readinessProbe:
      httpGet: {path: /ready, port: 80}
      failureThreshold: %s
      successThreshold: %s
      timeoutSeconds: %s`, x.pod, x.ns, x.fail, x.succ, x.timeout)
		out = append(out, gq(
			fmt.Sprintf("qg-p6probethr-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Threshold tuning on %s", x.pod),
			"Thresholds control how stubborn probes are before declaring failure.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) with a readiness httpGet probe (/ready:80) where failureThreshold=%s, successThreshold=%s, timeoutSeconds=%s.", x.ns, x.pod, x.fail, x.succ, x.timeout),
			solution, x.ns,
			genHints(
				"successThreshold must be 1 for liveness probes.",
				"timeoutSeconds bounds each individual probe call.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("fail", "failureThreshold="+x.fail, 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].readinessProbe.failureThreshold}", "^"+x.fail+"$"),
				gcr("succ", "successThreshold="+x.succ, 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].readinessProbe.successThreshold}", "^"+x.succ+"$"),
				gcr("to", "timeoutSeconds="+x.timeout, 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].readinessProbe.timeoutSeconds}", "^"+x.timeout+"$"),
			},
		))
	}
	return out
}

// -------------------------------------------------- odd units

func genP6OddUnits() []*models.Question {
	type v struct {
		ns, pod, field, val string
	}
	variants := []v{
		{"ckad-p6ou01", "ou-a", "requests.cpu", "1.5"},
		{"ckad-p6ou02", "ou-b", "requests.memory", "1536Mi"},
		{"ckad-p6ou03", "ou-c", "limits.cpu", "2500m"},
		{"ckad-p6ou04", "ou-d", "limits.memory", "2Gi"},
		{"ckad-p6ou05", "ou-e", "requests.memory", "1.25Gi"},
		{"ckad-p6ou06", "ou-f", "limits.cpu", "3.75"},
		{"ckad-p6ou07", "ou-g", "requests.cpu", "1250m"},
		{"ckad-p6ou08", "ou-h", "limits.memory", "1280Mi"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		parts := strings.SplitN(x.field, ".", 2)
		section, res := parts[0], parts[1]
		resYAML := res
		if res == "cpu" {
			resYAML = "cpu"
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
      %s:
        %s: %s`, x.pod, x.ns, section, resYAML, x.val)
		jsonField := ".spec.containers[0].resources." + section + "." + res
		out = append(out, gq(
			fmt.Sprintf("qg-p6oddunit-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Unit drill: %s=%s", x.field, x.val),
			"Quantity parsing accepts cores, millicores and binary/decimal bytes.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) setting resources.%s.%s exactly to %s.", x.ns, x.pod, section, res, x.val),
			solution, x.ns,
			genHints(
				"1.5 CPU equals 1500m; kubectl stores what you write.",
				"Memory units: Mi/Gi are binary, M/G decimal.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("val", fmt.Sprintf("%s=%s", x.field, x.val), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath="+jsonField, "^"+x.val+"$"),
			},
		))
	}
	return out
}

// ---------------------------------------- multi-label workloads

func genP6MultiLabelWorkloads() []*models.Question {
	type v struct {
		ns, dep, k1, v1, k2, v2 string
	}
	variants := []v{
		{"ckad-p6ml01", "ml-web", "app", "storefront", "tier", "frontend"},
		{"ckad-p6ml02", "ml-api", "app", "checkout", "tier", "backend"},
		{"ckad-p6ml03", "ml-cache", "app", "sessions", "tier", "middleware"},
		{"ckad-p6ml04", "ml-search", "app", "query", "tier", "backend"},
		{"ckad-p6ml05", "ml-ui", "app", "admin", "tier", "frontend"},
		{"ckad-p6ml06", "ml-worker", "app", "mailer", "tier", "async"},
		{"ckad-p6ml07", "ml-gw", "app", "gateway", "tier", "edge"},
		{"ckad-p6ml08", "ml-auth", "app", "identity", "tier", "backend"},
		{"ckad-p6ml09", "ml-feed", "app", "activity", "tier", "async"},
		{"ckad-p6ml10", "ml-cdn", "app", "assets", "tier", "edge"},
		{"ckad-p6ml11", "ml-ledger", "app", "billing", "tier", "backend"},
		{"ckad-p6ml12", "ml-chat", "app", "realtime", "tier", "stateful"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
spec:
  replicas: 2
  selector:
    matchLabels:
      %s: %s
      %s: %s
  template:
    metadata:
      labels:
        %s: %s
        %s: %s
    spec:
      containers:
      - name: web
        image: nginx:1.25`, x.dep, x.ns, x.k1, x.v1, x.k2, x.v2, x.k1, x.v1, x.k2, x.v2)
		out = append(out, gq(
			fmt.Sprintf("qg-p6multilabel-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyMedium,
			fmt.Sprintf("Two-label Deployment %s", x.dep),
			"Selectors can require several labels at once.",
			fmt.Sprintf("In namespace %s, create Deployment '%s' (nginx:1.25, 2 replicas) whose pods carry BOTH labels %s=%s and %s=%s, with a matching selector.", x.ns, x.dep, x.k1, x.v1, x.k2, x.v2),
			solution, x.ns,
			genHints(
				"matchLabels takes multiple key/value pairs.",
				"Template labels must satisfy the selector exactly.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("l1", fmt.Sprintf("Label %s=%s", x.k1, x.v1), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.metadata.labels."+x.k1+"}", x.v1),
				gcs("l2", fmt.Sprintf("Label %s=%s", x.k2, x.v2), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.metadata.labels."+x.k2+"}", x.v2),
				gcs("sel1", fmt.Sprintf("Selector %s=%s", x.k1, x.v1), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.selector.matchLabels."+x.k1+"}", x.v1),
				gcs("sel2", fmt.Sprintf("Selector %s=%s", x.k2, x.v2), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.selector.matchLabels."+x.k2+"}", x.v2),
			},
		))
	}
	return out
}

// --------------------------------------- rollout to revision

func genP6RolloutToRevision() []*models.Question {
	type v struct {
		ns, dep, good, mid, bad string
	}
	variants := []v{
		{"ckad-p6rr01", "rr-shop", "nginx:1.25", "nginx:1.26", "nginx:broken-9"},
		{"ckad-p6rr02", "rr-pay", "httpd:2.4", "httpd:2.4.59", "httpd:broke"},
		{"ckad-p6rr03", "rr-api", "redis:7.0", "redis:7.2", "redis:nope"},
		{"ckad-p6rr04", "rr-fe", "envoy:v1.27", "envoy:v1.28", "envoy:bad"},
		{"ckad-p6rr05", "rr-be", "rabbitmq:3.12", "rabbitmq:3.13", "rabbitmq:x"},
		{"ckad-p6rr06", "rr-search", "elastic:8.11", "elastic:8.13", "elastic:y"},
		{"ckad-p6rr07", "rr-mail", "fluentd:1.15", "fluentd:1.16", "fluentd:z"},
		{"ckad-p6rr08", "rr-id", "auth-app:v1", "auth-app:v2", "auth-app:vX"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{Name: "create healthy deployment (revision 1)", CommandArgs: fmt.Sprintf("create deployment %s --image=%s -n %s", x.dep, x.good, x.ns)},
			{Name: "update to second image (revision 2)", CommandArgs: fmt.Sprintf("set image deployment/%s nginx=%s -n %s", x.dep, x.mid, x.ns)},
			{Name: "break with third image (revision 3)", CommandArgs: fmt.Sprintf("set image deployment/%s nginx=%s -n %s", x.dep, x.bad, x.ns)},
		}
		solution := fmt.Sprintf("kubectl rollout history deployment/%s -n %s   # inspect revisions\nkubectl rollout undo deployment/%s --to-revision=1 -n %s",
			x.dep, x.ns, x.dep, x.ns)
		out = append(out, gqp(
			fmt.Sprintf("qg-p6rollrev-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Roll %s back to revision 1", x.dep),
			"rollout undo --to-revision jumps to any recorded revision.",
			fmt.Sprintf("Deployment '%s' in %s went through images %s -> %s -> %s (revisions 1..3) and is now stuck. Roll it back specifically to REVISION 1 (image %s).", x.dep, x.ns, x.good, x.mid, x.bad, x.good),
			solution, x.ns, prepare,
			genHints(
				"Plain 'rollout undo' goes back one step — you need --to-revision here.",
				"'rollout history' lists revisions and their images.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("img", fmt.Sprintf("Image restored to %s", x.good), 4,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.good),
			},
		))
	}
	return out
}

// --------------------------------------------- expose variants

func genP6ExposeVariants() []*models.Question {
	type v struct {
		ns, target, tkind, svcType, port, tport, proto string
	}
	variants := []v{
		{"ckad-p6ex01", "ex-app-a", "deployment", "NodePort", "80", "8080", "TCP"},
		{"ckad-p6ex02", "ex-app-b", "deployment", "LoadBalancer", "8080", "8080", "TCP"},
		{"ckad-p6ex03", "ex-app-c", "pod", "ClusterIP", "6379", "6379", "TCP"},
		{"ckad-p6ex04", "ex-app-d", "pod", "NodePort", "9090", "9090", "TCP"},
		{"ckad-p6ex05", "ex-app-e", "deployment", "ClusterIP", "5432", "5432", "TCP"},
		{"ckad-p6ex06", "ex-app-f", "deployment", "LoadBalancer", "443", "8443", "TCP"},
		{"ckad-p6ex07", "ex-app-g", "pod", "ClusterIP", "80", "8000", "TCP"},
		{"ckad-p6ex08", "ex-app-h", "deployment", "NodePort", "3000", "3000", "TCP"},
		{"ckad-p6ex09", "ex-app-i", "pod", "LoadBalancer", "5672", "5672", "TCP"},
		{"ckad-p6ex10", "ex-app-j", "deployment", "ClusterIP", "9200", "9200", "TCP"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		tk := x.tkind
		if tk == "deployment" {
			tk = "deployment"
		}
		prepCmd := fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.target, x.ns)
		if x.tkind == "pod" {
			prepCmd = fmt.Sprintf("run %s --image=nginx:1.25 -n %s --command -- sleep 3600", x.target, x.ns)
		}
		prepare := []models.SetupStep{{Name: "create " + x.tkind, CommandArgs: prepCmd}}
		solution := fmt.Sprintf("kubectl expose %s %s --type=%s --port=%s --target-port=%s --protocol=%s --name=%s-svc -n %s",
			x.tkind, x.target, x.svcType, x.port, x.tport, x.proto, x.target, x.ns)
		out = append(out, gqp(
			fmt.Sprintf("qg-p6expose-%02d", i+1), models.DomainServicesNetworking, models.DifficultyMedium,
			fmt.Sprintf("Expose %s as %s", x.target, x.svcType),
			"kubectl expose works on pods and deployments alike.",
			fmt.Sprintf("In namespace %s, expose %s '%s' as a %s Service named '%s-svc': port %s, target-port %s, protocol %s.", x.ns, x.tkind, x.target, x.svcType, x.target, x.port, x.tport, x.proto),
			solution, x.ns, prepare,
			genHints(
				"--target-port points at the container side.",
				"NodePort/LoadBalancer add external access on top of ClusterIP.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Service %s-svc exists", x.target), 1,
					"get svc "+x.target+"-svc -n "+x.ns+" -o name", "service/"+x.target+"-svc"),
				gcs("type", fmt.Sprintf("type=%s", x.svcType), 2,
					"get svc "+x.target+"-svc -n "+x.ns+" -o jsonpath={.spec.type}", x.svcType),
				gcr("port", fmt.Sprintf("port=%s", x.port), 1,
					"get svc "+x.target+"-svc -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", "^"+x.port+"$"),
				gcr("tport", fmt.Sprintf("targetPort=%s", x.tport), 1,
					"get svc "+x.target+"-svc -n "+x.ns+" -o jsonpath={.spec.ports[0].targetPort}", "^"+x.tport+"$"),
			},
		))
	}
	return out
}

// ---------------------------------------------- set commands

func genP6SetCommands() []*models.Question {
	type v struct {
		ns, dep, mode, k, v string
	}
	variants := []v{
		{"ckad-p6set01", "set-env-a", "env", "LOG_LEVEL", "trace"},
		{"ckad-p6set02", "set-env-b", "env", "FEATURE", "on"},
		{"ckad-p6set03", "set-env-c", "env", "POOL_SIZE", "32"},
		{"ckad-p6set04", "set-sa-a", "serviceaccount", "ci-runner", ""},
		{"ckad-p6set05", "set-sa-b", "serviceaccount", "deploy-bot", ""},
		{"ckad-p6set06", "set-res-a", "resources", "cpu", "200m"},
		{"ckad-p6set07", "set-res-b", "resources", "memory", "256Mi"},
		{"ckad-p6set08", "set-img-c", "image", "nginx", "nginx:1.26"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.dep, x.ns),
		}}
		var solution, task string
		var checks []models.Check
		switch x.mode {
		case "env":
			solution = fmt.Sprintf("kubectl set env deployment/%s %s=%s -n %s", x.dep, x.k, x.v, x.ns)
			task = fmt.Sprintf("In namespace %s, inject environment variable %s=%s into Deployment '%s'.", x.ns, x.k, x.v, x.dep)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("envname", fmt.Sprintf("Env %s present", x.k), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].env[0].name}", x.k),
				gcs("envval", fmt.Sprintf("Value %s", x.v), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].env[0].value}", x.v),
			}
		case "serviceaccount":
			prepare = append(prepare, models.SetupStep{
				Name:        "create serviceaccount",
				CommandArgs: fmt.Sprintf("create serviceaccount %s -n %s", x.k, x.ns),
			})
			solution = fmt.Sprintf("kubectl set serviceaccount deployment/%s %s -n %s", x.dep, x.k, x.ns)
			task = fmt.Sprintf("In namespace %s, make Deployment '%s' run as ServiceAccount '%s'.", x.ns, x.dep, x.k)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("sa", fmt.Sprintf("Runs as %s", x.k), 3,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.serviceAccountName}", x.k),
			}
		case "resources":
			flag := "--limits=cpu=" + x.v
			if x.k == "memory" {
				flag = "--limits=memory=" + x.v
			}
			solution = fmt.Sprintf("kubectl set resources deployment/%s %s -n %s", x.dep, flag, x.ns)
			task = fmt.Sprintf("In namespace %s, set a %s LIMIT of %s on Deployment '%s'.", x.ns, x.k, x.v, x.dep)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcr("res", fmt.Sprintf("%s limit=%s", x.k, x.v), 3,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].resources.limits."+x.k+"}", "^"+x.v+"$"),
			}
		default: // image
			solution = fmt.Sprintf("kubectl set image deployment/%s %s=%s -n %s", x.dep, x.k, x.v, x.ns)
			task = fmt.Sprintf("In namespace %s, switch Deployment '%s' container '%s' to image %s.", x.ns, x.dep, x.k, x.v)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("img", fmt.Sprintf("Image %s", x.v), 3,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.v),
			}
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-p6setcmd-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyMedium,
			fmt.Sprintf("kubectl set drill: %s", x.mode),
			"'kubectl set' mutates live fields without editing YAML.",
			task, solution, x.ns, prepare,
			genHints(
				"kubectl set supports env, image, resources, serviceaccount.",
				"Changes trigger a new rollout on Deployments.",
			),
			checks,
		))
	}
	return out
}

// -------------------------------------------- tolerations more

func genP6TolerationMore() []*models.Question {
	type v struct {
		ns, pod, key, op, val, eff string
	}
	variants := []v{
		{"ckad-p6tol01", "tol-exists", "gpu", "Exists", "", "NoSchedule"},
		{"ckad-p6tol02", "tol-pref", "preemptible", "Equal", "true", "PreferNoSchedule"},
		{"ckad-p6tol03", "tol-noexec", "draining", "Exists", "", "NoExecute"},
		{"ckad-p6tol04", "tol-equal", "arch-class", "Equal", "b2", "NoSchedule"},
		{"ckad-p6tol05", "tol-anyval", "dedicated-team", "Exists", "", "NoSchedule"},
		{"ckad-p6tol06", "tol-window", "maintenance", "Equal", "nightly", "NoExecute"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		valYAML := ""
		valDesc := fmt.Sprintf("operator %s", x.op)
		if x.op == "Equal" {
			valYAML = fmt.Sprintf("\n    value: %s", x.val)
			valDesc = fmt.Sprintf("operator Equal value=%s", x.val)
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  tolerations:
  - key: %s
    operator: %s%s
    effect: %s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.key, x.op, valYAML, x.eff)
		out = append(out, gq(
			fmt.Sprintf("qg-p6tolmore-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Toleration drill: %s", x.pod),
			"Tolerations pair operators and effects precisely.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) tolerating taint key=%s (%s) with effect %s.", x.ns, x.pod, x.key, valDesc, x.eff),
			solution, x.ns,
			genHints(
				"Exists ignores the value entirely.",
				"NoExecute evicts already-running Pods too.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("key", fmt.Sprintf("key=%s", x.key), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.tolerations[0].key}", x.key),
				gcs("op", fmt.Sprintf("operator=%s", x.op), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.tolerations[0].operator}", x.op),
				gcs("eff", fmt.Sprintf("effect=%s", x.eff), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.tolerations[0].effect}", x.eff),
			},
		))
	}
	return out
}

// ------------------------------------------- immutable configmaps

func genP6CMImmutable() []*models.Question {
	type v struct {
		ns, cm, key, val string
	}
	variants := []v{
		{"ckad-p6im01", "frozen-cfg-a", "endpoint", "https://a.internal"},
		{"ckad-p6im02", "frozen-cfg-b", "region", "us-west-2"},
		{"ckad-p6im03", "frozen-cfg-c", "max_conn", "500"},
		{"ckad-p6im04", "frozen-cfg-d", "feature_x", "off"},
		{"ckad-p6im05", "frozen-cfg-e", "tls_min", "1.2"},
		{"ckad-p6im06", "frozen-cfg-f", "queue", "jobs.v1"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
immutable: true
data:
  %s: %s`, x.cm, x.ns, x.key, x.val)
		out = append(out, gq(
			fmt.Sprintf("qg-p6cmimm-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Immutable ConfigMap %s", x.cm),
			"Immutable ConfigMaps reject updates and watch less — good for scale.",
			fmt.Sprintf("In namespace %s, create an IMMUTABLE ConfigMap named '%s' with data %s=%s.", x.ns, x.cm, x.key, x.val),
			solution, x.ns,
			genHints(
				"immutable is metadata-level, sibling to data.",
				"Once created it cannot be flipped back without delete/recreate.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("ConfigMap %s exists", x.cm), 1,
					"get configmap "+x.cm+" -n "+x.ns+" -o name", "configmap/"+x.cm),
				gcs("immutable", "immutable=true", 3,
					"get configmap "+x.cm+" -n "+x.ns+" -o jsonpath={.immutable}", "true"),
				gcs("data", fmt.Sprintf("%s=%s", x.key, x.val), 1,
					"get configmap "+x.cm+" -n "+x.ns+" -o jsonpath={.data."+x.key+"}", x.val),
			},
		))
	}
	return out
}

// ------------------------------------------- stringData secrets

func genP6SecretStringData() []*models.Question {
	type v struct {
		ns, sec, key, val string
	}
	variants := []v{
		{"ckad-p6sd01", "sd-secret-a", "username", "svc-a"},
		{"ckad-p6sd02", "sd-secret-b", "api_key", "key-9876"},
		{"ckad-p6sd03", "sd-secret-c", "client_id", "cid-42"},
		{"ckad-p6sd04", "sd-secret-d", "password", "hunter2000"},
		{"ckad-p6sd05", "sd-secret-e", "token", "tok-abc"},
		{"ckad-p6sd06", "sd-secret-f", "secret_key", "sk-live-9"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  %s: %s`, x.sec, x.ns, x.key, x.val)
		out = append(out, gq(
			fmt.Sprintf("qg-p6strdata-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyMedium,
			fmt.Sprintf("stringData Secret %s", x.sec),
			"stringData lets you write plaintext; the API server base64-encodes into data.",
			fmt.Sprintf("In namespace %s, create a Secret named '%s' using stringData with key %s=%s.", x.ns, x.sec, x.key, x.val),
			solution, x.ns,
			genHints(
				"stringData is write-only convenience; read back via .data.",
				"type defaults to Opaque.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Secret %s exists", x.sec), 1,
					"get secret "+x.sec+" -n "+x.ns+" -o name", "secret/"+x.sec),
				gcr("data-key", fmt.Sprintf(".data.%s populated", x.key), 3,
					"get secret "+x.sec+" -n "+x.ns+" -o jsonpath={.data."+x.key+"}", `^.+$`),
			},
		))
	}
	return out
}

// ------------------------------------------------ hostAliases

func genP6HostAliases() []*models.Question {
	type v struct {
		ns, pod, ip, host string
	}
	variants := []v{
		{"ckad-p6ha01", "ha-db", "192.168.1.10", "db.corp.local"},
		{"ckad-p6ha02", "ha-mq", "192.168.1.11", "mq.corp.local"},
		{"ckad-p6ha03", "ha-api", "10.20.30.40", "api.partner.io"},
		{"ckad-p6ha04", "ha-cache", "10.20.30.41", "cache.partner.io"},
		{"ckad-p6ha05", "ha-ldap", "172.31.0.9", "ldap.corp.local"},
		{"ckad-p6ha06", "ha-smtp", "172.31.0.10", "smtp.corp.local"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  hostAliases:
  - ip: %s
    hostnames:
    - %s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.ip, x.host)
		out = append(out, gq(
			fmt.Sprintf("qg-p6hostalias-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("/etc/hosts entry in %s", x.pod),
			"hostAliases append entries to the Pod's /etc/hosts.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) mapping hostname %s to IP %s via hostAliases.", x.ns, x.pod, x.host, x.ip),
			solution, x.ns,
			genHints(
				"hostAliases lives under Pod spec, not the container.",
				"Each alias has one ip and a list of hostnames.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("ip", fmt.Sprintf("Alias ip %s", x.ip), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.hostAliases[0].ip}", x.ip),
				gcs("host", fmt.Sprintf("Hostname %s", x.host), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.hostAliases[0].hostnames[0]}", x.host),
			},
		))
	}
	return out
}

// ------------------------------------------------ DNS settings

func genP6DnsSettings() []*models.Question {
	type v struct {
		ns, pod, policy, hostname, subdomain string
	}
	variants := []v{
		{"ckad-p6dns01", "dns-default", "Default", "", ""},
		{"ckad-p6dns02", "dns-hostnet", "ClusterFirstWithHostNet", "", ""},
		{"ckad-p6dns03", "dns-none", "None", "", ""},
		{"ckad-p6dns04", "dns-first", "ClusterFirst", "", ""},
		{"ckad-p6fqdn01", "fqdn-a", "ClusterFirst", "web-0", "headless-svc"},
		{"ckad-p6fqdn02", "fqdn-b", "ClusterFirst", "worker-1", "workers-ns"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		fqdn := x.hostname != ""
		hostNet := x.policy == "ClusterFirstWithHostNet"
		extraSpec := ""
		if fqdn {
			extraSpec = fmt.Sprintf("  hostname: %s\n  subdomain: %s\n", x.hostname, x.subdomain)
		}
		if hostNet {
			extraSpec += "  hostNetwork: true\n"
		}
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
%sextra-cont`, x.pod, x.ns, extraSpec)
		cont := `  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`
		solution = strings.Replace(solution, "extra-cont", cont, 1)
		solution = strings.Replace(solution, "spec:\n  dnsPolicy", "spec:\n  dnsPolicy", 1)
		solution = strings.Replace(solution, "spec:\n", "spec:\n  dnsPolicy: "+x.policy+"\n", 1)

		task := fmt.Sprintf("In namespace %s, create Pod '%s' (busybox:1.36) with dnsPolicy=%s", x.ns, x.pod, x.policy)
		if fqdn {
			task += fmt.Sprintf(", hostname=%s and subdomain=%s (so its FQDN is %s.%s)", x.hostname, x.subdomain, x.hostname, x.subdomain)
		}
		if hostNet {
			task += " together with hostNetwork: true"
		}
		task += "."

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
			gcs("policy", fmt.Sprintf("dnsPolicy=%s", x.policy), 3,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.dnsPolicy}", x.policy),
		}
		if fqdn {
			checks = append(checks, gcs("hostname", fmt.Sprintf("hostname=%s", x.hostname), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.hostname}", x.hostname))
			checks = append(checks, gcs("subdomain", fmt.Sprintf("subdomain=%s", x.subdomain), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.subdomain}", x.subdomain))
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p6dns-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("DNS policy on %s", x.pod),
			"dnsPolicy controls how the Pod resolves names.",
			task, solution, x.ns,
			genHints(
				"dnsPolicy sits directly under spec.",
				"hostname/subdomain build a stable FQDN via a headless Service.",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------------ grace period

func genP6GracePeriod() []*models.Question {
	type v struct {
		ns, pod, secs string
	}
	variants := []v{
		{"ckad-p6gp01", "gp-fast", "5"},
		{"ckad-p6gp02", "gp-slow", "120"},
		{"ckad-p6gp03", "gp-mid", "45"},
		{"ckad-p6gp04", "gp-instant", "1"},
		{"ckad-p6gp05", "gp-long", "300"},
		{"ckad-p6gp06", "gp-short", "10"},
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
    image: nginx:1.25`, x.pod, x.ns, x.secs)
		out = append(out, gq(
			fmt.Sprintf("qg-p6grace-%02d", i+1), models.DomainApplicationObservability, models.DifficultyMedium,
			fmt.Sprintf("Grace period %ss on %s", x.secs, x.pod),
			"terminationGracePeriodSeconds bounds how long SIGKILL waits after SIGTERM.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) with terminationGracePeriodSeconds=%s.", x.ns, x.pod, x.secs),
			solution, x.ns,
			genHints(
				"The field sits directly under Pod spec.",
				"Zero means kill immediately.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("grace", fmt.Sprintf("grace=%s", x.secs), 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.terminationGracePeriodSeconds}", "^"+x.secs+"$"),
			},
		))
	}
	return out
}

// ------------------------------------------- named container ports

func genP6NamedContainerPorts() []*models.Question {
	type v struct {
		ns, pod, pname, port, proto string
	}
	variants := []v{
		{"ckad-p6ncp01", "ncp-http", "http", "8080", "TCP"},
		{"ckad-p6ncp02", "ncp-https", "https", "8443", "TCP"},
		{"ckad-p6ncp03", "ncp-dns", "dns", "53", "UDP"},
		{"ckad-p6ncp04", "ncp-grpc", "grpc", "50051", "TCP"},
		{"ckad-p6ncp05", "ncp-metrics", "metrics", "9090", "TCP"},
		{"ckad-p6ncp06", "ncp-stats", "stats", "8125", "UDP"},
		{"ckad-p6ncp07", "ncp-admin", "admin", "9001", "TCP"},
		{"ckad-p6ncp08", "ncp-health", "health", "7000", "TCP"},
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
    - {name: %s, containerPort: %s, protocol: %s}`, x.pod, x.ns, x.pname, x.port, x.proto)
		out = append(out, gq(
			fmt.Sprintf("qg-p6namedcp-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("Named port %s on %s", x.pname, x.pod),
			"Named ports let Services reference targets symbolically.",
			fmt.Sprintf("In namespace %s, create Pod '%s' (nginx:1.25) exposing containerPort %s NAMED '%s' with protocol %s.", x.ns, x.pod, x.port, x.pname, x.proto),
			solution, x.ns,
			genHints(
				"name comes first alphabetically but matters most.",
				"Services can then use targetPort: <name>.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("name", fmt.Sprintf("Port named %s", x.pname), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].name}", x.pname),
				gcr("port", fmt.Sprintf("containerPort %s", x.port), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].containerPort}", "^"+x.port+"$"),
				gcs("proto", fmt.Sprintf("protocol %s", x.proto), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].protocol}", x.proto),
			},
		))
	}
	return out
}

// ---------------------------------------------- shareProcessNamespace

func genP6ShareProcess() []*models.Question {
	type v struct {
		ns, pod string
	}
	variants := []v{
		{"ckad-p6sp01", "sp-debugger"},
		{"ckad-p6sp02", "sp-profiler"},
		{"ckad-p6sp03", "sp-shellpair"},
		{"ckad-p6sp04", "sp-monitor"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  shareProcessNamespace: true
  containers:
  - name: main
    image: nginx:1.25
  - name: toolbox
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns)
		out = append(out, gq(
			fmt.Sprintf("qg-p6shareproc-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("Shared PID namespace in %s", x.pod),
			"shareProcessNamespace lets containers see each other's processes.",
			fmt.Sprintf("In namespace %s, create Pod '%s' with containers 'main' (nginx:1.25) and 'toolbox' (busybox:1.36) that SHARE the process namespace.", x.ns, x.pod),
			solution, x.ns,
			genHints(
				"shareProcessNamespace is Pod-level, default false.",
				"Useful for debugging with ps across containers.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("share", "shareProcessNamespace=true", 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.shareProcessNamespace}", "true"),
				gcr("two", "Has two containers", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `main toolbox`),
			},
		))
	}
	return out
}

// ------------------------------------- more simple pods

func genP6MoreSimplePods() []*models.Question {
	type v struct{ ns, name, image, port string }
	variants := []v{
		{"ckad-p6spod01", "geo-api", "geo:v2.1", "7000"},
		{"ckad-p6spod02", "img-resize", "imagick:7", "8088"},
		{"ckad-p6spod03", "pdf-gen", "pdfium:1.0", "9001"},
		{"ckad-p6spod04", "sms-send", "sms-gw:3.2", "7777"},
		{"ckad-p6spod05", "voice-nlu", "nlu:0.9", "5000"},
		{"ckad-p6spod06", "chart-render", "chartsrv:6", "4001"},
		{"ckad-p6spod07", "feed-pull", "feeder:2.4", "6060"},
		{"ckad-p6spod08", "otp-check", "otpv:1.1", "5555"},
		{"ckad-p6spod09", "zipkin", "zipkin:3", "9411"},
		{"ckad-p6spod10", "grafana", "grafana:10.4", "3001"},
		{"ckad-p6spod11", "prom", "prometheus:v2.52", "9091"},
		{"ckad-p6spod12", "loki", "loki:3.0", "3100"},
		{"ckad-p6spod13", "tempo", "tempo:2.4", "3200"},
		{"ckad-p6spod14", "vault", "vault:1.16", "8200"},
		{"ckad-p6spod15", "consul", "consul:1.18", "8500"},
		{"ckad-p6spod16", "nats", "nats:2.10", "4222"},
		{"ckad-p6spod17", "minio", "minio:2024", "9002"},
		{"ckad-p6spod18", "registry", "registry:2.8", "5001"},
		{"ckad-p6spod19", "jenkins", "jenkins:lts", "8081"},
		{"ckad-p6spod20", "sonar", "sonarqube:10", "9002"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if i%4 == 3 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p6simpod-%02d", i+1), models.DomainApplicationDesign, diff,
			fmt.Sprintf("Run the %s Pod", x.name),
			"Single-container Pods remain the core building block.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' using image '%s' exposing container port %s.", x.ns, x.name, x.image, x.port),
			fmt.Sprintf("kubectl run %s --image=%s --port=%s -n %s", x.name, x.image, x.port, x.ns),
			x.ns,
			genHints(
				"kubectl run handles this imperatively.",
				"Check with 'kubectl get pod -o yaml'.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.name), 1,
					"get pod "+x.name+" -n "+x.ns+" -o name", "pod/"+x.name),
				gcs("image", fmt.Sprintf("Uses %s", x.image), 2,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].image}", x.image),
				gcr("port", fmt.Sprintf("Exposes %s", x.port), 1,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].containerPort}", "^"+x.port+"$"),
			},
		))
	}
	return out
}

// ------------------------------------- more labeled pods

func genP6MoreLabeledPods() []*models.Question {
	type v struct{ ns, name, image, key, val string }
	variants := []v{
		{"ckad-p6lpod01", "lp-ingest", "ingestor:5", "stream", "events"},
		{"ckad-p6lpod02", "lp-enrich", "enricher:3", "stream", "clicks"},
		{"ckad-p6lpod03", "lp-store", "writer:9", "sink", "warehouse"},
		{"ckad-p6lpod04", "lp-alert", "alerter:2", "severity", "high"},
		{"ckad-p6lpod05", "lp-rollup", "roller:7", "window", "5m"},
		{"ckad-p6lpod06", "lp-export", "exporter:4", "format", "parquet"},
		{"ckad-p6lpod07", "lp-validate", "validator:6", "stage", "pre"},
		{"ckad-p6lpod08", "lp-mask", "masker:1", "pii", "scrub"},
		{"ckad-p6lpod09", "lp-sample", "sampler:8", "rate", "10pct"},
		{"ckad-p6lpod10", "lp-route", "router:3", "lane", "fast"},
		{"ckad-p6lpod11", "lp-archive", "archiver:2", "retention", "90d"},
		{"ckad-p6lpod12", "lp-replay", "replayer:5", "mode", "shadow"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		out = append(out, gq(
			fmt.Sprintf("qg-p6lblpod-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("Labeled Pod %s (%s=%s)", x.name, x.key, x.val),
			"Labels drive grouping and selection everywhere.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (image '%s') labeled %s=%s.", x.ns, x.name, x.image, x.key, x.val),
			fmt.Sprintf("kubectl run %s --image=%s -n %s -l %s=%s", x.name, x.image, x.ns, x.key, x.val),
			x.ns,
			genHints(
				"-l key=value attaches labels at creation.",
				"metadata.labels holds them in YAML.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.name), 1,
					"get pod "+x.name+" -n "+x.ns+" -o name", "pod/"+x.name),
				gcs("label", fmt.Sprintf("Label %s=%s", x.key, x.val), 3,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.metadata.labels."+x.key+"}", x.val),
			},
		))
	}
	return out
}

// ------------------------------------- more deployments

func genP6MoreDeployments() []*models.Question {
	type v struct {
		ns, name, image string
		replicas        int
		port            string
	}
	variants := []v{
		{"ckad-p6dep01", "ml-trainer", "trainer:1.4", 2, "6000"},
		{"ckad-p6dep02", "feat-store", "feast:0.40", 3, "6566"},
		{"ckad-p6dep03", "vec-db", "qdrant:v1.9", 2, "6333"},
		{"ckad-p6dep04", "graph-api", "neo4j:5.20", 1, "7474"},
		{"ckad-p6dep05", "doc-parse", "parser:2.0", 4, "8100"},
		{"ckad-p6dep06", "mail-relay", "relay:1.2", 2, "587"},
		{"ckad-p6dep07", "push-svc", "pusher:3.3", 3, "6001"},
		{"ckad-p6dep08", "ws-hub", "hub:2.2", 5, "6002"},
		{"ckad-p6dep09", "cron-runner", "runner:1.0", 2, "7070"},
		{"ckad-p6dep10", "audit-log", "auditd:4", 3, "5140"},
		{"ckad-p6dep11", "geo-router", "georouter:2.8", 4, "8090"},
		{"ckad-p6dep12", "rate-limit", "limiter:1.7", 3, "8091"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if x.replicas >= 4 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-p6dep-%02d", i+1), models.DomainApplicationDeployment, diff,
			fmt.Sprintf("Deploy %s (%d replicas)", x.name, x.replicas),
			"Deployments keep replica sets healthy and rolling.",
			fmt.Sprintf("In namespace %s, create Deployment '%s' with %d replicas of image '%s', label app=%s, container port %s.", x.ns, x.name, x.replicas, x.image, x.name, x.port),
			fmt.Sprintf("kubectl create deployment %s --image=%s -n %s --replicas=%d --port=%s", x.name, x.image, x.ns, x.replicas, x.port),
			x.ns,
			genHints(
				"--replicas sets count; label app=<name> is automatic.",
				"Verify with 'kubectl get deploy'.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.name), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o name", "deployment.apps/"+x.name),
				gcr("replicas", fmt.Sprintf("replicas=%d", x.replicas), 2,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.replicas}", fmt.Sprintf("^%d$", x.replicas)),
				gcs("image", fmt.Sprintf("Image %s", x.image), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.image),
			},
		))
	}
	return out
}

// ------------------------------------- more scaling

func genP6MoreScaling() []*models.Question {
	type v struct {
		ns, name string
		from, to int
	}
	variants := []v{
		{"ckad-p6sc01", "sc-front", 1, 7},
		{"ckad-p6sc02", "sc-back", 6, 2},
		{"ckad-p6sc03", "sc-side", 2, 9},
		{"ckad-p6sc04", "sc-core", 8, 3},
		{"ckad-p6sc05", "sc-edge", 3, 10},
		{"ckad-p6sc06", "sc-batch", 10, 1},
		{"ckad-p6sc07", "sc-realtime", 4, 6},
		{"ckad-p6sc08", "sc-nightly", 5, 2},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create source deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=%d", x.name, x.ns, x.from),
		}}
		dir := "Scale up"
		if x.to < x.from {
			dir = "Scale down"
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-p6scale-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("%s %s to %d", dir, x.name, x.to),
			"Replica counts change live with kubectl scale.",
			fmt.Sprintf("In namespace %s, scale Deployment '%s' from %d to %d replicas.", x.ns, x.name, x.from, x.to),
			fmt.Sprintf("kubectl scale deployment %s --replicas=%d -n %s", x.name, x.to, x.ns),
			x.ns, prepare,
			genHints(
				"One-liner: kubectl scale deploy NAME --replicas=N.",
				"Editing spec.replicas also works.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.name), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o name", "deployment.apps/"+x.name),
				gcr("replicas", fmt.Sprintf("replicas=%d", x.to), 3,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.replicas}", fmt.Sprintf("^%d$", x.to)),
			},
		))
	}
	return out
}

// ------------------------------------- more set-image

func genP6MoreSetImage() []*models.Question {
	type v struct{ ns, name, oldImg, newImg, container string }
	variants := []v{
		{"ckad-p6si01", "si-edge", "envoy:v1.27", "envoy:v1.29", "envoy"},
		{"ckad-p6si02", "si-cache", "memcached:1.6", "memcached:1.6.29", "memcached"},
		{"ckad-p6si03", "si-queue", "nats:2.9", "nats:2.10", "nats"},
		{"ckad-p6si04", "si-store", "minio:2023", "minio:2024", "minio"},
		{"ckad-p6si05", "si-ci", "jenkins:2.440", "jenkins:2.452", "jenkins"},
		{"ckad-p6si06", "si-mon", "prom:v2.51", "prom:v2.53", "prom"},
		{"ckad-p6si07", "si-log", "loki:2.9", "loki:3.0", "loki"},
		{"ckad-p6si08", "si-trace", "tempo:2.3", "tempo:2.4", "tempo"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment with old image",
			CommandArgs: fmt.Sprintf("create deployment %s --image=%s -n %s", x.name, x.oldImg, x.ns),
		}}
		out = append(out, gqp(
			fmt.Sprintf("qg-p6setimg-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("Upgrade %s to %s", x.name, x.newImg),
			"Image updates start a controlled rollout.",
			fmt.Sprintf("In namespace %s, update Deployment '%s' from '%s' to '%s'.", x.ns, x.name, x.oldImg, x.newImg),
			fmt.Sprintf("kubectl set image deployment/%s %s=%s -n %s", x.name, x.container, x.newImg, x.ns),
			x.ns, prepare,
			genHints(
				"kubectl set image deploy/NAME CONTAINER=IMAGE.",
				"A rollout starts automatically.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.name), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o name", "deployment.apps/"+x.name),
				gcs("image", fmt.Sprintf("Runs %s", x.newImg), 3,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.newImg),
			},
		))
	}
	return out
}

// ------------------------------------- more PVC mounts

func genP6MorePVCMounts() []*models.Question {
	type v struct{ ns, pvc, pod, mount string }
	variants := []v{
		{"ckad-p6pm01", "media-vol", "transcoder", "/media/in"},
		{"ckad-p6pm02", "model-vol", "inferencer", "/models"},
		{"ckad-p6pm03", "dataset-vol", "trainer", "/datasets"},
		{"ckad-p6pm04", "snapshot-vol", "backuper", "/snapshots"},
		{"ckad-p6pm05", "share-vol", "collaborator", "/workspace"},
		{"ckad-p6pm06", "raw-vol", "blockwriter", "/dev/raw"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name: "create claim " + x.pvc,
			YAML: fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
spec:
  accessModes: ['ReadWriteOnce']
  resources:
    requests:
      storage: 2Gi`, x.pvc),
			Namespace: x.ns,
		}}
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
    - {name: storage, mountPath: %s}
  volumes:
  - name: storage
    persistentVolumeClaim:
      claimName: %s`, x.pod, x.ns, x.mount, x.pvc)
		out = append(out, gqp(
			fmt.Sprintf("qg-p6pvcmount-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Mount claim %s", x.pvc),
			"PVC-backed volumes persist beyond Pod lifetimes.",
			fmt.Sprintf("In namespace %s (claim '%s' exists), create Pod '%s' (busybox:1.36) mounting the claim at %s.", x.ns, x.pvc, x.pod, x.mount),
			solution, x.ns, prepare,
			genHints(
				"Volume type persistentVolumeClaim references claimName.",
				"mountPath is the in-container location.",
			),
			[]models.Check{
				gcs("pod", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("claim", fmt.Sprintf("References %s", x.pvc), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].persistentVolumeClaim.claimName}", x.pvc),
				gcs("mount", fmt.Sprintf("Mounted at %s", x.mount), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].mountPath}", x.mount),
			},
		))
	}
	return out
}
