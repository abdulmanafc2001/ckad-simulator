package main

import (
	"fmt"

	"github.com/manaf/ckad-simulator/backend/internal/models"
)

// This file programmatically generates the bulk of the CKAD question bank.
// Every generated question is self-contained exactly like the hand-written
// ones: it provisions its own uniquely-named namespace, is solved against
// the live cluster, is graded with weighted kubectl checks, and cleans up
// after itself.

// gq assembles a generated question; total weight = sum of check weights.
func gq(id string, domain models.Domain, diff models.Difficulty, title, desc, task, solution, ns string, hints []string, checks []models.Check) *models.Question {
	w := 0
	for _, c := range checks {
		w += c.Weight
	}
	return &models.Question{
		ID:          id,
		Domain:      domain,
		Difficulty:  diff,
		Title:       title,
		Description: desc,
		Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace " + ns}},
		Task:        task,
		Hints:       hints,
		Solution:    solution,
		Checks:      checks,
		Cleanup:     []string{"delete namespace " + ns + " --ignore-not-found"},
		Weight:      w,
	}
}

// gqp is gq with additional environment-preparation steps. The namespace is
// always created first (gq seeds that step), then the caller's steps run, so
// resources that target the namespace find it already present.
func gqp(id string, domain models.Domain, diff models.Difficulty, title, desc, task, solution, ns string, prepare []models.SetupStep, hints []string, checks []models.Check) *models.Question {
	q := gq(id, domain, diff, title, desc, task, solution, ns, hints, checks)
	q.Prepare = append(q.Prepare, prepare...)
	return q
}

// gcs builds a substring expectation check.
func gcs(id, desc string, w int, args, want string) models.Check {
	return models.Check{ID: id, Description: desc, Weight: w, CommandArgs: args, ExpectSubstring: want}
}

// gcr builds a regex expectation check.
func gcr(id, desc string, w int, args, pattern string) models.Check {
	return models.Check{ID: id, Description: desc, Weight: w, CommandArgs: args, ExpectRegex: pattern}
}

func genHints(h1, h2 string) []string { return []string{h1, h2} }

func generatedQuestions() []*models.Question {
	var qs []*models.Question
	qs = append(qs, genSimplePods()...)
	qs = append(qs, genPodsWithLabels()...)
	qs = append(qs, genPodsWithCommands()...)
	qs = append(qs, genPodsWithEnv()...)
	qs = append(qs, genSidecars()...)
	qs = append(qs, genInitContainers()...)
	qs = append(qs, genJobs()...)
	qs = append(qs, genCronJobs()...)
	qs = append(qs, genPVCs()...)
	qs = append(qs, genPVCMounts()...)
	qs = append(qs, genDeployments()...)
	qs = append(qs, genScaling()...)
	qs = append(qs, genSetImage()...)
	qs = append(qs, genRollbacks()...)
	qs = append(qs, genLabelAnnotate()...)
	qs = append(qs, genProbes()...)
	qs = append(qs, genResources()...)
	qs = append(qs, genConfigMaps()...)
	qs = append(qs, genCMConsumers()...)
	qs = append(qs, genSecrets()...)
	qs = append(qs, genSecretConsumers()...)
	qs = append(qs, genServiceAccounts()...)
	qs = append(qs, genQuotasAndLR()...)
	qs = append(qs, genNamespaces()...)
	qs = append(qs, genServices()...)
	qs = append(qs, genIngresses()...)
	qs = append(qs, genNetworkPolicies()...)
	qs = append(qs, genHardScheduling()...)
	qs = append(qs, genHardSecurityContexts()...)
	qs = append(qs, genHardJobsAdvanced()...)
	qs = append(qs, genHardCronJobsAdvanced()...)
	qs = append(qs, genHardDeployStrategies()...)
	qs = append(qs, genHardNetpolRules()...)
	qs = append(qs, genHardIngressMultiPath()...)
	qs = append(qs, genHardSecretVolumes()...)
	qs = append(qs, genHardAmbassador()...)
	qs = append(qs, genHardHPA()...)
	qs = append(qs, genHardServicesAdvanced()...)
	qs = append(qs, genHardProbesExec()...)
	qs = append(qs, genP5TcpProbes()...)
	qs = append(qs, genP5HttpProbeTuning()...)
	qs = append(qs, genP5ResourceCombos()...)
	qs = append(qs, genP5EnvFrom()...)
	qs = append(qs, genP5SubPath()...)
	qs = append(qs, genP5EmptyDirVariants()...)
	qs = append(qs, genP5PVCAdvanced()...)
	qs = append(qs, genP5DepAdvancedFields()...)
	qs = append(qs, genP5StatefulSets()...)
	qs = append(qs, genP5DaemonSets()...)
	qs = append(qs, genP5CronSchedules()...)
	qs = append(qs, genP5NamedPortServices()...)
	qs = append(qs, genP5IpBlockNetpol()...)
	qs = append(qs, genP5SaBinding()...)
	qs = append(qs, genP5MatchExpressions()...)
	qs = append(qs, genP5NodeAffinity()...)
	qs = append(qs, genP5DownwardAPI()...)
	qs = append(qs, genP5LifecycleHooks()...)
	qs = append(qs, genP5PullPolicySecrets()...)
	qs = append(qs, genP5RestartPolicies()...)
	qs = append(qs, genP6UdpPorts()...)
	qs = append(qs, genP6HostPath()...)
	qs = append(qs, genP6MultiInitChains()...)
	qs = append(qs, genP6ArgInterpolation()...)
	qs = append(qs, genP6QuotaScopes()...)
	qs = append(qs, genP6PodAnnotations()...)
	qs = append(qs, genP6JobPatterns()...)
	qs = append(qs, genP6CronConcurrency()...)
	qs = append(qs, genP6ProbeThresholds()...)
	qs = append(qs, genP6OddUnits()...)
	qs = append(qs, genP6MultiLabelWorkloads()...)
	qs = append(qs, genP6RolloutToRevision()...)
	qs = append(qs, genP6ExposeVariants()...)
	qs = append(qs, genP6SetCommands()...)
	qs = append(qs, genP6TolerationMore()...)
	qs = append(qs, genP6CMImmutable()...)
	qs = append(qs, genP6SecretStringData()...)
	qs = append(qs, genP6HostAliases()...)
	qs = append(qs, genP6DnsSettings()...)
	qs = append(qs, genP6GracePeriod()...)
	qs = append(qs, genP6NamedContainerPorts()...)
	qs = append(qs, genP6ShareProcess()...)
	qs = append(qs, genP6MoreSimplePods()...)
	qs = append(qs, genP6MoreLabeledPods()...)
	qs = append(qs, genP6MoreDeployments()...)
	qs = append(qs, genP6MoreScaling()...)
	qs = append(qs, genP6MoreSetImage()...)
	qs = append(qs, genP6MorePVCMounts()...)
	return qs
}

// ---------------------------------------------------------------- pods

func genSimplePods() []*models.Question {
	type v struct{ ns, name, image, port string }
	variants := []v{
		{"ckad-gpod01", "cache", "redis:7.2", "6379"},
		{"ckad-gpod02", "auth", "auth-app:v2", "8080"},
		{"ckad-gpod03", "metrics", "exporter:1.0", "9090"},
		{"ckad-gpod04", "logger", "fluentd:1.16", "24224"},
		{"ckad-gpod05", "proxy", "envoy:v1.28", "10000"},
		{"ckad-gpod06", "queue", "rabbitmq:3.13", "5672"},
		{"ckad-gpod07", "search", "elastic:8.13", "9200"},
		{"ckad-gpod08", "db", "postgres:16.2", "5432"},
		{"ckad-gpod09", "mail", "smtp-relay:1.4", "25"},
		{"ckad-gpod10", "dns", "coredns:1.11", "53"},
		{"ckad-gpod11", "ui", "dashboard:v5", "3000"},
		{"ckad-gpod12", "worker", "buildkit:0.13", "5000"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if i%3 == 2 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-pod-%02d", i+1), models.DomainApplicationDesign, diff,
			fmt.Sprintf("Run the %s Pod", x.name),
			"Single-container Pods are the smallest deployable unit in Kubernetes.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' using the image '%s' that exposes container port %s.", x.ns, x.name, x.image, x.port),
			fmt.Sprintf("kubectl run %s --image=%s --port=%s -n %s", x.name, x.image, x.port, x.ns),
			x.ns,
			genHints(
				"Use 'kubectl run' with --image and --port.",
				"Verify with 'kubectl get pod "+x.name+" -n "+x.ns+" -o yaml'.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists in %s", x.name, x.ns), 1,
					"get pod "+x.name+" -n "+x.ns+" -o name", "pod/"+x.name),
				gcs("image", fmt.Sprintf("Uses image %s", x.image), 2,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].image}", x.image),
				gcr("port", fmt.Sprintf("Exposes container port %s", x.port), 1,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].ports[0].containerPort}", "^"+x.port+"$"),
			},
		))
	}
	return out
}

func genPodsWithLabels() []*models.Question {
	type v struct{ ns, name, image, key, val string }
	variants := []v{
		{"ckad-glbl01", "web-frontend", "nginx:1.25", "tier", "frontend"},
		{"ckad-glbl02", "api-backend", "httpd:2.4", "tier", "backend"},
		{"ckad-glbl03", "etl-worker", "busybox:1.36", "app", "etl"},
		{"ckad-glbl04", "cache-layer", "redis:7.2", "role", "cache"},
		{"ckad-glbl05", "msg-broker", "rabbitmq:3.13", "role", "broker"},
		{"ckad-glbl06", "edge-router", "envoy:v1.28", "layer", "edge"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		out = append(out, gq(
			fmt.Sprintf("qg-podlbl-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("Run a labeled Pod (%s=%s)", x.key, x.val),
			"Labels organize and select groups of objects.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' running image '%s' with the label %s=%s.", x.ns, x.name, x.image, x.key, x.val),
			fmt.Sprintf("kubectl run %s --image=%s -n %s -l %s=%s", x.name, x.image, x.ns, x.key, x.val),
			x.ns,
			genHints(
				"'kubectl run' accepts -l key=value to add labels.",
				"Labels live under metadata.labels in YAML.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.name), 1,
					"get pod "+x.name+" -n "+x.ns+" -o name", "pod/"+x.name),
				gcs("image", "Uses the requested image", 1,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].image}", x.image),
				gcs("label", fmt.Sprintf("Has label %s=%s", x.key, x.val), 2,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.metadata.labels."+x.key+"}", x.val),
			},
		))
	}
	return out
}

func genPodsWithCommands() []*models.Question {
	type v struct{ ns, name, cmd, arg string }
	variants := []v{
		{"ckad-gcmd01", "sleeper", "sleep", "3600"},
		{"ckad-gcmd02", "waiter", "sh", "-c sleep 1800"},
		{"ckad-gcmd03", "ticker", "watch", "date"},
		{"ckad-gcmd04", "greeter", "echo", "hello-from-k8s"},
		{"ckad-gcmd05", "idler", "tail", "-f /dev/null"},
		{"ckad-gcmd06", "clock", "date", "+%H:%M"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		solutionArg := x.arg
		if i == 1 || i == 4 {
			solutionArg = "'" + x.arg + "'"
		}
		out = append(out, gq(
			fmt.Sprintf("qg-podcmd-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("Run a command in a Pod (%s)", x.name),
			"Containers can run arbitrary commands and arguments instead of the image entrypoint.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' using busybox:1.36 that runs the command '%s' with argument '%s'.", x.ns, x.name, x.cmd, x.arg),
			fmt.Sprintf("kubectl run %s --image=busybox:1.36 -n %s -- %s %s", x.name, x.ns, x.cmd, solutionArg),
			x.ns,
			genHints(
				"The '--' separates kubectl flags from the container command.",
				"In YAML these are spec.containers[].command and .args.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.name), 1,
					"get pod "+x.name+" -n "+x.ns+" -o name", "pod/"+x.name),
				gcs("command", fmt.Sprintf("Runs command %s", x.cmd), 2,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].command[0]}", x.cmd),
				gcs("args", "Carries the expected argument", 2,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].args[0]}", firstWord(x.arg)),
			},
		))
	}
	return out
}

func firstWord(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			return s[:i]
		}
	}
	return s
}

func genPodsWithEnv() []*models.Question {
	type v struct{ ns, name, key, val string }
	variants := []v{
		{"ckad-genv01", "app-prod", "APP_ENV", "production"},
		{"ckad-genv02", "app-dev", "APP_ENV", "development"},
		{"ckad-genv03", "svc-url", "API_URL", "https://api.local"},
		{"ckad-genv04", "log-level", "LOG_LEVEL", "debug"},
		{"ckad-genv05", "region", "REGION", "eu-west-1"},
		{"ckad-genv06", "feature-x", "FEATURE_X", "enabled"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		out = append(out, gq(
			fmt.Sprintf("qg-podenv-%02d", i+1), models.DomainApplicationDesign, models.DifficultyEasy,
			fmt.Sprintf("Set an environment variable (%s)", x.key),
			"Environment variables configure container behavior at runtime.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' (busybox:1.36) whose container defines the environment variable %s=%s.", x.ns, x.name, x.key, x.val),
			fmt.Sprintf("kubectl run %s --image=busybox:1.36 -n %s --env=%s=%s", x.name, x.ns, x.key, x.val),
			x.ns,
			genHints(
				"'kubectl run' supports --env KEY=VALUE (repeatable).",
				"In YAML use spec.containers[].env with name/value pairs.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Pod %s exists", x.name), 1,
					"get pod "+x.name+" -n "+x.ns+" -o name", "pod/"+x.name),
				gcs("env-name", fmt.Sprintf("Defines env var %s", x.key), 2,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].name}", x.key),
				gcs("env-value", fmt.Sprintf("Value is %s", x.val), 2,
					"get pod "+x.name+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].value}", x.val),
			},
		))
	}
	return out
}

func genSidecars() []*models.Question {
	type v struct{ ns, app, side, path string }
	variants := []v{
		{"ckad-gside01", "shop", "log-tail", "/var/log/shop"},
		{"ckad-gside02", "payments", "log-ship", "/var/log/pay"},
		{"ckad-gside03", "catalog", "log-watch", "/var/log/cat"},
		{"ckad-gside04", "users", "log-collect", "/var/log/usr"},
		{"ckad-gside05", "orders", "log-drain", "/var/log/ord"},
		{"ckad-gside06", "inventory", "log-sync", "/var/log/inv"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyMedium
		if i%2 == 1 {
			diff = models.DifficultyHard
		}
		task := fmt.Sprintf(
			"In namespace %s, create a Pod named '%s' with two containers sharing an emptyDir volume named 'logs': main container 'app' (nginx:1.25) writing to %s/access.log via command 'sh -c \"while true; do date >> %s/access.log; sleep 5; done\"', and sidecar '%s' (busybox:1.36) mounting the same volume at /var/log and running 'tail -f /var/log/access.log'.",
			x.ns, x.app, x.path, x.path, x.side)
		solution := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: %s
  namespace: %s
spec:
  containers:
  - name: app
    image: nginx:1.25
    volumeMounts:
    - {name: logs, mountPath: %s}
  - name: %s
    image: busybox:1.36
    command: ['sh','-c','tail -f /var/log/access.log']
    volumeMounts:
    - {name: logs, mountPath: /var/log}
  volumes:
  - name: logs
    emptyDir: {}`, x.app, x.ns, x.path, x.side)
		out = append(out, gq(
			fmt.Sprintf("qg-sidecar-%02d", i+1), models.DomainApplicationDesign, diff,
			fmt.Sprintf("Add a log sidecar to %s", x.app),
			"Sidecars share the Pod lifecycle and filesystem to augment a main container.",
			task, solution, x.ns,
			genHints(
				"Both containers mount the same emptyDir volume at different paths.",
				"The sidecar only needs 'tail -f' on the shared log file.",
			),
			[]models.Check{
				gcr("main", "Main container app exists", 1,
					"get pod "+x.app+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )app( |$)`),
				gcr("sidecar", fmt.Sprintf("Sidecar %s exists", x.side), 2,
					"get pod "+x.app+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )`+x.side+`( |$)`),
				gcs("volume", "Shared emptyDir volume logs defined", 1,
					"get pod "+x.app+" -n "+x.ns+" -o jsonpath={.spec.volumes[*].name}", "logs"),
				gcs("main-mount", fmt.Sprintf("app writes into %s", x.path), 1,
					`get pod `+x.app+` -n `+x.ns+` -o jsonpath={.spec.containers[?(@.name=="app")].volumeMounts[0].mountPath}`, x.path),
				gcs("side-mount", "sidecar reads from /var/log", 1,
					`get pod `+x.app+` -n `+x.ns+` -o jsonpath={.spec.containers[?(@.name=="`+x.side+`")].volumeMounts[0].mountPath}`, "/var/log"),
			},
		))
	}
	return out
}

func genInitContainers() []*models.Question {
	type v struct{ ns, pod, init, file, msg string }
	variants := []v{
		{"ckad-ginit01", "bootstrap", "setup", "ready.txt", "ok"},
		{"ckad-ginit02", "seed", "planter", "seed.sql", "insert"},
		{"ckad-ginit03", "warmup", "heater", "cache.warm", "warm"},
		{"ckad-ginit04", "preflight", "check", "passed.flag", "pass"},
		{"ckad-ginit05", "provision", "maker", "done.marker", "done"},
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
    command: ['sh','-c','echo %s > /work/%s']
    volumeMounts:
    - {name: shared, mountPath: /work}
  containers:
  - name: web
    image: nginx:1.25
    volumeMounts:
    - {name: shared, mountPath: /usr/share/nginx/html}
  volumes:
  - name: shared
    emptyDir: {}`, x.pod, x.ns, x.init, x.msg, x.file)
		out = append(out, gq(
			fmt.Sprintf("qg-init-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("Bootstrap %s with an init container", x.pod),
			"Init containers run to completion before app containers start.",
			fmt.Sprintf("In namespace %s, create a Pod named '%s' with an initContainer named '%s' (busybox:1.36) that writes '%s' to /work/%s on an emptyDir volume 'shared', plus a main container 'web' (nginx:1.25) mounting the same volume at /usr/share/nginx/html.", x.ns, x.pod, x.init, x.msg, x.file),
			solution, x.ns,
			genHints(
				"initContainers sits under spec, sibling to containers.",
				"Mount the same emptyDir into both containers.",
			),
			[]models.Check{
				gcr("init", fmt.Sprintf("Init container %s exists", x.init), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.initContainers[*].name}", `(^| )`+x.init+`( |$)`),
				gcr("main", "Main container web exists", 1,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[*].name}", `(^| )web( |$)`),
				gcs("volume", "emptyDir volume shared defined", 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[*].name}", "shared"),
				gcs("mount", "web mounts /usr/share/nginx/html", 1,
					`get pod `+x.pod+` -n `+x.ns+` -o jsonpath={.spec.containers[?(@.name=="web")].volumeMounts[0].mountPath}`, "/usr/share/nginx/html"),
			},
		))
	}
	return out
}

// ------------------------------------------------------- jobs & cronjobs

func genJobs() []*models.Question {
	type v struct{ ns, name, comp, par string }
	variants := []v{
		{"ckad-gjob01", "batch-a", "1", "1"},
		{"ckad-gjob02", "batch-b", "3", "1"},
		{"ckad-gjob03", "batch-c", "5", "2"},
		{"ckad-gjob04", "batch-d", "4", "4"},
		{"ckad-gjob05", "batch-e", "6", "3"},
		{"ckad-gjob06", "batch-f", "2", "2"},
		{"ckad-gjob07", "batch-g", "8", "4"},
		{"ckad-gjob08", "batch-h", "10", "5"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if x.comp != "1" {
			diff = models.DifficultyMedium
		}
		var solution string
		if x.comp == "1" && x.par == "1" {
			solution = fmt.Sprintf("kubectl create job %s --image=busybox:1.36 -n %s -- echo Hello", x.name, x.ns)
		} else {
			solution = fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: %s
spec:
  completions: %s
  parallelism: %s
  template:
    spec:
      restartPolicy: Never
      containers:
      - name: work
        image: busybox:1.36
        command: ['sh','-c','echo Hello']`, x.name, x.ns, x.comp, x.par)
		}
		checks := []models.Check{
			gcs("exists", fmt.Sprintf("Job %s exists", x.name), 2,
				"get job "+x.name+" -n "+x.ns+" -o name", "job.batch/"+x.name),
			gcs("restart", "restartPolicy Never or OnFailure", 1,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.restartPolicy}", ""),
		}
		checks[1].ExpectRegex = `Never|OnFailure`
		if x.comp != "1" {
			checks = append(checks, gcr("completions", fmt.Sprintf("completions=%s", x.comp), 2,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.completions}", "^"+x.comp+"$"))
		}
		if x.par != "1" {
			checks = append(checks, gcr("parallelism", fmt.Sprintf("parallelism=%s", x.par), 2,
				"get job "+x.name+" -n "+x.ns+" -o jsonpath={.spec.parallelism}", "^"+x.par+"$"))
		}
		task := fmt.Sprintf("In namespace %s, create a Job named '%s' (busybox:1.36) that runs 'echo Hello'", x.ns, x.name)
		if x.comp != "1" {
			task += fmt.Sprintf(", with completions=%s and parallelism=%s", x.comp, x.par)
		}
		task += "."
		out = append(out, gq(
			fmt.Sprintf("qg-job-%02d", i+1), models.DomainApplicationDesign, diff,
			fmt.Sprintf("Batch Job %s", x.name),
			"Jobs run Pods to completion; completions/parallelism control batching.",
			task, solution, x.ns,
			genHints(
				"'kubectl create job' scaffolds the basics; edit for completions.",
				"Job pod templates require restartPolicy Never or OnFailure.",
			),
			checks,
		))
	}
	return out
}

func genCronJobs() []*models.Question {
	type v struct{ ns, name, schedule, extra string }
	variants := []v{
		{"ckad-gcron01", "tick-1m", "*/1 * * * *", ""},
		{"ckad-gcron02", "tick-2m", "*/2 * * * *", ""},
		{"ckad-gcron03", "tick-5m", "*/5 * * * *", ""},
		{"ckad-gcron04", "hourly", "0 * * * *", ""},
		{"ckad-gcron05", "daily", "0 0 * * *", ""},
		{"ckad-gcron06", "weekly", "0 0 * * 0", ""},
		{"ckad-gcron07", "no-overlap", "*/3 * * * *", "Forbid"},
		{"ckad-gcron08", "queued", "*/10 * * * *", "Forbid"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		checks := []models.Check{
			gcs("exists", fmt.Sprintf("CronJob %s exists", x.name), 2,
				"get cronjob "+x.name+" -n "+x.ns+" -o name", "cronjob.batch/"+x.name),
			gcs("schedule", fmt.Sprintf("Schedule is %s", x.schedule), 2,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.schedule}", x.schedule),
			gcs("image", "Runs busybox:1.36", 1,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.jobTemplate.spec.template.spec.containers[0].image}", "busybox:1.36"),
		}
		w := 5
		if x.extra != "" {
			checks = append(checks, gcs("concurrency", fmt.Sprintf("concurrencyPolicy=%s", x.extra), 1,
				"get cronjob "+x.name+" -n "+x.ns+" -o jsonpath={.spec.concurrencyPolicy}", x.extra))
			w = 6
		}
		solution := fmt.Sprintf("kubectl create cronjob %s --image=busybox:1.36 --schedule='%s' -n %s -- /bin/sh -c 'date'",
			x.name, x.schedule, x.ns)
		if x.extra != "" {
			solution += fmt.Sprintf("  # then set spec.concurrencyPolicy: %s", x.extra)
		}
		out = append(out, &models.Question{
			ID:          fmt.Sprintf("qg-cron-%02d", i+1),
			Domain:      models.DomainApplicationDesign,
			Difficulty:  models.DifficultyMedium,
			Title:       fmt.Sprintf("Schedule CronJob %s", x.name),
			Description: "CronJobs run Jobs on a repeating schedule.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace " + x.ns}},
			Task:        fmt.Sprintf("In namespace %s, create a CronJob named '%s' (busybox:1.36) with schedule '%s' running '/bin/sh -c date'%s.", x.ns, x.name, x.schedule, concurrencyHint(x.extra)),
			Hints:       genHints("'kubectl create cronjob --schedule' takes crontab syntax.", "Fields are: minute hour day month weekday."),
			Solution:    solution,
			Checks:      checks,
			Cleanup:     []string{"delete namespace " + x.ns + " --ignore-not-found"},
			Weight:      w,
		})
	}
	return out
}

func concurrencyHint(extra string) string {
	if extra == "" {
		return ""
	}
	return fmt.Sprintf(" with concurrencyPolicy %s", extra)
}
