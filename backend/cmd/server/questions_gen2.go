package main

import (
	"fmt"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
)

// ------------------------------------------------------------- storage

func genPVCs() []*models.Question {
	type v struct{ ns, name, size, mode string }
	variants := []v{
		{"ckad-gpvc01", "data-a", "1Gi", "ReadWriteOnce"},
		{"ckad-gpvc02", "data-b", "2Gi", "ReadWriteOnce"},
		{"ckad-gpvc03", "shared-ro", "5Gi", "ReadOnlyMany"},
		{"ckad-gpvc04", "small-cache", "500Mi", "ReadWriteOnce"},
		{"ckad-gpvc05", "big-archive", "10Gi", "ReadWriteOnce"},
		{"ckad-gpvc06", "shared-rw", "3Gi", "ReadWriteMany"},
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
  resources:
    requests:
      storage: %s`, x.name, x.ns, x.mode, x.size)
		out = append(out, gq(
			fmt.Sprintf("qg-pvc-%02d", i+1), models.DomainApplicationDesign, models.DifficultyMedium,
			fmt.Sprintf("Claim %s of storage", x.size),
			"PVCs request durable storage from the cluster's provisioner.",
			fmt.Sprintf("In namespace %s, create a PersistentVolumeClaim named '%s' requesting %s with access mode %s.", x.ns, x.name, x.size, x.mode),
			solution, x.ns,
			genHints(
				"spec.resources.requests.storage carries the size.",
				"accessModes is a list; minikube's default StorageClass binds RWO claims automatically.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("PVC %s exists", x.name), 2,
					"get pvc "+x.name+" -n "+x.ns+" -o name", "persistentvolumeclaim/"+x.name),
				gcs("size", fmt.Sprintf("Requests %s", x.size), 2,
					"get pvc "+x.name+" -n "+x.ns+" -o jsonpath={.spec.resources.requests.storage}", x.size),
				gcs("mode", fmt.Sprintf("Access mode %s", x.mode), 2,
					"get pvc "+x.name+" -n "+x.ns+" -o jsonpath={.spec.accessModes[0]}", x.mode),
			},
		))
	}
	return out
}

func genPVCMounts() []*models.Question {
	type v struct{ ns, pvc, pod, mount string }
	variants := []v{
		{"ckad-gpm01", "store-a", "writer", "/data"},
		{"ckad-gpm02", "store-b", "uploader", "/uploads"},
		{"ckad-gpm03", "docs", "reader", "/srv/docs"},
		{"ckad-gpm04", "logs-vol", "archiver", "/var/archives"},
		{"ckad-gpm05", "data-vol", "backup", "/backup"},
		{"ckad-gpm06", "temp-vol", "processor", "/tmp/store"},
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
      storage: 1Gi`, x.pvc),
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
			fmt.Sprintf("qg-pvcmount-%02d", i+1), models.DomainApplicationDesign, models.DifficultyHard,
			fmt.Sprintf("Mount PVC %s into a Pod", x.pvc),
			"Pods consume PersistentVolumeClaims through volume mounts.",
			fmt.Sprintf("In namespace %s (a PVC '%s' already exists), create a Pod named '%s' (busybox:1.36) that mounts '%s' at %s via the claim.", x.ns, x.pvc, x.pod, x.pvc, x.mount),
			solution, x.ns, prepare,
			genHints(
				"Volumes of type persistentVolumeClaim reference the claim by name.",
				"mountPath is where the volume appears inside the container.",
			),
			[]models.Check{
				gcs("pod", fmt.Sprintf("Pod %s exists", x.pod), 1,
					"get pod "+x.pod+" -n "+x.ns+" -o name", "pod/"+x.pod),
				gcs("claim", fmt.Sprintf("References claim %s", x.pvc), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.volumes[0].persistentVolumeClaim.claimName}", x.pvc),
				gcs("mount", fmt.Sprintf("Mounted at %s", x.mount), 2,
					"get pod "+x.pod+" -n "+x.ns+" -o jsonpath={.spec.containers[0].volumeMounts[0].mountPath}", x.mount),
			},
		))
	}
	return out
}

// ---------------------------------------------------------- deployments

func genDeployments() []*models.Question {
	type v struct {
		ns, name, image string
		replicas        int
		port            string
	}
	variants := []v{
		{"ckad-gdep01", "web-a", "nginx:1.25", 2, "80"},
		{"ckad-gdep02", "web-b", "nginx:1.26", 3, "80"},
		{"ckad-gdep03", "api-a", "httpd:2.4", 2, "8080"},
		{"ckad-gdep04", "api-b", "gcr.io/app:v3", 4, "8080"},
		{"ckad-gdep05", "cache-a", "redis:7.2", 1, "6379"},
		{"ckad-gdep06", "queue-a", "rabbitmq:3.13", 2, "5672"},
		{"ckad-gdep07", "search-a", "elastic:8.13", 3, "9200"},
		{"ckad-gdep08", "ui-a", "dashboard:v5", 2, "3000"},
		{"ckad-gdep09", "auth-a", "auth-app:v1", 5, "8081"},
		{"ckad-gdep10", "log-a", "fluentd:1.16", 2, "24224"},
		{"ckad-gdep11", "db-proxy", "proxysql:2.5", 3, "6033"},
		{"ckad-gdep12", "mailer", "smtp-relay:1.4", 2, "25"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		diff := models.DifficultyEasy
		if x.replicas >= 4 {
			diff = models.DifficultyMedium
		}
		out = append(out, gq(
			fmt.Sprintf("qg-dep-%02d", i+1), models.DomainApplicationDeployment, diff,
			fmt.Sprintf("Deploy %s with %d replicas", x.name, x.replicas),
			"Deployments manage replicated, self-healing sets of Pods.",
			fmt.Sprintf("In namespace %s, create a Deployment named '%s' with %d replicas using image '%s'. The Pods must carry the label app=%s and expose container port %s.", x.ns, x.name, x.replicas, x.image, x.name, x.port),
			fmt.Sprintf("kubectl create deployment %s --image=%s -n %s --replicas=%d --port=%s\n# then ensure the pod template label app=%s (default with create deployment)",
				x.name, x.image, x.ns, x.replicas, x.port, x.name),
			x.ns,
			genHints(
				"'kubectl create deployment --replicas=N' sets the replica count.",
				"The selector/pod label defaults to app=<name>.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.name), 2,
					"get deploy "+x.name+" -n "+x.ns+" -o name", "deployment.apps/"+x.name),
				gcr("replicas", fmt.Sprintf("spec.replicas=%d", x.replicas), 2,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.replicas}", fmt.Sprintf("^%d$", x.replicas)),
				gcs("image", fmt.Sprintf("Uses image %s", x.image), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.image),
				gcs("selector", fmt.Sprintf("Selector matches app=%s", x.name), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.selector.matchLabels.app}", x.name),
			},
		))
	}
	return out
}

func genScaling() []*models.Question {
	type v struct {
		ns, name string
		from, to int
	}
	variants := []v{
		{"ckad-gscale01", "fe-app", 2, 5},
		{"ckad-gscale02", "be-app", 1, 3},
		{"ckad-gscale03", "worker", 3, 6},
		{"ckad-gscale04", "crawler", 2, 8},
		{"ckad-gscale05", "renderer", 4, 1},
		{"ckad-gscale06", "buffer", 5, 2},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create source deployment",
			CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s --replicas=%d", x.name, x.ns, x.from),
		}}
		dir := "Scale"
		if x.to < x.from {
			dir = "Scale down"
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-scale-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("%s %s to %d replicas", dir, x.name, x.to),
			"Replica counts can be changed live without recreating the Deployment.",
			fmt.Sprintf("In namespace %s, scale the existing Deployment '%s' (currently %d replicas) to %d replicas.", x.ns, x.name, x.from, x.to),
			fmt.Sprintf("kubectl scale deployment %s --replicas=%d -n %s", x.name, x.to, x.ns),
			x.ns, prepare,
			genHints(
				"'kubectl scale deployment NAME --replicas=N' does it in one line.",
				"You can also edit spec.replicas in the YAML.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.name), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o name", "deployment.apps/"+x.name),
				gcr("replicas", fmt.Sprintf("spec.replicas=%d", x.to), 3,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.replicas}", fmt.Sprintf("^%d$", x.to)),
			},
		))
	}
	return out
}

func genSetImage() []*models.Question {
	type v struct{ ns, name, oldImg, newImg, container string }
	variants := []v{
		{"ckad-gimg01", "frontend", "nginx:1.25", "nginx:1.26", "nginx"},
		{"ckad-gimg02", "backend", "httpd:2.4", "httpd:2.4.59", "httpd"},
		{"ckad-gimg03", "cache", "redis:7.0", "redis:7.2", "redis"},
		{"ckad-gimg04", "bus", "rabbitmq:3.12", "rabbitmq:3.13", "rabbitmq"},
		{"ckad-gimg05", "search", "elastic:8.11", "elastic:8.13", "elastic"},
		{"ckad-gimg06", "gateway", "envoy:v1.27", "envoy:v1.28", "envoy"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{{
			Name:        "create deployment with old image",
			CommandArgs: fmt.Sprintf("create deployment %s --image=%s -n %s", x.name, x.oldImg, x.ns),
		}}
		out = append(out, gqp(
			fmt.Sprintf("qg-setimg-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("Update %s image to %s", x.name, x.newImg),
			"'kubectl set image' updates a container image without editing YAML.",
			fmt.Sprintf("In namespace %s, update the Deployment '%s' so its container runs image '%s' instead of '%s'.", x.ns, x.name, x.newImg, x.oldImg),
			fmt.Sprintf("kubectl set image deployment/%s %s=%s -n %s", x.name, x.container, x.newImg, x.ns),
			x.ns, prepare,
			genHints(
				"kubectl set image deployment/NAME CONTAINER=IMAGE",
				"A new rollout starts automatically after the change.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.name), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o name", "deployment.apps/"+x.name),
				gcs("image", fmt.Sprintf("Container runs %s", x.newImg), 3,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.newImg),
			},
		))
	}
	return out
}

func genRollbacks() []*models.Question {
	type v struct{ ns, name, good, bad, container string }
	variants := []v{
		{"ckad-groll01", "payments", "redis:6.2", "redis:9.9-broken", "redis"},
		{"ckad-groll02", "catalog", "nginx:1.25", "nginx:99.9-missing", "nginx"},
		{"ckad-groll03", "identity", "httpd:2.4", "httpd:88.8-none", "httpd"},
		{"ckad-groll04", "notify", "rabbitmq:3.13", "rabbitmq:0.0-rc", "rabbitmq"},
		{"ckad-groll05", "gateway", "envoy:v1.28", "envoy:v99.0-broken", "envoy"},
		{"ckad-groll06", "analytics", "fluentd:1.16", "fluentd:0.0-fail", "fluentd"},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		prepare := []models.SetupStep{
			{
				Name:        "create healthy deployment",
				CommandArgs: fmt.Sprintf("create deployment %s --image=%s -n %s", x.name, x.good, x.ns),
			},
			{
				Name:        "break it with a bad image",
				CommandArgs: fmt.Sprintf("set image deployment/%s %s=%s -n %s", x.name, x.container, x.bad, x.ns),
			},
		}
		out = append(out, gqp(
			fmt.Sprintf("qg-rollback-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyMedium,
			fmt.Sprintf("Roll back %s", x.name),
			"Deployments keep revision history so bad rollouts can be undone.",
			fmt.Sprintf("The Deployment '%s' in namespace %s was updated to the non-existent image '%s' and its rollout is stuck. Roll it back to the previous revision.", x.name, x.ns, x.bad),
			fmt.Sprintf("kubectl rollout undo deployment/%s -n %s", x.name, x.ns),
			x.ns, prepare,
			genHints(
				"'kubectl rollout undo deployment/NAME' returns to the last revision.",
				"Check history with 'kubectl rollout history deployment/NAME'.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("Deployment %s exists", x.name), 1,
					"get deploy "+x.name+" -n "+x.ns+" -o name", "deployment.apps/"+x.name),
				gcs("image", fmt.Sprintf("Image restored to %s", x.good), 3,
					"get deploy "+x.name+" -n "+x.ns+" -o jsonpath={.spec.template.spec.containers[0].image}", x.good),
			},
		))
	}
	return out
}

func genLabelAnnotate() []*models.Question {
	type v struct {
		ns, kind, res, key, val string
		isAnnotation            bool
	}
	variants := []v{
		{"ckad-gmeta01", "pod", "legacy", "tier", "legacy", false},
		{"ckad-gmeta02", "deployment", "shop", "version", "v2", false},
		{"ckad-gmeta03", "service", "billing", "track", "stable", false},
		{"ckad-gmeta04", "pod", "janitor", "owner", "platform-team", true},
		{"ckad-gmeta05", "deployment", "search", "oncall", "green", true},
		{"ckad-gmeta06", "service", "geo", "contact", "netops", true},
		{"ckad-gmeta07", "pod", "canary", "weight", "10", false},
		{"ckad-gmeta08", "deployment", "sync", "slot", "blue", false},
	}
	out := make([]*models.Question, 0, len(variants))
	for i, x := range variants {
		resPath := x.kind
		if x.kind == "deployment" {
			resPath = "deploy"
		}
		if x.kind == "service" {
			resPath = "svc"
		}
		var prepare []models.SetupStep
		switch x.kind {
		case "pod":
			prepare = []models.SetupStep{{
				Name:        "create target pod",
				CommandArgs: fmt.Sprintf("run %s --image=busybox:1.36 -n %s --command -- sleep 3600", x.res, x.ns),
			}}
		case "deployment":
			prepare = []models.SetupStep{{
				Name:        "create target deployment",
				CommandArgs: fmt.Sprintf("create deployment %s --image=nginx:1.25 -n %s", x.res, x.ns),
			}}
		case "service":
			prepare = []models.SetupStep{
				{
					Name:        "create backing deployment",
					CommandArgs: fmt.Sprintf("create deployment %s-app --image=nginx:1.25 -n %s", x.res, x.ns),
				},
				{
					Name:        "expose it as a service",
					CommandArgs: fmt.Sprintf("expose deployment %s-app --port=80 --name=%s -n %s", x.res, x.res, x.ns),
				},
			}
		}

		what := "label"
		flag := "-l"
		field := "labels"
		if x.isAnnotation {
			what = "annotate"
			flag = "--annotation"
			field = "annotations"
		}

		out = append(out, gqp(
			fmt.Sprintf("qg-meta-%02d", i+1), models.DomainApplicationDeployment, models.DifficultyEasy,
			fmt.Sprintf("%s %s %s: %s=%s", what, x.kind, x.res, x.key, x.val),
			"Labels identify objects; annotations attach arbitrary metadata.",
			fmt.Sprintf("In namespace %s, add the %s %s=%s to the existing %s named '%s'.", x.ns, what, x.key, x.val, x.kind, x.res),
			fmt.Sprintf("kubectl %s %s %s %s %s=%s -n %s", what, resPath, x.res, flag, x.key, x.val, x.ns),
			x.ns, prepare,
			genHints(
				"'kubectl label' and 'kubectl annotate' modify metadata in place.",
				"Overwriting an existing key needs --overwrite.",
			),
			[]models.Check{
				gcs("exists", fmt.Sprintf("%s %s exists", x.kind, x.res), 1,
					"get "+resPath+" "+x.res+" -n "+x.ns+" -o name", resPath+"/"+x.res),
				gcs("meta", fmt.Sprintf("Has %s %s=%s", field, x.key, x.val), 3,
					fmt.Sprintf("get %s %s -n %s -o jsonpath={.metadata.%s.%s}", resPath, x.res, x.ns, field, x.key), x.val),
			},
		))
	}
	return out
}
