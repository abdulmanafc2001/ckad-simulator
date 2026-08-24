package main

import (
	"fmt"
	"strings"

	"github.com/manaf/ckad-simulator/backend/internal/models"
)

// ----------------------------------------------------------- probes

func genProbes() []*models.Question {
	type v struct {
		ns, dep, probe, path string
		isLiveness           bool
	}
	variants := []v{
		{"ckad-gprobe01", "alive", "livenessProbe", "/healthz", true},
		{"ckad-gprobe02", "ready", "readinessProbe", "/ready", false},
		{"ckad-gprobe03", "check", "livenessProbe", "/status", true},
		{"ckad-gprobe04", "live", "readinessProbe", "/ping", false},
		{"ckad-gprobe05", "health", "livenessProbe", "/-/healthy", true},
		{"ckad-gprobe06", "avail", "readinessProbe", "/healthcheck", false},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		probeName := "Liveness"
		if !x.isLiveness {
			probeName = "Readiness"
		}
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.dep, x.ns),
		}}
		solution := fmt.Sprintf(`kubectl patch deployment %s -n %s --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/0/%s","object":null}]'`, x.dep, x.ns, x.probe)
		if x.isLiveness {
			solution = fmt.Sprintf(`kubectl patch deployment %s -n %s --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/0/livenessProbe","object":{"httpGet":{"path":"%s","port":80},"initialDelaySeconds":5,"periodSeconds":10}}]'`,
				x.dep, x.ns, x.path)
		} else {
			solution = fmt.Sprintf(`kubectl patch deployment %s -n %s --type='json' -p='[{"op":"add","path":"/spec/template/spec/containers/0/readinessProbe","object":{"httpGet":{"path":"%s","port":80},"initialDelaySeconds":5,"periodSeconds":10}}]'`,
				x.dep, x.ns, x.path)
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-probe-%02d", i+1), models.DomainApplicationObservability, models.DifficultyMedium,
			fmt.Sprintf("Add %s to %s", probeName, x.dep),
			"Probes let kubelet verify container health at the container port.",
			fmt.Sprintf("In namespace %s, add a %s httpGet probe (path %s, port 80) to the Deployment '%s'.", x.ns, probeName, x.path, x.dep),
			solution, x.ns, prepare,
			genHints(
				fmt.Sprintf("Use 'kubectl edit deploy' or 'kubectl patch deploy' to add the %s.", probeName),
				fmt.Sprintf("In YAML, the probe goes under spec.template.spec.containers[0].%s.", x.probe),
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("path", fmt.Sprintf("%s path is %s", probeName, x.path), 3,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0]."+x.probe+".httpGet.path}", x.path),
				gcs("port", "Probe port is 80", 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0]."+x.probe+".httpGet.port}", "80"),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- resources

func genResources() []*models.Question {
	type v struct {
		ns, dep, cpuReq, cpuLim, memReq, memLim string
	}
	variants := []v{
		{"ckad-gres01", "web-a", "100m", "200m", "64Mi", "128Mi"},
		{"ckad-gres02", "api-b", "250m", "500m", "128Mi", "256Mi"},
		{"ckad-gres03", "worker-c", "100m", "500m", "64Mi", "512Mi"},
		{"ckad-gres04", "cache-d", "50m", "100m", "32Mi", "64Mi"},
		{"ckad-gres05", "proxy-e", "200m", "1000m", "256Mi", "512Mi"},
		{"ckad-gres06", "mailer-f", "100m", "300m", "64Mi", "256Mi"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.dep, x.ns),
		}}
		solution := fmt.Sprintf(`kubectl set resources deployment %s -n %s --requests=cpu=%s,memory=%s --limits=cpu=%s,memory=%s`,
			x.dep, x.ns, x.cpuReq, x.memReq, x.cpuLim, x.memLim)
		out = append(out, gqp(
			fmt.Sprintf("qg-res-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyMedium,
			fmt.Sprintf("Set resources on %s", x.dep),
			"Resource requests and limits control scheduling and runtime throttling.",
			fmt.Sprintf("In namespace %s, set resource requests (cpu=%s, memory=%s) and limits (cpu=%s, memory=%s) on Deployment '%s'.",
				x.ns, x.cpuReq, x.memReq, x.cpuLim, x.memLim, x.dep),
			solution, x.ns, prepare,
			genHints(
				"'kubectl set resources deploy NAME --requests=... --limits=...'.",
				"Resources are per-container under spec.template.spec.containers[].resources.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.dep), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o name", "deployment.apps/"+x.dep),
				gcs("cpu-req", fmt.Sprintf("CPU request is %s", x.cpuReq), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].resources.requests.cpu}", x.cpuReq),
				gcs("cpu-lim", fmt.Sprintf("CPU limit is %s", x.cpuLim), 2,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].resources.limits.cpu}", x.cpuLim),
				gcs("mem-req", fmt.Sprintf("Memory request is %s", x.memReq), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].resources.requests.memory}", x.memReq),
				gcs("mem-lim", fmt.Sprintf("Memory limit is %s", x.memLim), 1,
					"get deploy "+x.dep+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].resources.limits.memory}", x.memLim),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- configmaps

func genConfigMaps() []*models.Question {
	type v struct {
		ns, name, key, val string
		fromLiteral        bool
	}
	variants := []v{
		{"ckad-gcm01", "app-config", "APP_MODE", "production", true},
		{"ckad-gcm02", "db-config", "DB_HOST", "postgres.default.svc", true},
		{"ckad-gcm03", "cache-config", "CACHE_TTL", "300", true},
		{"ckad-gcm04", "log-config", "LOG_LEVEL", "info", true},
		{"ckad-gcm05", "feature-flags", "ENABLE_BETA", "true", true},
		{"ckad-gcm06", "endpoint-config", "API_URL", "https://api.internal:8080", true},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if i%3 == 1 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-cm-%02d", i+1), models.DomainApplicationEnvironment, diff,
			fmt.Sprintf("Create ConfigMap %s", x.name),
			"ConfigMaps decouple configuration from container images.",
			fmt.Sprintf("In namespace %s, create a ConfigMap named '%s' with data key '%s' set to '%s' (--from-literal).", x.ns, x.name, x.key, x.val),
			fmt.Sprintf("kubectl create configmap %s --from-literal=%s=%s -n %s", x.name, x.key, x.val, x.ns),
			x.ns,
			genHints(
				"'kubectl create configmap NAME --from-literal=KEY=VALUE'.",
				"The result lives under .data in the ConfigMap YAML.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("ConfigMap %s exists", x.name), 2,
					"get configmap "+x.name+" -n "+x.ns+" -o name", "configmap/"+x.name),
				gcs("key", fmt.Sprintf("Data key %s present", x.key), 2,
					"get configmap "+x.name+" -n "+x.ns+" -o jsonpath={.data."+x.key+"}", x.val),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- configmap consumers

func genCMConsumers() []*models.Question {
	type v struct {
		ns, pod, cmName, cmKey, mountPath, envKey string
		useEnvVar                                 bool
	}
	variants := []v{
		{"ckad-gcmc01", "cfg-app", "app-config", "APP_MODE", "/etc/config", "APP_MODE", true},
		{"ckad-gcmc02", "cfg-web", "web-config", "LISTEN_ADDR", "/etc/web", "LISTEN_ADDR", true},
		{"ckad-gcmc03", "cfg-worker", "worker-config", "QUEUE_URL", "/opt/cfg", "", false},
		{"ckad-gcmc04", "cfg-proxy", "proxy-config", "UPSTREAM", "/etc/nginx/conf.d", "UPSTREAM", true},
		{"ckad-gcmc05", "cfg-svc", "service-config", "ENDPOINT", "/etc/svc", "", false},
		{"ckad-gcmc06", "cfg-mq", "mq-config", "BROKER_URL", "/etc/mq", "BROKER_URL", true},
		{"ckad-gcmc07", "cfg-auth", "auth-config", "ISSUER", "/etc/auth", "AUTH_ISSUER", true},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		var prepare []models.SetupStep
		if x.useEnvVar {
			prepare = []models.SetupStep{{
				Name:        "create configmap",
				CommandArgs: fmt.Sprintf("create configmap %s --from-literal=%s=test-val -n %s", x.cmName, x.cmKey, x.ns),
			}}
		} else {
			prepare = []models.SetupStep{{
				Name:        "create configmap",
				CommandArgs: fmt.Sprintf("create configmap %s --from-literal=%s=test-val -n %s", x.cmName, x.cmKey, x.ns),
			}}
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
    - {name: cfg, mountPath: %s}
  volumes:
  - name: cfg
    configMap:
      name: %s`, x.pod, x.ns, x.mountPath, x.cmName)

		checks := []models.Check{
			gcs("pod", fmt.Sprintf("Pod %s exists", x.pod), 1,
				"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
			gcs("cm-ref", fmt.Sprintf("Volume references ConfigMap %s", x.cmName), 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].configMap.name}", x.cmName),
			gcs("mount", fmt.Sprintf("Mounted at %s", x.mountPath), 2,
				"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].mountPath}", x.mountPath),
		}
		task := fmt.Sprintf("In namespace %s (ConfigMap '%s' already exists), create a Pod named '%s' (busybox:1.36) that mounts the ConfigMap '%s' key '%s' at %s as a file.",
			x.ns, x.cmName, x.pod, x.cmName, x.cmKey, x.mountPath)

		if x.useEnvVar {
			solution = fmt.Sprintf(`apiVersion: v1
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
      valueFrom:
        configMapKeyRef:
          name: %s
          key: %s`, x.pod, x.ns, x.envKey, x.cmName, x.cmKey)

			task = fmt.Sprintf("In namespace %s (ConfigMap '%s' already exists), create a Pod named '%s' (busybox:1.36) that exposes ConfigMap key '%s' as environment variable %s.",
				x.ns, x.cmName, x.pod, x.cmKey, x.envKey)

			checks = []models.Check{
				gcs("pod", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("env-name", fmt.Sprintf("Env var %s defined", x.envKey), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].name}", x.envKey),
				gcs("cm-ref", "References ConfigMap "+x.cmName, 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].valueFrom.configMapKeyRef.name}", x.cmName),
			}
		}

		out = append(out, gqp(
			fmt.Sprintf("qg-cmcon-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyMedium,
			fmt.Sprintf("Mount ConfigMap %s into %s", x.cmName, x.pod),
			"ConfigMaps can be mounted as files or exposed as environment variables.",
			task, solution, x.ns, prepare,
			genHints(
				"Use 'volumes' with configMap type and a volumeMount to expose as files.",
				"Alternatively, use env[].valueFrom.configMapKeyRef to inject as env vars.",
			),
			checks,
		))
	}
	return out
}

// ----------------------------------------------------------- secrets

func genSecrets() []*models.Question {
	type v struct {
		ns, name, key, val string
	}
	variants := []v{
		{"ckad-gsec01", "db-creds", "password", "s3cret!"},
		{"ckad-gsec02", "api-key", "key", "ak-xyz-12345"},
		{"ckad-gsec03", "tls-pair", "tls.crt", "LS0tLS1CRUdJTi"},
		{"ckad-gsec04", "reg-cred", ".dockerconfigjson", "{}"},
		{"ckad-gsec05", "ssh-key", "ssh-privatekey", "-----BEGIN"},
		{"ckad-gsec06", "app-secret", "token", "eyJhbGciOi"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if i >= 3 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-sec-%02d", i+1), models.DomainApplicationEnvironment, diff,
			fmt.Sprintf("Create Secret %s", x.name),
			"Secrets store sensitive configuration like passwords and tokens.",
			fmt.Sprintf("In namespace %s, create a Secret named '%s' with data key '%s' (--from-literal).", x.ns, x.name, x.key),
			fmt.Sprintf("kubectl create secret generic %s --from-literal=%s='%s' -n %s", x.name, x.key, x.val, x.ns),
			x.ns,
			genHints(
				"'kubectl create secret generic NAME --from-literal=KEY=VALUE'.",
				"Secrets are base64-encoded in YAML but kubectl handles encoding.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Secret %s exists", x.name), 2,
					"get secret "+x.name+" -n "+x.ns+" -o name", "secret/"+x.name),
				gcs("type", "Secret type is Opaque", 1,
					"get secret "+x.name+" -n "+x.ns+" -o jsonpath={.type}", "Opaque"),
				gcs("key", fmt.Sprintf("Data key %s present", x.key), 2,
					"get secret "+x.name+" -n "+x.ns+" -o jsonpath={.data."+x.key+"}", ""),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- secret consumers

func genSecretConsumers() []*models.Question {
	type v struct {
		ns, pod, secName, secKey, envKey string
	}
	variants := []v{
		{"ckad-gscon01", "db-app", "db-creds", "password", "DB_PASSWORD"},
		{"ckad-gscon02", "api-app", "api-key", "key", "API_KEY"},
		{"ckad-gscon03", "auth-app", "auth-secret", "token", "AUTH_TOKEN"},
		{"ckad-gscon04", "worker-app", "worker-creds", "password", "WORKER_PASS"},
		{"ckad-gscon05", "mail-app", "smtp-cred", "password", "SMTP_PASS"},
		{"ckad-gscon06", "pay-app", "payment-key", "secret", "PAYMENT_SECRET"},
		{"ckad-gscon07", "grpc-app", "grpc-token", "token", "GRPC_AUTH_TOKEN"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create secret",
			CommandArgs: fmt.Sprintf("create secret generic %s --from-literal=%s=dummy -n %s", x.secName, x.secKey, x.ns),
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
    env:
    - name: %s
      valueFrom:
        secretKeyRef:
          name: %s
          key: %s`, x.pod, x.ns, x.envKey, x.secName, x.secKey)
		out = append(out, gqp(
			fmt.Sprintf("qg-scon-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyMedium,
			fmt.Sprintf("Inject Secret %s into %s", x.secName, x.pod),
			"Secrets can be injected as environment variables via secretKeyRef.",
			fmt.Sprintf("In namespace %s (Secret '%s' already exists with key '%s'), create a Pod named '%s' (busybox:1.36) that injects secret key '%s' as env var %s.",
				x.ns, x.secName, x.secKey, x.pod, x.secKey, x.envKey),
			solution, x.ns, prepare,
			genHints(
				"Use env[].valueFrom.secretKeyRef to reference a specific secret key.",
				"Both name and key are required in secretKeyRef.",
			),
			[]models.Check{
				gcs("pod", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("env-name", fmt.Sprintf("Env var %s defined", x.envKey), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].name}", x.envKey),
				gcs("sec-ref", "References Secret "+x.secName, 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].env[0].valueFrom.secretKeyRef.name}", x.secName),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- service accounts

func genServiceAccounts() []*models.Question {
	type v struct {
		ns, saName, role string
	}
	variants := []v{
		{"ckad-gsa01", "deployer", "admin"},
		{"ckad-gsa02", "reader", "view"},
		{"ckad-gsa03", "migrator", "edit"},
		{"ckad-gsa04", "monitor", "view"},
		{"ckad-gsa05", "operator", "edit"},
		{"ckad-gsa06", "auditor", "view"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if x.role == "admin" {
			diff = models.DifficultyMedium
		}
		solution := fmt.Sprintf(`kubectl create serviceaccount %s -n %s`, x.saName, x.ns)
		out = append(out, gq(
			fmt.Sprintf("qg-sa-%02d", i+1), models.DomainApplicationEnvironment, diff,
			fmt.Sprintf("Create ServiceAccount %s", x.saName),
			"ServiceAccounts provide identity for Pods and control RBAC access.",
			fmt.Sprintf("In namespace %s, create a ServiceAccount named '%s'.", x.ns, x.saName),
			solution, x.ns,
			genHints(
				"'kubectl create serviceaccount NAME' creates a ServiceAccount.",
				"The SA is referenced by name in Pod spec.serviceAccountName.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("ServiceAccount %s exists", x.saName), 3,
					"get serviceaccount "+x.saName+" -n "+x.ns+" -o name", "serviceaccount/"+x.saName),
				gcs("sa-token", "SA has a token secret", 1,
					"get serviceaccount "+x.saName+" -n "+x.ns+" -o jsonpath={.secrets[0].name}", ""),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- quotas and limit ranges

func genQuotasAndLR() []*models.Question {
	type quotaVar struct {
		ns, qName, pods, cpu, mem string
	}
	type lrVar struct {
		ns, lrName, defCPU, defMem, reqCPU, reqMem, maxCPU, maxMem string
	}
	quotaVariants := []quotaVar{
		{"ckad-gq01", "dev-quota", "10", "4", "8Gi"},
		{"ckad-gq02", "staging-quota", "20", "8", "16Gi"},
		{"ckad-gq03", "prod-quota", "50", "16", "32Gi"},
	}
	lrVariants := []lrVar{
		{"ckad-gl01", "dev-limits", "200m", "256Mi", "100m", "128Mi", "1", "1Gi"},
		{"ckad-gl02", "staging-limits", "500m", "512Mi", "250m", "256Mi", "2", "2Gi"},
		{"ckad-gl03", "prod-limits", "1000m", "1Gi", "500m", "512Mi", "4", "4Gi"},
	}
	out := make([]*models.Question, 0, len(quotaVariants)+len(lrVariants))

	for i, x := range quotaVariants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: ResourceQuota
metadata:
  name: %s
  namespace: %s
spec:
  hard:
    pods: "%s"
    requests.cpu: "%s"
    requests.memory: "%s"`, x.qName, x.ns, x.pods, x.cpu, x.mem)
		out = append(out, gq(
			fmt.Sprintf("qg-quota-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyMedium,
			fmt.Sprintf("Create ResourceQuota %s", x.qName),
			"ResourceQuotas limit the total resources a namespace can consume.",
			fmt.Sprintf("In namespace %s, create a ResourceQuota named '%s' limiting pods=%s, requests.cpu=%s, requests.memory=%s.",
				x.ns, x.qName, x.pods, x.cpu, x.mem),
			solution, x.ns,
			genHints(
				"ResourceQuota spec.hard defines the limits.",
				"Each field under hard is a string value in quotes.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("ResourceQuota %s exists", x.qName), 2,
					"get resourcequota "+x.qName+" -n "+x.ns+" -o name", "resourcequota/"+x.qName),
				gcs("pods", fmt.Sprintf("Pods limit is %s", x.pods), 2,
					"get resourcequota "+x.qName+" -n "+x.ns+" -o jsonpath={.spec.hard.pods}", x.pods),
				gcs("cpu", fmt.Sprintf("CPU limit is %s", x.cpu), 1,
					"get resourcequota "+x.qName+" -n "+x.ns+" -o jsonpath={.spec.hard.requests\\.cpu}", x.cpu),
			},
		))
	}

	for i, x := range lrVariants {
		solution := fmt.Sprintf(`apiVersion: v1
kind: LimitRange
metadata:
  name: %s
  namespace: %s
spec:
  limits:
  - type: Container
    default:
      cpu: %s
      memory: %s
    defaultRequest:
      cpu: %s
      memory: %s
    max:
      cpu: %s
      memory: %s`, x.lrName, x.ns, x.defCPU, x.defMem, x.reqCPU, x.reqMem, x.maxCPU, x.maxMem)
		out = append(out, gq(
			fmt.Sprintf("qg-lr-%02d", i+1), models.DomainApplicationEnvironment, models.DifficultyHard,
			fmt.Sprintf("Create LimitRange %s", x.lrName),
			"LimitRanges set default and maximum resource limits per container in a namespace.",
			fmt.Sprintf("In namespace %s, create a LimitRange named '%s' with container defaults (cpu=%s, memory=%s), defaultRequest (cpu=%s, memory=%s), and max (cpu=%s, memory=%s).",
				x.ns, x.lrName, x.defCPU, x.defMem, x.reqCPU, x.reqMem, x.maxCPU, x.maxMem),
			solution, x.ns,
			genHints(
				"LimitRange applies per-container defaults and hard maximums.",
				"Type must be 'Container' (not Pod).",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("LimitRange %s exists", x.lrName), 2,
					"get limitrange "+x.lrName+" -n "+x.ns+" -o name", "limitrange/"+x.lrName),
				gcs("def-cpu", fmt.Sprintf("Default CPU is %s", x.defCPU), 2,
					"get limitrange "+x.lrName+" -n "+x.ns+" -o jsonpath={.spec.limits[0].default.cpu}", x.defCPU),
				gcs("max-cpu", fmt.Sprintf("Max CPU is %s", x.maxCPU), 1,
					"get limitrange "+x.lrName+" -n "+x.ns+" -o jsonpath={.spec.limits[0].max.cpu}", x.maxCPU),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- namespaces

func genNamespaces() []*models.Question {
	type v struct {
		ns, name, labelKey, labelVal string
	}
	variants := []v{
		{"ckad-gns01", "dev-team", "env", "dev"},
		{"ckad-gns02", "qa-team", "env", "qa"},
		{"ckad-gns03", "staging", "env", "staging"},
		{"ckad-gns04", "prod-us", "region", "us-east-1"},
		{"ckad-gns05", "prod-eu", "region", "eu-west-1"},
		{"ckad-gns06", "sandbox", "purpose", "testing"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if i%3 == 2 {
			diff = models.DifficultyMedium
		}
		solution := fmt.Sprintf("kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -\nkubectl label namespace %s %s=%s --overwrite",
			x.name, x.name, x.labelKey, x.labelVal)
		out = append(out, gq(
			fmt.Sprintf("qg-ns-%02d", i+1), models.DomainApplicationEnvironment, diff,
			fmt.Sprintf("Create namespace %s with labels", x.name),
			"Namespaces isolate cluster resources and can carry labels for organization.",
			fmt.Sprintf("Create a namespace named '%s' with the label %s=%s.", x.name, x.labelKey, x.labelVal),
			solution, x.name,
			genHints(
				"Use 'kubectl create namespace NAME'.",
				"Add labels with 'kubectl label namespace NAME KEY=VALUE'.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Namespace %s exists", x.name), 2,
					"get namespace "+x.name+" -o name", "namespace/"+x.name),
				gcs("label", fmt.Sprintf("Has label %s=%s", x.labelKey, x.labelVal), 2,
					"get namespace "+x.name+" -o jsonpath={.metadata.labels."+x.labelKey+"}", x.labelVal),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- services

func genServices() []*models.Question {
	type v struct {
		ns, dep, svcName, svcType string
		port, targetPort          string
	}
	variants := []v{
		{"ckad-gsvc01", "web-a", "web-svc", "ClusterIP", "80", "8080"},
		{"ckad-gsvc02", "api-b", "api-svc", "ClusterIP", "8080", "8080"},
		{"ckad-gsvc03", "cache-c", "cache-svc", "ClusterIP", "6379", "6379"},
		{"ckad-gsvc04", "db-d", "db-svc", "ClusterIP", "5432", "5432"},
		{"ckad-gsvc05", "msg-e", "msg-svc", "NodePort", "5672", "5672"},
		{"ckad-gsvc06", "web-f", "web-np", "NodePort", "80", "80"},
		{"ckad-gsvc07", "login-g", "login-svc", "ClusterIP", "443", "8443"},
		{"ckad-gsvc08", "queue-h", "queue-svc", "ClusterIP", "15672", "15672"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.dep, x.ns),
		}}

		var solution string
		var checks []models.Check
		if x.svcType == "NodePort" {
			solution = fmt.Sprintf("kubectl expose deployment %s --type=NodePort --port=%s --target-port=%s --name=%s -n %s",
				x.dep, x.port, x.targetPort, x.svcName, x.ns)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svcName), 2,
					"get svc "+x.svcName+" -n "+x.ns+" -o name", "service/"+x.svcName),
				gcs("type", "Service type is NodePort", 2,
					"get svc "+x.svcName+" -n "+x.ns+" -o jsonpath={.spec.type}", "NodePort"),
				gcs("port", fmt.Sprintf("Service port is %s", x.port), 1,
					"get svc "+x.svcName+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", x.port),
			}
		} else {
			solution = fmt.Sprintf("kubectl expose deployment %s --port=%s --target-port=%s --name=%s -n %s",
				x.dep, x.port, x.targetPort, x.svcName, x.ns)
			checks = []models.Check{
				gcs("exists", fmt.Sprintf("Service %s exists", x.svcName), 2,
					"get svc "+x.svcName+" -n "+x.ns+" -o name", "service/"+x.svcName),
				gcs("type", "Service type is ClusterIP", 1,
					"get svc "+x.svcName+" -n "+x.ns+" -o jsonpath={.spec.type}", "ClusterIP"),
				gcs("port", fmt.Sprintf("Service port is %s", x.port), 1,
					"get svc "+x.svcName+" -n "+x.ns+" -o jsonpath={.spec.ports[0].port}", x.port),
				gcs("target", fmt.Sprintf("Target port is %s", x.targetPort), 1,
					"get svc "+x.svcName+" -n "+x.ns+" -o jsonpath={.spec.ports[0].targetPort}", x.targetPort),
			}
		}

		out = append(out, gqp(
			fmt.Sprintf("qg-svc-%02d", i+1), models.DomainServicesNetworking, models.DifficultyEasy,
			fmt.Sprintf("Expose %s as %s Service", x.dep, x.svcType),
			"Services provide stable network endpoints for a set of Pods.",
			fmt.Sprintf("In namespace %s, expose Deployment '%s' as a %s Service named '%s' on port %s (target-port %s).",
				x.ns, x.dep, x.svcType, x.svcName, x.port, x.targetPort),
			solution, x.ns, prepare,
			genHints(
				"'kubectl expose deployment NAME --port=P --target-port=T'.",
				"Add --type=NodePort for NodePort services.",
			),
			checks,
		))
	}
	return out
}

// ----------------------------------------------------------- ingresses

func genIngresses() []*models.Question {
	type v struct {
		ns, svcName, host, path, pathType string
	}
	variants := []v{
		{"ckad-ging01", "web-svc", "shop.example.com", "/app", "Prefix"},
		{"ckad-ging02", "api-svc", "api.example.com", "/v1", "Prefix"},
		{"ckad-ging03", "auth-svc", "auth.example.com", "/login", "Exact"},
		{"ckad-ging04", "docs-svc", "docs.example.com", "/guide", "Prefix"},
		{"ckad-ging05", "admin-svc", "admin.example.com", "/panel", "Prefix"},
		{"ckad-ging06", "status-svc", "status.example.com", "/health", "Exact"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create service",
			CommandArgs: fmt.Sprintf("create deployment app --image=nginx:1.25 -n %s && expose deployment app --port=80 --name=%s -n %s", x.ns, x.svcName, x.ns),
		}}
		solution := fmt.Sprintf(`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: %s-ing
  namespace: %s
spec:
  rules:
  - host: %s
    http:
      paths:
      - path: %s
        pathType: %s
        backend:
          service:
            name: %s
            port:
              number: 80`, strings.TrimSuffix(x.svcName, "-svc"), x.ns, x.host, x.path, x.pathType, x.svcName)
		out = append(out, gqp(
			fmt.Sprintf("qg-ing-%02d", i+1), models.DomainServicesNetworking, models.DifficultyMedium,
			fmt.Sprintf("Create Ingress for %s", x.host),
			"Ingress resources expose HTTP routes from outside the cluster to Services.",
			fmt.Sprintf("In namespace %s (Service '%s' already exists), create an Ingress named '%s-ing' routing host %s path %s (%s) to Service '%s' port 80.",
				x.ns, x.svcName, strings.TrimSuffix(x.svcName, "-svc"), x.host, x.path, x.pathType, x.svcName),
			solution, x.ns, prepare,
			genHints(
				"apiVersion is networking.k8s.io/v1.",
				"Each rule has a host and an http.paths list.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Ingress %s-ing exists", strings.TrimSuffix(x.svcName, "-svc")), 2,
					"get ingress "+strings.TrimSuffix(x.svcName, "-svc")+"-ing -n "+x.ns+" -o name", "ingress.networking.k8s.io/"+strings.TrimSuffix(x.svcName, "-svc")+"-ing"),
				gcs("host", fmt.Sprintf("Host is %s", x.host), 2,
					"get ingress "+strings.TrimSuffix(x.svcName, "-svc")+"-ing -n "+x.ns+" -o jsonpath={.spec.rules[0].host}", x.host),
				gcs("path", fmt.Sprintf("Path is %s", x.path), 2,
					"get ingress "+strings.TrimSuffix(x.svcName, "-svc")+"-ing -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[0].path}", x.path),
				gcs("svc", fmt.Sprintf("Backend service is %s", x.svcName), 1,
					"get ingress "+strings.TrimSuffix(x.svcName, "-svc")+"-ing -n "+x.ns+" -o jsonpath={.spec.rules[0].http.paths[0].backend.service.name}", x.svcName),
			},
		))
	}
	return out
}

// ----------------------------------------------------------- network policies

func genNetworkPolicies() []*models.Question {
	type v struct {
		ns, npName, podKey, podVal string
		policyTypes                []string
		ingressAllow               bool
		egressAllow                bool
	}
	variants := []v{
		{"ckad-gnp01", "deny-all-ingress", "app", "web", []string{"Ingress"}, true, false},
		{"ckad-gnp02", "allow-frontend", "tier", "frontend", []string{"Ingress"}, true, false},
		{"ckad-gnp03", "deny-all-egress", "role", "internal", []string{"Egress"}, false, true},
		{"ckad-gnp04", "allow-dns", "dns", "enabled", []string{"Egress"}, false, true},
		{"ckad-gnp05", "restrict-db", "tier", "database", []string{"Ingress"}, true, false},
		{"ckad-gnp06", "isolate-api", "app", "api", []string{"Ingress", "Egress"}, true, true},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyMedium
		if len(x.policyTypes) == 2 {
			diff = models.DifficultyHard
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
  policyTypes: %s`, x.npName, x.ns, x.podKey, x.podVal, formatStrSlice(x.policyTypes))
		if x.ingressAllow {
			solution += "\n  ingress: []"
		}
		if x.egressAllow {
			solution += "\n  egress: []"
		}

		task := fmt.Sprintf("In namespace %s, create a NetworkPolicy named '%s' that selects Pods with label %s=%s and specifies policyTypes %s.",
			x.ns, x.npName, x.podKey, x.podVal, formatStrSlice(x.policyTypes))

		out = append(out, gq(
			fmt.Sprintf("qg-np-%02d", i+1), models.DomainServicesNetworking, diff,
			fmt.Sprintf("Create NetworkPolicy %s", x.npName),
			"NetworkPolicies control traffic flow between Pods at the IP/port level.",
			task, solution, x.ns,
			genHints(
				"apiVersion is networking.k8s.io/v1.",
				"policyTypes lists Ingress, Egress, or both.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("NetworkPolicy %s exists", x.npName), 2,
					"get networkpolicy "+x.npName+" -n "+x.ns+" -o name", "networkpolicy.networking.k8s.io/"+x.npName),
				gcs("selector", fmt.Sprintf("Selects label %s=%s", x.podKey, x.podVal), 2,
					"get networkpolicy "+x.npName+" -n "+x.ns+" -o jsonpath={.spec.podSelector.matchLabels."+x.podKey+"}", x.podVal),
				gcs("policy-type", fmt.Sprintf("Includes %s", x.policyTypes[0]), 2,
					"get networkpolicy "+x.npName+" -n "+x.ns+" -o jsonpath={.spec.policyTypes}", x.policyTypes[0]),
			},
		))
	}
	return out
}

func formatStrSlice(ss []string) string {
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
