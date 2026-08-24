<div align="center">

# ⎈ CKAD Simulator

**A free, open-source practice simulator for the Certified Kubernetes Application Developer (CKAD) exam**

[![Questions](https://img.shields.io/badge/questions-695-blue)](#question-bank)
[![Domains](https://img.shields.io/badge/domains-5-green)](#question-bank)
[![License: MIT](https://img.shields.io/badge/license-MIT-purple)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=white)](https://react.dev)

*Real tasks · Real cluster · Real kubectl — graded live with partial credit*

</div>

---

## ✨ What is this?

CKAD Simulator is a killer.sh-style exam environment that runs entirely against
a **live Kubernetes cluster**. Every task provisions its own namespace, is
solved with real `kubectl` commands or YAML manifests in an embedded terminal,
and is **graded by inspecting actual cluster state** — not by string matching
your answer.

```mermaid
graph LR
    U[Browser] --> T[xterm.js terminal]
    T --> B[Go backend]
    B -- "exec kubectl" --> K[Kubernetes API]
    B -- "weighted checks" --> G[Partial-credit scoring]
```

### Highlights

- 🖥️ **In-browser terminal** — xterm.js with tab completion, pipes, redirects,
  and emulated `vi` / `nano` editors (write YAML without leaving the page)
- 🎯 **Live-cluster grading** — every check runs real `kubectl` queries;
  partially-correct solutions earn partial points
- 🧪 **Self-contained tasks** — each question creates & cleans up its own namespace
- ⏱️ **Exam mode** — 2-hour timer, domain-weighted random selection,
  results hidden until you submit (just like the real thing)
- 📚 **695 questions** across all five CKAD domains and three difficulty levels
- 🔍 **Review mode** — after the exam, see exactly which checks failed and why,
  with reference solutions

## 🚀 Quickstart

Prerequisites: **Go 1.26+**, **Node 20+**, a Kubernetes cluster (`minikube start`)
and `kubectl` pointed at it.

```bash
# Terminal 1 — backend
cd backend && go run ./cmd/server        # http://localhost:8080

# Terminal 2 — frontend
cd frontend && npm install && npm run dev # http://localhost:5173
```

Open <http://localhost:5173>, hit **Start a 2-hour exam session**, and good luck! 🍀

## 📊 Question Bank

| Difficulty | Count |
|-----------:|------:|
| Easy       | 150   |
| Medium     | 249   |
| Hard       | 296   |
| **Total**  | **695** |

Coverage spans every CKAD domain — pods, deployments, jobs/cronjobs, config &
secrets, probes, resource management, RBAC/service accounts, quotas, services,
ingress, network policies, scheduling (affinity/taints), security contexts,
StatefulSets/DaemonSets, HPA, rollout strategies, the ambassador pattern and more.

## ☁️ Deployment

The [`deploy/`](deploy/) folder ships a complete cloud kit: multi-stage
Dockerfiles (backend bundles `kubectl`), nginx SPA proxy, kustomize manifests
with RBAC, and scripts. Verified end-to-end on minikube; guides included for
free tiers (Oracle Always-Free + k3s, OKE, Civo) and managed clouds
(GKE/AKS/EKS).

```bash
./deploy/build.sh
minikube image load ghcr.io/manaf/ckad-backend:local ghcr.io/manaf/ckad-frontend:local
./deploy/deploy.sh minikube
```

> ⚠️ **Security**: candidates get broad cluster access by design. Put
> authentication in front of any public deployment (see deploy/README.md).

## 🏗️ Architecture

| Piece | Tech | Notes |
|---|---|---|
| Backend | Go + Gin | REST API, session/exam logic, exec-based checker |
| Grading engine | `kubectl` + JSONPath | Weighted checks, invert rules, regex/substring expectations |
| Frontend | React 19 + Vite + xterm.js | Exam UI, terminal, editor emulation, results review |
| State | In-memory | Swap for Redis/Postgres to scale horizontally |

## 🤝 Contributing

Issues and PRs are welcome! Good first contributions:

- New question families (follow the generator pattern in `backend/cmd/server/questions_gen*.go`)
- A persistent session store
- Auth options for public deployments
- More locales / UI polish

## 📄 License

[MIT](LICENSE) © abdulmanafc2001
