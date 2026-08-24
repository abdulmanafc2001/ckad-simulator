package main

import "github.com/manaf/ckad-simulator/backend/internal/models"

// seedQuestions returns the full CKAD question bank: the hand-written
// originals plus the programmatically generated families (questions_gen.go).
func seedQuestions() []*models.Question {
	return append(manualQuestions(), generatedQuestions()...)
}

// manualQuestions returns the original hand-written CKAD question bank. Every
// task is self-contained: it prepares its own namespace, is solved against
// the live cluster (minikube), is graded by weighted kubectl checks
// (killer.sh style partial credit), and cleans up after itself.
func manualQuestions() []*models.Question {
	return []*models.Question{
		{
			ID:          "q-pod-nginx",
			Domain:      models.DomainApplicationDesign,
			Difficulty:  models.DifficultyEasy,
			Title:       "Run a single nginx Pod",
			Description: "Create the simplest possible workload: a single Pod running the nginx web server.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-basics"}},
			Task:        "Create a Pod named 'web' using the image 'nginx:1.25' in the 'ckad-basics' namespace.",
			Hints: []string{
				"You can generate a Pod imperatively with 'kubectl run'.",
				"Use --image to set the container image and -n to set the namespace.",
			},
			Solution: "kubectl run web --image=nginx:1.25 -n ckad-basics",
			Weight:   4,
			Checks: []models.Check{
				{ID: "pod-exists", Description: "Pod web exists in ckad-basics", Weight: 2,
					CommandArgs: "get pod web -n ckad-basics -o name", ExpectSubstring: "pod/web"},
				{ID: "pod-image", Description: "Pod uses image nginx:1.25", Weight: 2,
					CommandArgs: "get pod web -n ckad-basics -o jsonpath={.spec.containers[0].image}", ExpectSubstring: "nginx:1.25"},
			},
			Cleanup: []string{"delete namespace ckad-basics --ignore-not-found"},
		},
		{
			ID:          "q-multi-container",
			Domain:      models.DomainApplicationDesign,
			Difficulty:  models.DifficultyMedium,
			Title:       "Init container ordering",
			Description: "Init containers run to completion before app containers start. Use one to prepare shared state.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-init"}},
			Task:        "In namespace ckad-init, create a Pod named 'init-demo' with an initContainer named 'prep' (busybox) that writes 'hi' to /work/index.html on an emptyDir volume, and a main container 'web' (nginx) mounting the same volume at /usr/share/nginx/html.",
			Hints: []string{
				"initContainers is a list under spec, sibling to containers.",
				"Mount the same emptyDir volume into both the init and main containers.",
			},
			Solution: `apiVersion: v1
kind: Pod
metadata:
  name: init-demo
  namespace: ckad-init
spec:
  initContainers:
  - name: prep
    image: busybox:1.36
    command: ['sh','-c','echo hi > /work/index.html']
    volumeMounts:
    - {name: shared, mountPath: /work}
  containers:
  - name: web
    image: nginx:1.25
    volumeMounts:
    - {name: shared, mountPath: /usr/share/nginx/html}
  volumes:
  - name: shared
    emptyDir: {}`,
			Weight: 6,
			Checks: []models.Check{
				{ID: "init-exists", Description: "Init container prep exists", Weight: 2,
					CommandArgs: `get pod init-demo -n ckad-init -o jsonpath={.spec.initContainers[*].name}`, ExpectRegex: `(^| )prep( |$)`},
				{ID: "main-exists", Description: "Main container web exists", Weight: 1,
					CommandArgs: `get pod init-demo -n ckad-init -o jsonpath={.spec.containers[*].name}`, ExpectRegex: `(^| )web( |$)`},
				{ID: "shared-vol", Description: "Pod defines emptyDir volume shared", Weight: 2,
					CommandArgs: `get pod init-demo -n ckad-init -o jsonpath={.spec.volumes[*].name}`, ExpectRegex: `(^| )shared( |$)`},
				{ID: "main-mount", Description: "web mounts /usr/share/nginx/html", Weight: 1,
					CommandArgs: `get pod init-demo -n ckad-init -o jsonpath={.spec.containers[?(@.name=="web")].volumeMounts[0].mountPath}`, ExpectSubstring: "/usr/share/nginx/html"},
			},
			Cleanup: []string{"delete namespace ckad-init --ignore-not-found"},
		},
		{
			ID:          "q-job",
			Domain:      models.DomainApplicationDesign,
			Difficulty:  models.DifficultyMedium,
			Title:       "Create a Job",
			Description: "A Job creates one or more Pods and ensures a specified number complete successfully.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-job"}},
			Task:        "In namespace ckad-job, create a Job named 'pi' that runs image perl:5.34 computing pi to 2000 digits ('perl -Mbignum=bpi -wle \"print bpi(2000)\"').",
			Hints: []string{
				"Use 'kubectl create job' to scaffold it.",
				"A Job's restartPolicy must be Never or OnFailure.",
			},
			Solution: `kubectl create job pi --image=perl:5.34 -n ckad-job -- perl -Mbignum=bpi -wle 'print bpi(2000)'`,
			Weight:   6,
			Checks: []models.Check{
				{ID: "job-exists", Description: "Job pi exists", Weight: 2,
					CommandArgs: "get job pi -n ckad-job -o name", ExpectSubstring: "job.batch/pi"},
				{ID: "job-image", Description: "Job uses perl image", Weight: 2,
					CommandArgs: "get job pi -n ckad-job -o jsonpath={.spec.template.spec.containers[0].image}", ExpectSubstring: "perl"},
				{ID: "job-restart", Description: "restartPolicy is Never or OnFailure", Weight: 2,
					CommandArgs: "get job pi -n ckad-job -o jsonpath={.spec.template.spec.restartPolicy}", ExpectRegex: `Never|OnFailure`},
			},
			Cleanup: []string{"delete namespace ckad-job --ignore-not-found"},
		},
		{
			ID:          "q-cronjob",
			Domain:      models.DomainApplicationDesign,
			Difficulty:  models.DifficultyMedium,
			Title:       "Schedule a CronJob",
			Description: "A CronJob runs Jobs on a repeating schedule using cron syntax.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-cron"}},
			Task:        "In namespace ckad-cron, create a CronJob named 'hello' running every minute (*/1 * * * *) with busybox:1.36 executing '/bin/sh -c date; echo Hello'.",
			Hints: []string{
				"The schedule '*/1 * * * *' means every minute.",
				"Use 'kubectl create cronjob' with --schedule.",
			},
			Solution: `kubectl create cronjob hello --image=busybox:1.36 --schedule='*/1 * * * *' -n ckad-cron -- /bin/sh -c 'date; echo Hello'`,
			Weight:   6,
			Checks: []models.Check{
				{ID: "cron-exists", Description: "CronJob hello exists", Weight: 2,
					CommandArgs: "get cronjob hello -n ckad-cron -o name", ExpectSubstring: "cronjob.batch/hello"},
				{ID: "cron-schedule", Description: "Schedule is */1 * * * *", Weight: 2,
					CommandArgs: "get cronjob hello -n ckad-cron -o jsonpath={.spec.schedule}", ExpectSubstring: "*/1 * * * *"},
				{ID: "cron-image", Description: "Uses busybox image", Weight: 2,
					CommandArgs: "get cronjob hello -n ckad-cron -o jsonpath={.spec.jobTemplate.spec.template.spec.containers[0].image}", ExpectSubstring: "busybox"},
			},
			Cleanup: []string{"delete namespace ckad-cron --ignore-not-found"},
		},
		{
			ID:          "q-pv-pvc",
			Domain:      models.DomainApplicationDesign,
			Difficulty:  models.DifficultyMedium,
			Title:       "Persistent storage claim",
			Description: "Pods request durable storage through a PersistentVolumeClaim (PVC).",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-store"}},
			Task:        "In namespace ckad-store, create a PersistentVolumeClaim named 'data' requesting 1Gi of ReadWriteOnce storage.",
			Hints: []string{
				"The kind is PersistentVolumeClaim.",
				"accessModes is a list; storage goes under resources.requests.",
			},
			Solution: `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data
  namespace: ckad-store
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 1Gi`,
			Weight: 6,
			Checks: []models.Check{
				{ID: "pvc-exists", Description: "PVC data exists", Weight: 2,
					CommandArgs: "get pvc data -n ckad-store -o name", ExpectSubstring: "persistentvolumeclaim/data"},
				{ID: "pvc-size", Description: "Requests 1Gi storage", Weight: 2,
					CommandArgs: "get pvc data -n ckad-store -o jsonpath={.spec.resources.requests.storage}", ExpectSubstring: "1Gi"},
				{ID: "pvc-access", Description: "AccessMode ReadWriteOnce", Weight: 2,
					CommandArgs: "get pvc data -n ckad-store -o jsonpath={.spec.accessModes[0]}", ExpectSubstring: "ReadWriteOnce"},
			},
			Cleanup: []string{"delete namespace ckad-store --ignore-not-found"},
		},
		{
			ID:          "q-deploy-nginx",
			Domain:      models.DomainApplicationDeployment,
			Difficulty:  models.DifficultyEasy,
			Title:       "Create a Deployment",
			Description: "Deployments manage a replicated, self-healing set of Pods.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-deploy"}},
			Task:        "In namespace ckad-deploy, create a Deployment named 'web' with 3 replicas using image nginx:1.25.",
			Hints: []string{
				"Use 'kubectl create deployment' then scale, or pass --replicas.",
				"Verify with 'kubectl get deploy web'.",
			},
			Solution: `kubectl create deployment web --image=nginx:1.25 --replicas=3 -n ckad-deploy`,
			Weight:   4,
			Checks: []models.Check{
				{ID: "deploy-exists", Description: "Deployment web exists", Weight: 1,
					CommandArgs: "get deploy web -n ckad-deploy -o name", ExpectSubstring: "deployment.apps/web"},
				{ID: "deploy-replicas", Description: "Declares 3 replicas", Weight: 2,
					CommandArgs: "get deploy web -n ckad-deploy -o jsonpath={.spec.replicas}", ExpectSubstring: "3"},
				{ID: "deploy-image", Description: "Uses nginx:1.25", Weight: 1,
					CommandArgs: "get deploy web -n ckad-deploy -o jsonpath={.spec.template.spec.containers[0].image}", ExpectSubstring: "nginx:1.25"},
			},
			Cleanup: []string{"delete namespace ckad-deploy --ignore-not-found"},
		},
		{
			ID:          "q-deploy-rollback",
			Domain:      models.DomainApplicationDeployment,
			Difficulty:  models.DifficultyMedium,
			Title:       "Roll back a Deployment",
			Description: "Deployments keep revision history so you can undo bad rollouts.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-rollout"},
				{Name: "deploy v1", Namespace: "ckad-rollout", YAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 1
  selector: {matchLabels: {app: api}}
  template:
    metadata: {labels: {app: api}}
    spec:
      containers:
        - name: api
          image: nginx:1.24
`},
				{Name: "upgrade to broken v2", CommandArgs: "set image deployment/api api=nginx:9.9 -n ckad-rollout"},
			},
			Task: "The Deployment api in namespace ckad-rollout was upgraded to a non-existent image nginx:9.9 and its rollout is stuck. Roll it back to the previous revision.",
			Hints: []string{
				"Check history with: kubectl rollout history deployment/api -n ckad-rollout",
				"Undo with: kubectl rollout undo deployment/api -n ckad-rollout",
			},
			Solution: `kubectl rollout undo deployment/api -n ckad-rollout`,
			Weight:   6,
			Checks: []models.Check{
				{ID: "image-restored", Description: "Image rolled back to nginx:1.24", Weight: 4,
					CommandArgs: "get deploy api -n ckad-rollout -o jsonpath={.spec.template.spec.containers[0].image}", ExpectSubstring: "nginx:1.24"},
				{ID: "rollout-ok", Description: "Rollout is complete (available)", Weight: 2,
					CommandArgs: "get deploy api -n ckad-rollout -o jsonpath={.status.conditions[?(@.type==\"Available\")].status}", ExpectSubstring: "True"},
			},
			Cleanup: []string{"delete namespace ckad-rollout --ignore-not-found"},
		},
		{
			ID:          "q-labels",
			Domain:      models.DomainApplicationDeployment,
			Difficulty:  models.DifficultyEasy,
			Title:       "Label a resource",
			Description: "Labels are key/value pairs attached to objects for grouping and selection.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-labels"},
				{Name: "create pod", Namespace: "ckad-labels", YAML: `apiVersion: v1
kind: Pod
metadata:
  name: labeled
spec:
  containers:
    - name: main
      image: nginx:1.25
`},
			},
			Task: "In namespace ckad-labels, add the label tier=frontend to the existing Pod 'labeled'.",
			Hints: []string{
				"kubectl label pod <name> key=value adds or updates labels.",
				"Use --overwrite if the key already exists.",
			},
			Solution: `kubectl label pod labeled tier=frontend -n ckad-labels`,
			Weight:   4,
			Checks: []models.Check{
				{ID: "label-set", Description: "Pod has label tier=frontend", Weight: 4,
					CommandArgs: "get pod labeled -n ckad-labels -o jsonpath={.metadata.labels.tier}", ExpectSubstring: "frontend"},
			},
			Cleanup: []string{"delete namespace ckad-labels --ignore-not-found"},
		},
		{
			ID:          "q-probe",
			Domain:      models.DomainApplicationObservability,
			Difficulty:  models.DifficultyMedium,
			Title:       "Liveness and readiness probes",
			Description: "Probes let the kubelet and Services know whether a container is healthy.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-probes"},
				{Name: "deploy api", Namespace: "ckad-probes", YAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 1
  selector: {matchLabels: {app: api}}
  template:
    metadata: {labels: {app: api}}
    spec:
      containers:
        - name: api
          image: nginx:1.25
          ports: [{containerPort: 80}]
`},
			},
			Task: "In namespace ckad-probes, update Deployment api so its container has an HTTP livenessProbe and readinessProbe both hitting path /healthz on port 80.",
			Hints: []string{
				"kubectl edit deployment api -n ckad-probes and add livenessProbe/readinessProbe under the container.",
				"Both probes need httpGet: {path: /healthz, port: 80}.",
			},
			Solution: `kubectl patch deployment api -n ckad-probes --type=strategic -p '
spec:
  template:
    spec:
      containers:
        - name: api
          livenessProbe: {httpGet: {path: /healthz, port: 80}}
          readinessProbe: {httpGet: {path: /healthz, port: 80}}'`,
			Weight: 6,
			Checks: []models.Check{
				{ID: "liveness-path", Description: "Liveness probe hits /healthz", Weight: 2,
					CommandArgs: "get deploy api -n ckad-probes -o jsonpath={.spec.template.spec.containers[0].livenessProbe.httpGet.path}", ExpectSubstring: "/healthz"},
				{ID: "readiness-path", Description: "Readiness probe hits /healthz", Weight: 2,
					CommandArgs: "get deploy api -n ckad-probes -o jsonpath={.spec.template.spec.containers[0].readinessProbe.httpGet.path}", ExpectSubstring: "/healthz"},
				{ID: "probe-port", Description: "Probes target port 80", Weight: 2,
					CommandArgs: "get deploy api -n ckad-probes -o jsonpath={.spec.template.spec.containers[0].livenessProbe.httpGet.port}", ExpectSubstring: "80"},
			},
			Cleanup: []string{"delete namespace ckad-probes --ignore-not-found"},
		},
		{
			ID:          "q-annotations",
			Domain:      models.DomainApplicationDeployment,
			Difficulty:  models.DifficultyEasy,
			Title:       "Annotate a resource",
			Description: "Annotations attach arbitrary non-identifying metadata to objects.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-annot"},
				{Name: "create pod", Namespace: "ckad-annot", YAML: `apiVersion: v1
kind: Pod
metadata:
  name: noted
spec:
  containers:
    - name: main
      image: nginx:1.25
`},
			},
			Task: "In namespace ckad-annot, annotate the Pod 'noted' with owner=team-alpha.",
			Hints: []string{
				"kubectl annotate pod <name> key=value works like kubectl label.",
			},
			Solution: `kubectl annotate pod noted owner=team-alpha -n ckad-annot`,
			Weight:   4,
			Checks: []models.Check{
				{ID: "annotation-set", Description: "Pod annotated owner=team-alpha", Weight: 4,
					CommandArgs: "get pod noted -n ckad-annot -o jsonpath={.metadata.annotations.owner}", ExpectSubstring: "team-alpha"},
			},
			Cleanup: []string{"delete namespace ckad-annot --ignore-not-found"},
		},
		{
			ID:          "q-configmap",
			Domain:      models.DomainApplicationEnvironment,
			Difficulty:  models.DifficultyEasy,
			Title:       "Create a ConfigMap",
			Description: "ConfigMaps store non-confidential configuration as key/value pairs.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-cm"}},
			Task:        "In namespace ckad-cm, create a ConfigMap named app-config containing the entry MODE=production.",
			Hints: []string{
				"kubectl create configmap app-config --from-literal=MODE=production",
			},
			Solution: `kubectl create configmap app-config --from-literal=MODE=production -n ckad-cm`,
			Weight:   4,
			Checks: []models.Check{
				{ID: "cm-exists", Description: "ConfigMap app-config exists", Weight: 1,
					CommandArgs: "get configmap app-config -n ckad-cm -o name", ExpectSubstring: "configmap/app-config"},
				{ID: "cm-data", Description: "Has MODE=production", Weight: 3,
					CommandArgs: "get configmap app-config -n ckad-cm -o jsonpath={.data.MODE}", ExpectSubstring: "production"},
			},
			Cleanup: []string{"delete namespace ckad-cm --ignore-not-found"},
		},
		{
			ID:          "q-secret",
			Domain:      models.DomainApplicationEnvironment,
			Difficulty:  models.DifficultyEasy,
			Title:       "Create a Secret",
			Description: "Secrets store sensitive data such as passwords and tokens.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-secret"}},
			Task:        "In namespace ckad-secret, create a generic Secret named db-cred with keys USER=admin and PASS=s3cr3t.",
			Hints: []string{
				"kubectl create secret generic db-cred --from-literal=USER=admin --from-literal=PASS=s3cr3t",
			},
			Solution: `kubectl create secret generic db-cred --from-literal=USER=admin --from-literal=PASS=s3cr3t -n ckad-secret`,
			Weight:   4,
			Checks: []models.Check{
				{ID: "secret-exists", Description: "Secret db-cred exists", Weight: 1,
					CommandArgs: "get secret db-cred -n ckad-secret -o name", ExpectSubstring: "secret/db-cred"},
				{ID: "secret-user", Description: "Key USER present", Weight: 1,
					CommandArgs: "get secret db-cred -n ckad-secret -o jsonpath={.data.USER}", ExpectRegex: ".+"},
				{ID: "secret-pass", Description: "Key PASS present", Weight: 2,
					CommandArgs: "get secret db-cred -n ckad-secret -o jsonpath={.data.PASS}", ExpectRegex: ".+"},
			},
			Cleanup: []string{"delete namespace ckad-secret --ignore-not-found"},
		},
		{
			ID:          "q-env-var",
			Domain:      models.DomainApplicationEnvironment,
			Difficulty:  models.DifficultyMedium,
			Title:       "Inject an environment variable",
			Description: "Containers consume ConfigMap values through env or envFrom.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-env"},
				{Name: "create configmap", Namespace: "ckad-env", YAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  MODE: production
`},
			},
			Task: "In namespace ckad-env, create a Pod named env-consumer (busybox:1.36, command sleep 3600) whose environment variable APP_MODE comes from ConfigMap app-config, key MODE.",
			Hints: []string{
				"In the Pod spec use env[].valueFrom.configMapKeyRef {name: app-config, key: MODE}.",
			},
			Solution: `cat <<EOF | kubectl apply -n ckad-env -f -
apiVersion: v1
kind: Pod
metadata:
  name: env-consumer
spec:
  containers:
    - name: main
      image: busybox:1.36
      command: ["sleep", "3600"]
      env:
        - name: APP_MODE
          valueFrom:
            configMapKeyRef: {name: app-config, key: MODE}
EOF`,
			Weight: 6,
			Checks: []models.Check{
				{ID: "pod-exists", Description: "Pod env-consumer exists", Weight: 2,
					CommandArgs: "get pod env-consumer -n ckad-env -o name", ExpectSubstring: "pod/env-consumer"},
				{ID: "env-name", Description: "Env var APP_MODE defined", Weight: 2,
					CommandArgs: "get pod env-consumer -n ckad-env -o jsonpath={.spec.containers[0].env[0].name}", ExpectSubstring: "APP_MODE"},
				{ID: "env-ref", Description: "Value comes from configMapKeyRef key MODE", Weight: 2,
					CommandArgs: "get pod env-consumer -n ckad-env -o jsonpath={.spec.containers[0].env[0].valueFrom.configMapKeyRef.key}", ExpectSubstring: "MODE"},
			},
			Cleanup: []string{"delete namespace ckad-env --ignore-not-found"},
		},
		{
			ID:          "q-resources",
			Domain:      models.DomainApplicationObservability,
			Difficulty:  models.DifficultyMedium,
			Title:       "Set resource requests and limits",
			Description: "Requests guarantee resources; limits cap consumption.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-res"}},
			Task:        "In namespace ckad-res, create a Pod named resource-demo (nginx:1.25) with requests cpu=100m memory=128Mi and limits cpu=250m memory=256Mi.",
			Hints: []string{
				"Under the container add resources.requests and resources.limits.",
			},
			Solution: `cat <<EOF | kubectl apply -n ckad-res -f -
apiVersion: v1
kind: Pod
metadata:
  name: resource-demo
spec:
  containers:
    - name: main
      image: nginx:1.25
      resources:
        requests: {cpu: 100m, memory: 128Mi}
        limits: {cpu: 250m, memory: 256Mi}
EOF`,
			Weight: 4,
			Checks: []models.Check{
				{ID: "req-cpu", Description: "Requests cpu=100m", Weight: 1,
					CommandArgs: "get pod resource-demo -n ckad-res -o jsonpath={.spec.containers[0].resources.requests.cpu}", ExpectSubstring: "100m"},
				{ID: "req-mem", Description: "Requests memory=128Mi", Weight: 1,
					CommandArgs: "get pod resource-demo -n ckad-res -o jsonpath={.spec.containers[0].resources.requests.memory}", ExpectSubstring: "128Mi"},
				{ID: "lim-cpu", Description: "Limits cpu=250m", Weight: 1,
					CommandArgs: "get pod resource-demo -n ckad-res -o jsonpath={.spec.containers[0].resources.limits.cpu}", ExpectSubstring: "250m"},
				{ID: "lim-mem", Description: "Limits memory=256Mi", Weight: 1,
					CommandArgs: "get pod resource-demo -n ckad-res -o jsonpath={.spec.containers[0].resources.limits.memory}", ExpectSubstring: "256Mi"},
			},
			Cleanup: []string{"delete namespace ckad-res --ignore-not-found"},
		},
		{
			ID:          "q-sa",
			Domain:      models.DomainApplicationEnvironment,
			Difficulty:  models.DifficultyMedium,
			Title:       "Create a ServiceAccount",
			Description: "ServiceAccounts provide an identity for processes running in Pods.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-sa"}},
			Task:        "In namespace ckad-sa, create a ServiceAccount named pipeline and a Pod named sa-consumer (busybox:1.36, sleep 3600) that uses it.",
			Hints: []string{
				"kubectl create serviceaccount pipeline -n ckad-sa",
				"Set spec.serviceAccountName on the Pod.",
			},
			Solution: `kubectl create serviceaccount pipeline -n ckad-sa
cat <<EOF | kubectl apply -n ckad-sa -f -
apiVersion: v1
kind: Pod
metadata:
  name: sa-consumer
spec:
  serviceAccountName: pipeline
  containers:
    - name: main
      image: busybox:1.36
      command: ["sleep", "3600"]
EOF`,
			Weight: 6,
			Checks: []models.Check{
				{ID: "sa-exists", Description: "ServiceAccount pipeline exists", Weight: 2,
					CommandArgs: "get sa pipeline -n ckad-sa -o name", ExpectSubstring: "serviceaccount/pipeline"},
				{ID: "pod-exists", Description: "Pod sa-consumer exists", Weight: 2,
					CommandArgs: "get pod sa-consumer -n ckad-sa -o name", ExpectSubstring: "pod/sa-consumer"},
				{ID: "pod-sa", Description: "Pod uses serviceAccountName pipeline", Weight: 2,
					CommandArgs: "get pod sa-consumer -n ckad-sa -o jsonpath={.spec.serviceAccountName}", ExpectSubstring: "pipeline"},
			},
			Cleanup: []string{"delete namespace ckad-sa --ignore-not-found"},
		},
		{
			ID:          "q-resource-quota",
			Domain:      models.DomainApplicationEnvironment,
			Difficulty:  models.DifficultyHard,
			Title:       "Apply a ResourceQuota",
			Description: "ResourceQuotas constrain total resource consumption per namespace.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-quota"}},
			Task:        "In namespace ckad-quota, create a ResourceQuota named team-quota limiting the namespace to 2 Pods and requests.cpu=500m.",
			Hints: []string{
				"Under spec.hard set pods: '2' and requests.cpu: 500m.",
			},
			Solution: `cat <<EOF | kubectl apply -n ckad-quota -f -
apiVersion: v1
kind: ResourceQuota
metadata:
  name: team-quota
spec:
  hard:
    pods: "2"
    requests.cpu: 500m
EOF`,
			Weight: 6,
			Checks: []models.Check{
				{ID: "quota-exists", Description: "ResourceQuota team-quota exists", Weight: 2,
					CommandArgs: "get quota team-quota -n ckad-quota -o name", ExpectSubstring: "resourcequota/team-quota"},
				{ID: "quota-pods", Description: "Limits pods to 2", Weight: 2,
					CommandArgs: "get quota team-quota -n ckad-quota -o jsonpath={.spec.hard.pods}", ExpectSubstring: "2"},
				{ID: "quota-cpu", Description: "Limits requests.cpu to 500m", Weight: 2,
					CommandArgs: "get quota team-quota -n ckad-quota -o jsonpath={.spec.hard.requests\\.cpu}", ExpectSubstring: "500m"},
			},
			Cleanup: []string{"delete namespace ckad-quota --ignore-not-found"},
		},
		{
			ID:          "q-limitrange",
			Domain:      models.DomainApplicationEnvironment,
			Difficulty:  models.DifficultyHard,
			Title:       "Define a LimitRange",
			Description: "LimitRanges apply default requests/limits to containers in a namespace.",
			Prepare:     []models.SetupStep{{Name: "create namespace", CommandArgs: "create namespace ckad-lr"}},
			Task:        "In namespace ckad-lr, create a LimitRange named default-limits that gives containers a default CPU limit of 200m and a default memory limit of 256Mi.",
			Hints: []string{
				"Use type: Container with default: {cpu: 200m, memory: 256Mi}.",
			},
			Solution: `cat <<EOF | kubectl apply -n ckad-lr -f -
apiVersion: v1
kind: LimitRange
metadata:
  name: default-limits
spec:
  limits:
    - type: Container
      default:
        cpu: 200m
        memory: 256Mi
EOF`,
			Weight: 6,
			Checks: []models.Check{
				{ID: "lr-exists", Description: "LimitRange default-limits exists", Weight: 2,
					CommandArgs: "get limitrange default-limits -n ckad-lr -o name", ExpectSubstring: "limitrange/default-limits"},
				{ID: "lr-cpu", Description: "Default CPU limit 200m", Weight: 2,
					CommandArgs: "get limitrange default-limits -n ckad-lr -o jsonpath={.spec.limits[0].default.cpu}", ExpectSubstring: "200m"},
				{ID: "lr-mem", Description: "Default memory limit 256Mi", Weight: 2,
					CommandArgs: "get limitrange default-limits -n ckad-lr -o jsonpath={.spec.limits[0].default.memory}", ExpectSubstring: "256Mi"},
			},
			Cleanup: []string{"delete namespace ckad-lr --ignore-not-found"},
		},
		{
			ID:          "q-namespace",
			Domain:      models.DomainApplicationEnvironment,
			Difficulty:  models.DifficultyEasy,
			Title:       "Create a Namespace",
			Description: "Namespaces split cluster resources between multiple users or projects.",
			Prepare:     nil,
			Task:        "Create a Namespace named ckad-team-x.",
			Hints: []string{
				"kubectl create namespace ckad-team-x",
			},
			Solution: `kubectl create namespace ckad-team-x`,
			Weight:   4,
			Checks: []models.Check{
				{ID: "ns-exists", Description: "Namespace ckad-team-x exists", Weight: 4,
					CommandArgs: "get namespace ckad-team-x -o name", ExpectSubstring: "namespace/ckad-team-x"},
			},
			Cleanup: []string{"delete namespace ckad-team-x --ignore-not-found"},
		},
		{
			ID:          "q-svc-clusterip",
			Domain:      models.DomainServicesNetworking,
			Difficulty:  models.DifficultyMedium,
			Title:       "Expose a Deployment (ClusterIP)",
			Description: "ClusterIP Services give a stable virtual IP for a set of Pods.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-svc"},
				{Name: "deploy web", Namespace: "ckad-svc", YAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  labels: {app: web}
spec:
  replicas: 2
  selector: {matchLabels: {app: web}}
  template:
    metadata: {labels: {app: web}}
    spec:
      containers:
        - name: web
          image: nginx:1.25
          ports: [{containerPort: 80}]
`},
			},
			Task: "In namespace ckad-svc, expose Deployment web as a Service named web-svc of type ClusterIP on port 80 targeting container port 80.",
			Hints: []string{
				"kubectl expose deployment web --name=web-svc --port=80 --target-port=80 -n ckad-svc",
			},
			Solution: `kubectl expose deployment web --name=web-svc --port=80 --target-port=80 -n ckad-svc`,
			Weight:   4,
			Checks: []models.Check{
				{ID: "svc-exists", Description: "Service web-svc exists", Weight: 1,
					CommandArgs: "get svc web-svc -n ckad-svc -o name", ExpectSubstring: "service/web-svc"},
				{ID: "svc-type", Description: "Type is ClusterIP", Weight: 1,
					CommandArgs: "get svc web-svc -n ckad-svc -o jsonpath={.spec.type}", ExpectSubstring: "ClusterIP"},
				{ID: "svc-port", Description: "Port 80", Weight: 1,
					CommandArgs: "get svc web-svc -n ckad-svc -o jsonpath={.spec.ports[0].port}", ExpectSubstring: "80"},
				{ID: "svc-target", Description: "targetPort 80", Weight: 1,
					CommandArgs: "get svc web-svc -n ckad-svc -o jsonpath={.spec.ports[0].targetPort}", ExpectSubstring: "80"},
			},
			Cleanup: []string{"delete namespace ckad-svc --ignore-not-found"},
		},
		{
			ID:          "q-svc-nodeport",
			Domain:      models.DomainServicesNetworking,
			Difficulty:  models.DifficultyMedium,
			Title:       "Expose a Deployment (NodePort)",
			Description: "NodePort Services open a static port on every node for external traffic.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-np"},
				{Name: "deploy web", Namespace: "ckad-np", YAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
  labels: {app: web}
spec:
  replicas: 1
  selector: {matchLabels: {app: web}}
  template:
    metadata: {labels: {app: web}}
    spec:
      containers:
        - name: web
          image: nginx:1.25
          ports: [{containerPort: 80}]
`},
			},
			Task: "In namespace ckad-np, expose Deployment web as a Service named web-np of type NodePort on port 80 targeting container port 80.",
			Hints: []string{
				"kubectl expose deployment web --name=web-np --type=NodePort --port=80 --target-port=80 -n ckad-np",
			},
			Solution: `kubectl expose deployment web --name=web-np --type=NodePort --port=80 --target-port=80 -n ckad-np`,
			Weight:   6,
			Checks: []models.Check{
				{ID: "svc-exists", Description: "Service web-np exists", Weight: 2,
					CommandArgs: "get svc web-np -n ckad-np -o name", ExpectSubstring: "service/web-np"},
				{ID: "svc-type", Description: "Type is NodePort", Weight: 2,
					CommandArgs: "get svc web-np -n ckad-np -o jsonpath={.spec.type}", ExpectSubstring: "NodePort"},
				{ID: "svc-port", Description: "Port 80", Weight: 2,
					CommandArgs: "get svc web-np -n ckad-np -o jsonpath={.spec.ports[0].port}", ExpectSubstring: "80"},
			},
			Cleanup: []string{"delete namespace ckad-np --ignore-not-found"},
		},
		{
			ID:          "q-ingress",
			Domain:      models.DomainServicesNetworking,
			Difficulty:  models.DifficultyHard,
			Title:       "Create an Ingress",
			Description: "Ingresses route external HTTP(S) traffic to Services by host and path.",
			Prepare: []models.SetupStep{
				{Name: "create namespace", CommandArgs: "create namespace ckad-ing"},
				{Name: "deploy web + svc", Namespace: "ckad-ing", YAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  replicas: 1
  selector: {matchLabels: {app: web}}
  template:
    metadata: {labels: {app: web}}
    spec:
      containers:
        - name: web
          image: nginx:1.25
          ports: [{containerPort: 80}]
---
apiVersion: v1
kind: Service
metadata:
  name: web-svc
spec:
  selector: {app: web}
  ports: [{port: 80, targetPort: 80}]
`},
			},
			Task: "In namespace ckad-ing, create an Ingress named app-ingress routing host app.local (path /) to Service web-svc on port 80.",
			Hints: []string{
				"Use networking.k8s.io/v1 Ingress with rules[].host=app.local.",
				"Backend: service {name: web-svc, port: {number: 80}}.",
			},
			Solution: `cat <<EOF | kubectl apply -n ckad-ing -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  rules:
    - host: app.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: web-svc
                port: {number: 80}
EOF`,
			Weight: 8,
			Checks: []models.Check{
				{ID: "ing-exists", Description: "Ingress app-ingress exists", Weight: 2,
					CommandArgs: "get ingress app-ingress -n ckad-ing -o name", ExpectSubstring: "ingress.networking.k8s.io/app-ingress"},
				{ID: "ing-host", Description: "Host is app.local", Weight: 2,
					CommandArgs: "get ingress app-ingress -n ckad-ing -o jsonpath={.spec.rules[0].host}", ExpectSubstring: "app.local"},
				{ID: "ing-svc", Description: "Routes to web-svc", Weight: 2,
					CommandArgs: "get ingress app-ingress -n ckad-ing -o jsonpath={.spec.rules[0].http.paths[0].backend.service.name}", ExpectSubstring: "web-svc"},
				{ID: "ing-port", Description: "Backend port 80", Weight: 2,
					CommandArgs: "get ingress app-ingress -n ckad-ing -o jsonpath={.spec.rules[0].http.paths[0].backend.service.port.number}", ExpectSubstring: "80"},
			},
			Cleanup: []string{"delete namespace ckad-ing --ignore-not-found"},
		},
	}
}
