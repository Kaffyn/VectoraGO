# Deploying Vectora

Vectora can be deployed as a local daemon or a containerized cloud service.

## 1. Local Deployment (Windows/macOS/Linux)

1. Download the binary for your platform.
2. Run `vectora start` to launch the background service.
3. Use `vectora config` to set your API keys.

## 2. Docker Deployment

```bash
docker build -t vectora:latest .
docker run -d \
  -v vectora-data:/data \
  -e GEMINI_API_KEY=your_key \
  -p 8080:8080 \
  vectora:latest
```

## 3. Kubernetes Deployment

We provide standard Helm-style manifests in the `k8s/` directory.

### Quick Start with Kubectl:

```bash
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/secret.yaml
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml
```

### Autoscaling

Vectora is stateless (per request) but maintains state in BoltDB. For HA, we recommend using a shared PVC or the `TenantManager` with an external SQL/S3 provider (forthcoming).

## 4. Environment Variables

| Variable      | Description          | Default         |
| ------------- | -------------------- | --------------- |
| `PORT`        | Service port         | 8080            |
| `DATA_DIR`    | Storage location     | ~/.vectora/data |
| `JWT_SECRET`  | Secret for auth      | auto-generated  |
| `CGO_ENABLED` | Required for usearch | 1               |

---

_Status: Production Ready_
