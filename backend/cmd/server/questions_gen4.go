package main

import (
	"fmt"
	"strings"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// This file adds harder, exam-realistic questions: scheduling constraints,
// security contexts, workload tuning fields, rollout strategies, real
// NetworkPolicy rules, multi-path Ingress, secret volumes, the ambassador
// pattern, autoscaling, advanced Services and exec probes.

// ------------------------------------------------------- scheduling

func genHardScheduling() []*models.Question {
	type v struct {
		ns, pod, mode, key, val, effect string
	}
	variants := []v{
		{"ckad-hsched01", "ssd-app", "nodeSelector", "disktype", "ssd", ""},
		{"ckad-hsched02", "gpu-app", "nodeSelector", "accelerator", "a100", ""},
		{"ckad-hsched03", "zone-app", "nodeSelector", "topology.kubernetes.io/zone", "east", ""},
		{"ckad-hsched04", "spot-app", "toleration", "spot", "true", "NoSchedule"},
		{"ckad-hsched05", "maint-app", "toleration", "maintenance", "", "NoExecute"},
		{"ckad-hsched06", "batch-app", "toleration", "dedicated", "batch", "NoSchedule"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var solution, task string
		var checks []models.Check
		switch x.mode {
		case "nodeSelector":
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) that is only scheduled on nodes carrying the label %s=%s.", x.ns, x.pod, x.key, x.val)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  nodeSelector:
    %s: %s
  containers:
  - name: web
    image: nginx:1.25`, x.pod, x.ns, x.key, x.val)
			keyPath := ".metadata.labels." + x.key
			if strings.Contains(x.key, ".") || strings.Contains(x.key, "/") {
				keyPath = fmt.Sprintf(`.metadata.labels['%s']`, x.key)
			}
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("selector", fmt.Sprintf("nodeSelector has %s=%s", x.key, x.val), 3,
					fmt.Sprintf("get pod %s -n %s -o jsonpath={.spec.nodeSelector%s}", x.pod, x.ns, keyPath), x.val),
			}
		default: // toleration
			operator := "Equal"
			valPart := ""
			if x.val != "" {
				valPart = fmt.Sprintf("\n    value: %s", x.val)
			} else {
				operator = "Exists"
			}
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) that tolerates the taint key=%s%s effect=%s.", x.ns, x.pod, x.key, valPartPlain(x.val), x.effect)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  tolerations:
  - key: %s%s
    operator: %s
    effect: %s
  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']`, x.pod, x.ns, x.key, valPart, operator, x.effect)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("key", fmt.Sprintf("Toleration key is %s", x.key), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.tolerations[0].key}", x.key),
				gcr("effect", fmt.Sprintf("Toleration effect is %s", x.effect), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.tolerations[0].effect}", "^"+x.effect+"$"),
			}
		}
		diff := models.DifficultyHard
		if i < 2 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-hsched-%02d", i+1), models.DomainApplicationDeployment, diff,
			fmt.Sprintf("Schedule %s with %s", x.pod, x.mode),
			"nodeSelector and tolerations steer Pods onto the right nodes.",
			task, solution, x.ns,
			genHints(
				"nodeSelector maps label keys to required values.",
				"Tolerations pair with node taints; operator Equal needs a value, Exists does not.",
			),
			checks,
		))
	}
	return out
}

func valPartPlain(val string) string {
	if val == "" {
		return ""
	}
	return fmt.Sprintf(" value=%s", val)
}

// --------------------------------------------------- security contexts

func genHardSecurityContexts() []*models.Question {
	type v struct {
		ns, pod, focus string
	}
	variants := []v{
		{"ckad-hsec01", "uid-app", "pod-runasuser-fsgroup"},
		{"ckad-hsec02", "noroot-app", "container-nonroot"},
		{"ckad-hsec03", "noesc-app", "no-priv-escalation"},
		{"ckad-hsec04", "rofs-app", "readonly-rootfs"},
		{"ckad-hsec05", "dropcap-app", "drop-all-caps"},
		{"ckad-hsec06", "bindcap-app", "add-cap"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var solution, task string
		var checks []models.Check
		base := func(extraContainer, extraPod string) string {
			return fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
%s  containers:
  - name: app
    image: busybox:1.36
    command: ['sh','-c','sleep 3600']
%s`, x.pod, x.ns, extraPod, extraContainer)
		}
		switch x.focus {
		case "pod-runasuser-fsgroup":
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) whose Pod-level securityContext sets runAsUser=1000 and fsGroup=2000.", x.ns, x.pod)
			solution = base("", "  securityContext:\n    runAsUser: 1000\n    fsGroup: 2000\n")
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcr("uid", "Pod securityContext.runAsUser=1000", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.securityContext.runAsUser}", "^1000$"),
				gcr("fsgroup", "Pod securityContext.fsGroup=2000", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.securityContext.fsGroup}", "^2000$"),
			}
		case "container-nonroot":
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) whose container sets securityContext.runAsNonRoot=true.", x.ns, x.pod)
			solution = base("    securityContext:\n      runAsNonRoot: true\n", "")
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("nonroot", "Container runAsNonRoot=true", 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.runAsNonRoot}", "true"),
			}
		case "no-priv-escalation":
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) whose container sets securityContext.allowPrivilegeEscalation=false.", x.ns, x.pod)
			solution = base("    securityContext:\n      allowPrivilegeEscalation: false\n", "")
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("noesc", "allowPrivilegeEscalation=false", 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.allowPrivilegeEscalation}", "false"),
			}
		case "readonly-rootfs":
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) whose container sets securityContext.readOnlyRootFilesystem=true.", x.ns, x.pod)
			solution = base("    securityContext:\n      readOnlyRootFilesystem: true\n", "")
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("rofs", "readOnlyRootFilesystem=true", 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.readOnlyRootFilesystem}", "true"),
			}
		case "drop-all-caps":
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) whose container drops ALL Linux capabilities (securityContext.capabilities.drop: [ALL]).", x.ns, x.pod)
			solution = base("    securityContext:\n      capabilities:\n        drop: [ALL]\n", "")
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("drop", "Capabilities drop includes ALL", 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.capabilities.drop[*]}", "ALL"),
			}
		default: // add-cap
			task = fmt.Sprintf("In namespace %s, create a Pod named '%s' (nginx:1.25) whose container ADDs the NET_BIND_SERVICE capability.", x.ns, x.pod)
			solution = base("    securityContext:\n      capabilities:\n        add: [NET_BIND_SERVICE]\n", "")
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("add", "Capabilities add includes NET_BIND_SERVICE", 3,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].securityContext.capabilities.add[*]}", "NET_BIND_SERVICE"),
			}
		}
		out = append(out, gq(
			fmt.Sprintf("qg-hsecctx-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Harden %s with securityContext", x.pod),
			"Security contexts control the UID, privileges and capabilities of containers.",
			task, solution, x.ns,
			genHints(
				"Pod-wide settings live under spec.securityContext; per-container under spec.containers[].securityContext.",
				"Verify with 'kubectl get pod NAME -o yaml'.",
			),
			checks,
		))
	}
	return out
}

// -------------------------------------------------- job/cronjob tuning

func genHardJobsAdvanced() []*models.Question {
	type v struct {
		ns, name       string
		deadline, bl   string
		ttl, comp, par string
	}
	variants := []v{
		{"ckad-hjob01", "quick-job", "60", "", "", "", ""},
		{"ckad-hjob02", "retry-job", "", "3", "", "", ""},
		{"ckad-hjob03", "ttl-job", "", "", "120", "", ""},
		{"ckad-hjob04", "fanout-job", "", "2", "", "4", "2"},
		{"ckad-hjob05", "strict-job", "30", "1", "", "", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		fields := ""
		if x.deadline != "" {
			fields += fmt.Sprintf("  activeDeadlineSeconds: %s\n", x.deadline)
		}
		if x.bl != "" {
			fields += fmt.Sprintf("  backoffLimit: %s\n", x.bl)
		}
		if x.ttl != "" {
			fields += fmt.Sprintf("  ttlSecondsAfterFinished: %s\n", x.ttl)
		}
		if x.comp != "" {
			fields += fmt.Sprintf("  completions: %s\n  parallelism: %s\n", x.comp, x.par)
		}
		solution := fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: %s
spec:
%s  template:
    spec:
      restartPolicy: Never
      containers:
      - name: work
        image: busybox:1.36
        command: ['sh','-c','echo done']`, x.name, x.ns, fields)

		task := fmt.Sprintf("In namespace %s, create a Job named '%s' (busybox:1.36) running 'echo done'", x.ns, x.name)
		var parts []string
		if x.deadline != "" {
			parts = append(parts, fmt.Sprintf("activeDeadlineSeconds=%s", x.deadline))
		}
		if x.bl != "" {
			parts = append(parts, fmt.Sprintf("backoffLimit=%s", x.bl))
		}
		if x.ttl != "" {
			parts = append(parts, fmt.Sprintf("ttlSecondsAfterFinished=%s", x.ttl))
		}
		if x.comp != "" {
			parts = append(parts, fmt.Sprintf("completions=%s, parallelism=%s", x.comp, x.par))
		}
		if len(parts) > 0 {
			task += " with " + strings.Join(parts, ", ")
		}
		task += "."

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Job %s exists", x.name), 1,
				"get job "+x.name+" -n "+x.ns+" -o name", "job.batch/"+x.name),
			gcr("restart", "restartPolicy Never or OnFailure", 1,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.restartPolicy}", `Never|OnFailure`),
		}
		w := 3
		if x.deadline != "" {
			checks = append(checks, gcr("deadline", fmt.Sprintf("activeDeadlineSeconds=%s", x.deadline), 2,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.activeDeadlineSeconds}", "^"+x.deadline+"$"))
			w++
		}
		if x.bl != "" {
			checks = append(checks, gcr("backoff", fmt.Sprintf("backoffLimit=%s", x.bl), 2,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.backoffLimit}", "^"+x.bl+"$"))
			w++
		}
		if x.ttl != "" {
			checks = append(checks, gcr("ttl", fmt.Sprintf("ttlSecondsAfterFinished=%s", x.ttl), 2,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.ttlSecondsAfterFinished}", "^"+x.ttl+"$"))
			w++
		}
		if x.comp != "" {
			checks = append(checks, gcr("comp", fmt.Sprintf("completions=%s", x.comp), 1,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.completions}", "^"+x.comp+"$"))
			checks = append(checks, gcr("par", fmt.Sprintf("parallelism=%s", x.par), 1,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.parallelism}", "^"+x.par+"$"))
			w += 2
		}

		out = append(out, gq(
			fmt.Sprintf("qg-hjobadv-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Tune Job %s", x.name),
			"Jobs expose deadline, retry and garbage-collection knobs that exams love.",
			task, solution, x.ns,
			genHints(
				"All tuning fields sit directly under spec of the Job (not the template).",
				"activeDeadlineSeconds kills the whole Job once elapsed.",
			),
			checks,
		))
		_ = w
	}
	return out
}

func genHardCronJobsAdvanced() []*models.Question {
	type v struct {
		ns, name, schedule, conc, sds, shist, fhist string
	}
	variants := []v{
		{"ckad-hcron01", "guard-cron", "*/2 * * * *", "Forbid", "", "2", ""},
		{"ckad-hcron02", "late-cron", "*/5 * * * *", "", "120", "", ""},
		{"ckad-hcron03", "swap-cron", "*/10 * * * *", "Replace", "", "", "1"},
		{"ckad-hcron04", "lean-cron", "@hourly", "", "", "0", "0"},
		{"ckad-hcron05", "tight-cron", "*/3 * * * *", "Forbid", "60", "1", "1"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		extra := ""
		if x.conc != "" {
			extra += fmt.Sprintf("  concurrencyPolicy: %s\n", x.conc)
		}
		if x.sds != "" {
			extra += fmt.Sprintf("  startingDeadlineSeconds: %s\n", x.sds)
		}
		if x.shist != "" {
			extra += fmt.Sprintf("  successfulJobsHistoryLimit: %s\n", x.shist)
		}
		if x.fhist != "" {
			extra += fmt.Sprintf("  failedJobsHistoryLimit: %s\n", x.fhist)
		}
		solution := fmt.Sprintf(`apiVersion: batch/v1
kind: CronJob
metadata:
  name: %s
  namespace: %s
spec:
  schedule: '%s'
%sextra-job-template`, x.name, x.ns, x.schedule, extra)
		solution = strings.Replace(solution, "extra-job-template", `  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
          - name: tick
            image: busybox:1.36
            command: ['sh','-c','date']`, 1)

		task := fmt.Sprintf("In namespace %s, create a CronJob named '%s' (busybox:1.36) with schedule '%s'", x.ns, x.name, x.schedule)
		var parts []string
		if x.conc != "" {
			parts = append(parts, fmt.Sprintf("concurrencyPolicy=%s", x.conc))
		}
		if x.sds != "" {
			parts = append(parts, fmt.Sprintf("startingDeadlineSeconds=%s", x.sds))
		}
		if x.shist != "" {
			parts = append(parts, fmt.Sprintf("successfulJobsHistoryLimit=%s", x.shist))
		}
		if x.fhist != "" {
			parts = append(parts, fmt.Sprintf("failedJobsHistoryLimit=%s", x.fhist))
		}
		if len(parts) > 0 {
			task += ", " + strings.Join(parts, ", ")
		}
		task += "."

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("CronJob %s exists", x.name), 1,
				"get cronjob "+x.name+" -n "+x.ns+" -o name", "cronjob.batch/"+x.name),
			gcs("schedule", fmt.Sprintf("Schedule is %s", x.schedule), 2,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.schedule}", x.schedule),
		}
		if x.conc != "" {
			checks = append(checks, gcs("conc", fmt.Sprintf("concurrencyPolicy=%s", x.conc), 1,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.concurrencyPolicy}", x.conc))
		}
		if x.sds != "" {
			checks = append(checks, gcr("sds", fmt.Sprintf("startingDeadlineSeconds=%s", x.sds), 1,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.startingDeadlineSeconds}", "^"+x.sds+"$"))
		}
		if x.shist != "" {
			checks = append(checks, gcr("shist", fmt.Sprintf("successfulJobsHistoryLimit=%s", x.shist), 1,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.successfulJobsHistoryLimit}", "^"+x.shist+"$"))
		}
		if x.fhist != "" {
			checks = append(checks, gcr("fhist", fmt.Sprintf("failedJobsHistoryLimit=%s", x.fhist), 1,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.failedJobsHistoryLimit}", "^"+x.fhist+"$"))
		}

		out = append(out, gq(
			fmt.Sprintf("qg-hcronadv-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Fine-tune CronJob %s", x.name),
			"CronJobs support overlap policies, deadlines and history trimming.",
			task, solution, x.ns,
			genHints(
				"Tuning fields live directly under spec, next to schedule.",
				"'kubectl create cronjob ... --schedule' scaffolds; then edit the YAML.",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------------- rollout strategies

func genHardDeployStrategies() []*models.Question {
	type v struct {
		ns, dep, stype, surge, unavail, hist string
	}
	variants := []v{
		{"ckad-hstrat01", "recreate-app", "Recreate", "", "", ""},
		{"ckad-hstrat02", "surge-app", "RollingUpdate", "2", "0", ""},
		{"ckad-hstrat03", "pct-app", "RollingUpdate", "25%", "25%", ""},
		{"ckad-hstrat04", "fast-app", "RollingUpdate", "1", "2", ""},
		{"ckad-hstrat05", "leanhist-app", "Recreate", "", "", "3"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=4", x.dep, x.ns),
		}}
		strategyBlock := fmt.Sprintf("  strategy:\n    type: %s\n", x.stype)
		if x.stype == "RollingUpdate" {
			strategyBlock += fmt.Sprintf("    rollingUpdate:\n      maxSurge: %s\n      maxUnavailable: %s\n", x.surge, x.unavail)
		}
		if x.hist != "" {
			strategyBlock += fmt.Sprintf("  revisionHistoryLimit: %s\n", x.hist)
		}
		solution := fmt.Sprintf(`# Save, edit spec, then: kubectl apply -f deploy.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
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
        image: nginx:1.25`, x.dep, x.ns, strategyBlock, x.dep, x.dep)

		task := fmt.Sprintf("In namespace %s, configure the existing Deployment '%s' to use the %s rollout strategy", x.ns, x.dep, x.stype)
		if x.stype == "RollingUpdate" {
			task += fmt.Sprintf(" with maxSurge=%s and maxUnavailable=%s", x.surge, x.unavail)
		}
		if x.hist != "" {
			task += fmt.Sprintf(", keeping revisionHistoryLimit=%s", x.hist)
		}
		task += "."

		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
				"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
			gcs("stype", fmt.Sprintf("strategy.type=%s", x.stype), 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.strategy.type}", x.stype),
		}
		if x.surge != "" {
			checks = append(checks, gcr("surge", fmt.Sprintf("maxSurge=%s", x.surge), 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.strategy.rollingUpdate.maxSurge}", "^"+x.surge+"$"))
			checks = append(checks, gcr("unavail", fmt.Sprintf("maxUnavailable=%s", x.unavail), 2,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.strategy.rollingUpdate.maxUnavailable}", "^"+x.unavail+"$"))
		}
		if x.hist != "" {
			checks = append(checks, gcr("hist", fmt.Sprintf("revisionHistoryLimit=%s", x.hist), 1,
				"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.revisionHistoryLimit}", "^"+x.hist+"$"))
		}

		out = append(out, gqp(
			fmt.Sprintf("qg-hstrat-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyHard,
			fmt.Sprintf("Control rollouts of %s", x.dep),
			"RollingUpdate surge/unavailable values trade speed against availability.",
			task, solution, x.ns, prepare,
			genHints(
				"kubectl edit deploy or apply an updated YAML both work.",
				"maxSurge/maxUnavailable accept counts (2) or percentages (25%).",
			),
			checks,
		))
	}
	return out
}

// ------------------------------------------------- network policy rules

func genHardNetpolRules() []*models.Question {
	type v struct {
		ns, np, selKey, selVal, dir, peerKey, peerVal, port string
	}
	variants := []v{
		{"ckad-hnp01", "fe-to-be", "app", "backend", "ingress", "app", "frontend", "8080"},
		{"ckad-hnp02", "mon-to-api", "app", "api", "ingress", "role", "monitoring", "9090"},
		{"ckad-hnp03", "be-to-db", "app", "db-client", "egress", "app", "database", "5432"},
		{"ckad-hnp04", "wk-to-cache", "tier", "worker", "egress", "tier", "cache", "6379"},
		{"ckad-hnp05", "gw-to-pay", "app", "payments", "ingress", "app", "gateway", "8443"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		listField := x.dir
		fromTo := "from"
		dirWord := "Ingress"
		if x.dir == "egress" {
			fromTo = "to"
			dirWord = "Egress"
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
  policyTypes:
  - %s
  %s:
  - %s:
    - podSelector:
        matchLabels:
          %s: %s
    ports:
    - protocol: TCP
      port: %s`,
			x.np, x.ns, x.selKey, x.selVal, dirWord, listField, fromTo, x.peerKey, x.peerVal, x.port)

		task := fmt.Sprintf("In namespace %s, create a NetworkPolicy named '%s': Pods labeled %s=%s may only receive/send traffic (%s direction) to/from Pods labeled %s=%s on TCP port %s. Include policyTypes [%s].",
			x.ns, x.np, x.selKey, x.selVal, x.dir, x.peerKey, x.peerVal, x.port, dirWord)

		out = append(out, gq(
			fmt.Sprintf("qg-hnprule-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Rule-based NetworkPolicy %s", x.np),
			"Real NetworkPolicies combine podSelectors with port restrictions.",
			task, solution, x.ns,
			genHints(
				fmt.Sprintf("Under spec.%s[0], use %s[0].podSelector.matchLabels for the peer.", listField, fromTo),
				"ports entries carry protocol and port.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("NetworkPolicy %s exists", x.np), 1,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o name", "networkpolicy.networking.k8s.io/"+x.np),
				gcs("sel", fmt.Sprintf("Selects Pods %s=%s", x.selKey, x.selVal), 1,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.podSelector.matchLabels."+x.selKey+"}", x.selVal),
				gcs("types", fmt.Sprintf("policyTypes includes %s", dirWord), 1,
					"get networkpolicy "+x.np+" -n "+x.ns+" -o jsonpath={.spec.policyTypes}", dirWord),
				gcs("peer", fmt.Sprintf("%s peer %s=%s", x.dir, x.peerKey, x.peerVal), 2,
					fmt.Sprintf("get networkpolicy %s -n %s -o jsonpath={.spec.%s[0].%s[0].podSelector.matchLabels.%s}", x.np, x.ns, listField, fromTo, x.peerKey), x.peerVal),
				gcr("port", fmt.Sprintf("TCP port %s allowed", x.port), 2,
					fmt.Sprintf("get networkpolicy %s -n %s -o jsonpath={.spec.%s[0].ports[0].port}", x.np, x.ns, listField), "^"+x.port+"$"),
			},
		))
	}
	return out
}

// ------------------------------------------------ multi-path ingress

func genHardIngressMultiPath() []*models.Question {
	type v struct {
		ns, ing, host, p1, s1, p2, s2 string
	}
	variants := []v{
		{"ckad-hing01", "shop-ing", "shop.example.com", "/api", "svc-api", "/web", "svc-web"},
		{"ckad-hing02", "portal-ing", "portal.example.com", "/auth", "svc-auth", "/static", "svc-assets"},
		{"ckad-hing03", "dev-ing", "dev.example.com", "/v1", "svc-v1", "/v2", "svc-v2"},
		{"ckad-hing04", "media-ing", "media.example.com", "/video", "svc-video", "/audio", "svc-audio"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{Name: "create backing services", CommandArgs: fmt.Sprintf("create deployment dummy --image=nginx:1.25 -n %s", x.ns)},
			{Name: "expose " + x.s1, CommandArgs: fmt.Sprintf("expose deployment dummy --port=80 --name=%s -n %s", x.s1, x.ns)},
			{Name: "expose " + x.s2, CommandArgs: fmt.Sprintf("expose deployment dummy --port=80 --name=%s -n %s", x.s2, x.ns)},
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
      - path: %s
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: 80
      - path: %s
        pathType: Prefix
        backend:
          service:
            name: %s
            port:
              number: 80`, x.ing, x.ns, x.host, x.p1, x.s1, x.p2, x.s2)

		out = append(out, gqp(
			fmt.Sprintf("qg-hingpath-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Multi-path Ingress %s", x.ing),
			"One Ingress host can fan out to different Services per path prefix.",
			fmt.Sprintf("In namespace %s (Services '%s' and '%s' already exist), create an Ingress named '%s' for host %s routing path %s to '%s' and path %s to '%s' (both Prefix, port 80).",
				x.ns, x.s1, x.s2, x.ing, x.host, x.p1, x.s1, x.p2, x.s2),
			solution, x.ns, prepare,
			genHints(
				"http.paths is an ordered list — put both entries under the same host rule.",
				"pathType Prefix matches the path and anything below it.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Ingress %s exists", x.ing), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o name", "ingress.networking.k8s.io/"+x.ing),
				gcs("host", fmt.Sprintf("Host is %s", x.host), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].host}", x.host),
				gcs("p1", fmt.Sprintf("Path %s routes to %s", x.p1, x.s1), 2,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[0].path}", x.p1),
				gcs("b1", fmt.Sprintf("Backend of %s is %s", x.p1, x.s1), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[0].backend.service.name}", x.s1),
				gcs("p2", fmt.Sprintf("Path %s routes to %s", x.p2, x.s2), 2,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[1].path}", x.p2),
				gcs("b2", fmt.Sprintf("Backend of %s is %s", x.p2, x.s2), 1,
					"get ingress "+x.ing+" -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[1].backend.service.name}", x.s2),
			},
		))
	}
	return out
}

// ---------------------------------------------------- secret volumes

func genHardSecretVolumes() []*models.Question {
	type v struct {
		ns, pod, sec, mnt string
		readOnly          bool
		items             bool
	}
	variants := []v{
		{"ckad-hsvol01", "cred-app", "app-creds", "/etc/creds", true, false},
		{"ckad-hsvol02", "tls-app", "tls-bundle", "/etc/tls", true, false},
		{"ckad-hsvol03", "tok-app", "service-token", "/var/run/tokens", false, false},
		{"ckad-hsvol04", "cfg-app", "full-secret", "/etc/app", true, true},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		mountExtra := ""
		if x.readOnly {
			mountExtra = "\n      readOnly: true"
		}
		volBody := fmt.Sprintf("    secret:\n      secretName: %s", x.sec)
		if x.items {
			volBody += "\n      items:\n      - key: username\n        path: user.txt"
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
    - {name: secrets, mountPath: %s}%s
  volumes:
  - name: secrets
%s`, x.pod, x.ns, x.mnt, mountExtra, volBody)

		task := fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) that mounts the existing Secret '%s' as a volume at %s", x.ns, x.pod, x.sec, x.mnt)
		if x.readOnly {
			task += " with readOnly: true"
		}
		if x.items {
			task += "; project only key 'username' into file user.txt using items"
		}
		task += ". First create the Secret yourself with key username=admin (--from-literal)."

		prepare := []models.SetupStep{{
			Name:        "create secret " + x.sec,
			CommandArgs: fmt.Sprintf("create secret generic %s --from-literal=username=admin --from-literal=password=hunter2 -n %s", x.sec, x.ns),
		}}

		checks := []models.Check{
			gcs("pod", fmt.Sprintf("Pod %s exists", x.pod), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
			gcs("secret-ref", fmt.Sprintf("Volume references Secret %s", x.sec), 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].secret.secretName}", x.sec),
			gcs("mount", fmt.Sprintf("Mounted at %s", x.mnt), 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].mountPath}", x.mnt),
		}
		if x.readOnly {
			checks = append(checks, gcs("readonly", "Mount is readOnly", 1,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].readOnly}", "true"))
		}
		if x.items {
			checks = append(checks, gcs("items", "items projects username to user.txt", 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].secret.items[0].path}", "user.txt"))
		}

		out = append(out, gqp(
			fmt.Sprintf("qg-hsecvol-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Mount Secret %s as a volume", x.sec),
			"Secrets can be projected into Pods as files instead of environment variables.",
			task, solution, x.ns, prepare,
			genHints(
				"Volume type 'secret' references the Secret by secretName.",
				"items lets you pick keys and rename them via path.",
			),
			checks,
		))
	}
	return out
}

// --------------------------------------------- ambassador pattern

func genHardAmbassador() []*models.Question {
	type v struct {
		ns, pod, amb, remoteSvc, ambPort string
	}
	variants := []v{
		{"ckad-hmulti01", "dbclient", "db-proxy", "redis-master", "15000"},
		{"ckad-hmulti02", "apiclient", "api-forward", "remote-api", "15001"},
		{"ckad-hmulti03", "mqclient", "mq-relay", "remote-mq", "15002"},
		{"ckad-hmulti04", "storeclient", "store-sidecar", "remote-store", "15003"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create remote service " + x.remoteSvc,
			CommandArgs: fmt.Sprintf("create deployment remote --image=redis:7.2 -n %s && expose deployment remote --port=6379 --name=%s -n %s", x.ns, x.remoteSvc, x.ns),
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
  - name: %s
    image: busybox:1.36
    command: ['sh','-c','nc -lk -p %s']
    ports:
    - {containerPort: %s}`, x.pod, x.ns, x.amb, x.ambPort, x.ambPort)

		out = append(out, gqp(
			fmt.Sprintf("qg-hamb-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Ambassador pattern for %s", x.pod),
			"An ambassador container proxies localhost traffic to an external Service.",
			fmt.Sprintf("In namespace %s (Service '%s' already exists), create a Pod named '%s' with two containers: 'app' (busybox:1.36, sleeps forever) and ambassador '%s' (busybox:1.36) exposing containerPort %s. The app should talk to %s via localhost:%s.",
				x.ns, x.remoteSvc, x.pod, x.amb, x.ambPort, x.remoteSvc, x.ambPort),
			solution, x.ns, prepare,
			genHints(
				"The ambassador is just a second container in the same Pod.",
				"Apps connect to localhost:<port> without knowing the remote Service.",
			),
			[]models.Check{
				gcr("app", "Main container app exists", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )app( |$)`),
				gcr("amb", fmt.Sprintf("Ambassador %s exists", x.amb), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )`+x.amb+`( |$)`),
				gcr("port", fmt.Sprintf("Ambassador exposes port %s", x.ambPort), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[1].ports[0].containerPort}", "^"+x.ambPort+"$"),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- HPA

func genHardHPA() []*models.Question {
	type v struct {
		ns, dep   string
		min, max  string
		cpuTarget string
	}
	variants := []v{
		{"ckad-hhpa01", "bursty-api", "2", "6", "75"},
		{"ckad-hhpa02", "spiky-web", "1", "10", "60"},
		{"ckad-hhpa03", "elastic-fe", "3", "8", "80"},
		{"ckad-hhpa04", "auto-worker", "2", "5", "70"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=%s", x.dep, x.ns, x.min),
		}}
		solution := fmt.Sprintf("kubectl autoscale deployment %s --min=%s --max=%s --cpu-percent=%s -n %s",
			x.dep, x.min, x.max, x.cpuTarget, x.ns)
		out = append(out, gqp(
			fmt.Sprintf("qg-hhpa-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyMedium,
			fmt.Sprintf("Autoscale %s", x.dep),
			"The HorizontalPodAutoscaler scales replica counts on CPU (or custom) metrics.",
			fmt.Sprintf("In namespace %s, create an HorizontalPodAutoscaler for Deployment '%s' with min=%s, max=%s replicas targeting %s%% CPU utilization.",
				x.ns, x.dep, x.min, x.max, x.cpuTarget),
			solution, x.ns, prepare,
			genHints(
				"'kubectl autoscale deployment NAME --min --max --cpu-percent' does it imperatively.",
				"The HPA object is named after the target Deployment.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("HPA for %s exists", x.dep), 2,
					"get hpa "+x.dep+" -n "+x.ns+" -o name", "horizontalpodautoscaler.autoscaling/"+x.dep),
				gcr("min", fmt.Sprintf("minReplicas=%s", x.min), 1,
					"get hpa "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.minReplicas}", "^"+x.min+"$"),
				gcr("max", fmt.Sprintf("maxReplicas=%s", x.max), 1,
					"get hpa "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.maxReplicas}", "^"+x.max+"$"),
				gcr("cpu", fmt.Sprintf("targetCPUUtilizationPercentage=%s", x.cpuTarget), 2,
					"get hpa "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.targetCPUUtilizationPercentage}", "^"+x.cpuTarget+"$"),
			},
		))
	}
	return out
}

// ------------------------------------------------ advanced services

func genHardServicesAdvanced() []*models.Question {
	type v struct {
		ns, svc, mode, arg1, arg2 string
	}
	variants := []v{
		{"ckad-hsvc01", "headless-svc", "headless", "", ""},
		{"ckad-hsvc02", "sticky-svc", "sessionAffinity", "ClientIP", ""},
		{"ckad-hsvc03", "dualport-svc", "multiport", "80:8080", "443:8443"},
		{"ckad-hsvc04", "ext-dns", "externalName", "db.example.com", ""},
		{"ckad-hsvc05", "mapped-svc", "externalIPs", "192.168.5.10", ""},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var solution, task string
		var checks []models.Check
		switch x.mode {
		case "headless":
			task = fmt.Sprintf("In namespace %s, create a headless Service named '%s' selecting app=web with port 80->targetPort 8080 (clusterIP: None).", x.ns, x.svc)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  clusterIP: None
  selector:
    app: web
  ports:
  - port: 80
    targetPort: 8080`, x.svc, x.ns)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcs("clusterip", "clusterIP is None", 3,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.clusterIP}", "None"),
				gcs("port", "Service port 80", 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", "80"),
			}
		case "sessionAffinity":
			task = fmt.Sprintf("In namespace %s, create a Service named '%s' (selecting app=web) with port 80 and sessionAffinity set to ClientIP.", x.ns, x.svc)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: web
  sessionAffinity: ClientIP
  ports:
  - port: 80`, x.svc, x.ns)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcs("affinity", "sessionAffinity=ClientIP", 3,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.sessionAffinity}", "ClientIP"),
			}
		case "multiport":
			p1 := strings.SplitN(x.arg1, ":", 2)
			p2 := strings.SplitN(x.arg2, ":", 2)
			task = fmt.Sprintf("In namespace %s, create a Service named '%s' selecting app=web exposing TWO ports: %s->%s (name http) and %s->%s (name https).", x.ns, x.svc, p1[0], p1[1], p2[0], p2[1])
			solution = fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: web
  ports:
  - {name: http, port: %s, targetPort: %s}
  - {name: https, port: %s, targetPort: %s}`, x.svc, x.ns, p1[0], p1[1], p2[0], p2[1])
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcr("port1", fmt.Sprintf("First port is %s", p1[0]), 2,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", "^"+p1[0]+"$"),
				gcr("port2", fmt.Sprintf("Second port is %s", p2[0]), 2,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[1].port}", "^"+p2[0]+"$"),
				gcs("name1", "First port named http", 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[0].name}", "http"),
				gcs("name2", "Second port named https", 1,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.ports[1].name}", "https"),
			}
		case "externalName":
			task = fmt.Sprintf("In namespace %s, create an ExternalName Service named '%s' pointing at %s.", x.ns, x.svc, x.arg1)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  type: ExternalName
  externalName: %s`, x.svc, x.ns, x.arg1)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcs("type", "type is ExternalName", 2,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.type}", "ExternalName"),
				gcs("external", fmt.Sprintf("externalName is %s", x.arg1), 2,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.externalName}", x.arg1),
			}
		default: // externalIPs
			task = fmt.Sprintf("In namespace %s, create a Service named '%s' selecting app=web (port 80) that also lists external IP %s in spec.externalIPs.", x.ns, x.svc, x.arg1)
			solution = fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
spec:
  selector:
    app: web
  externalIPs:
  - %s
  ports:
  - port: 80`, x.svc, x.ns, x.arg1)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svc), 1,
					"get svc "+x.svc+" -n "+x.ns+" -o name", "service/"+x.svc),
				gcs("extip", fmt.Sprintf("externalIPs includes %s", x.arg1), 3,
					"get svc "+x.svc+" -n "+x.ns+" -o jsonpath={.spec.externalIPs[*]}", x.arg1),
			}
		}
		out = append(out, gq(
			fmt.Sprintf("qg-hsvcadv-%02d", i+1), models.DomainServicesNetworking, models.DifficultyHard,
			fmt.Sprintf("Advanced Service: %s", x.mode),
			"Services have modes beyond plain ClusterIP: headless, sticky, multi-port, ExternalName.",
			task, solution, x.ns,
			genHints(
				"These variants usually need hand-written YAML rather than kubectl expose.",
				"Check your work with 'kubectl get svc NAME -o yaml'.",
			),
			checks,
		))
	}
	return out
}

// ----------------------------------------------------- exec probes

func genHardProbesExec() []*models.Question {
	type v struct {
		ns, pod, kind, cmd1, cmd2, period string
	}
	variants := []v{
		{"ckad-hprobe01", "flagged", "livenessProbe", "cat", "/tmp/ready", "10"},
		{"ckad-hprobe02", "tested", "readinessProbe", "test", "-f /tmp/live", "15"},
		{"ckad-hprobe03", "scripted", "livenessProbe", "sh", "-c echo ok", "20"},
		{"ckad-hprobe04", "polled", "readinessProbe", "ls", "/healthcheck", "5"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		kindName := "Liveness"
		if x.kind == "readinessProbe" {
			kindName = "Readiness"
		}
		cmdArg := "'" + x.cmd2 + "'"
		if !strings.Contains(x.cmd2, " ") {
			cmdArg = x.cmd2
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
    %s:
      exec:
        command: ['%s', '%s']
      initialDelaySeconds: 5
      periodSeconds: %s`, x.pod, x.ns, x.kind, x.cmd1, x.cmd2, x.period)

		out = append(out, gq(
			fmt.Sprintf("qg-hexpprobe-%02d", i+1), models.DomainApplicationObservability, models.DifficultyHard,
			fmt.Sprintf("%s exec probe on %s", kindName, x.pod),
			"Exec probes run a command inside the container instead of hitting an HTTP endpoint.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) with a %s of type exec running '%s %s', periodSeconds=%s and initialDelaySeconds=5.",
				x.ns, x.pod, kindName, x.cmd1, x.cmd2, x.period),
			solution, x.ns,
			genHints(
				"A probe without httpGet/tcpSocket defaults to exec when you supply command.",
				"command is a list of strings, argv-style.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("cmd1", fmt.Sprintf("Probe command starts with %s", x.cmd1), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0]."+x.kind+".exec.command[0]}", x.cmd1),
				gcr("period", fmt.Sprintf("periodSeconds=%s", x.period), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0]."+x.kind+".periodSeconds}", "^"+x.period+"$"),
			},
		))
		_ = cmdArg
	}
	return out
}
