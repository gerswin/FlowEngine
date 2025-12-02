# FlowEngine - Deployment Guide

## Deployment Options

1. [Docker Compose (Local/Dev)](#docker-compose)
2. [Google Cloud Run](#google-cloud-run)
3. [Kubernetes](#kubernetes)
4. [Azure Container Apps](#azure-container-apps)

---

## Docker Compose

### Local Development

```bash
# Start full stack
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop
docker-compose down

# Clean volumes
docker-compose down -v
```

### Services

| Service | Port | Description |
|---------|------|-------------|
| api | 8080 | FlowEngine API |
| postgres | 5432 | PostgreSQL 16 |
| redis | 6379 | Redis 7 |

---

## Google Cloud Run

### Prerequisites

```bash
# Install gcloud CLI
# Authenticate
gcloud auth login
gcloud config set project YOUR_PROJECT_ID
```

### 1. Create Cloud SQL Instance

```bash
# Create PostgreSQL instance
gcloud sql instances create flowengine-db \
  --database-version=POSTGRES_16 \
  --tier=db-f1-micro \
  --region=us-central1 \
  --root-password=YOUR_ROOT_PASSWORD

# Create database
gcloud sql databases create flowengine --instance=flowengine-db

# Create user
gcloud sql users create flowuser \
  --instance=flowengine-db \
  --password=YOUR_DB_PASSWORD
```

### 2. Create Secrets

```bash
# Database password
echo -n "YOUR_DB_PASSWORD" | gcloud secrets create db-password --data-file=-

# JWT secret
openssl rand -base64 32 | gcloud secrets create jwt-secret --data-file=-

# Grant access to Cloud Run service account
gcloud secrets add-iam-policy-binding db-password \
  --member="serviceAccount:YOUR_PROJECT_NUMBER-compute@developer.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

gcloud secrets add-iam-policy-binding jwt-secret \
  --member="serviceAccount:YOUR_PROJECT_NUMBER-compute@developer.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```

### 3. Build and Push Image

```bash
# Build with Cloud Build
gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/flowengine:latest

# Or build locally and push
docker build -t gcr.io/YOUR_PROJECT_ID/flowengine:latest .
docker push gcr.io/YOUR_PROJECT_ID/flowengine:latest
```

### 4. Deploy to Cloud Run

```bash
gcloud run deploy flowengine-api \
  --image=gcr.io/YOUR_PROJECT_ID/flowengine:latest \
  --platform=managed \
  --region=us-central1 \
  --allow-unauthenticated \
  --add-cloudsql-instances=YOUR_PROJECT_ID:us-central1:flowengine-db \
  --set-env-vars="PORT=8080" \
  --set-env-vars="GIN_MODE=release" \
  --set-env-vars="LOG_LEVEL=info" \
  --set-env-vars="LOG_FORMAT=json" \
  --set-env-vars="POSTGRES_HOST=/cloudsql/YOUR_PROJECT_ID:us-central1:flowengine-db" \
  --set-env-vars="POSTGRES_PORT=5432" \
  --set-env-vars="POSTGRES_USER=flowuser" \
  --set-env-vars="POSTGRES_DB=flowengine" \
  --set-env-vars="POSTGRES_SSLMODE=disable" \
  --set-secrets="POSTGRES_PASSWORD=db-password:latest" \
  --set-secrets="JWT_SECRET=jwt-secret:latest" \
  --memory=512Mi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=10 \
  --timeout=300
```

### 5. (Optional) Add Redis via Memorystore

```bash
# Create Redis instance
gcloud redis instances create flowengine-cache \
  --size=1 \
  --region=us-central1 \
  --redis-version=redis_7_0

# Get Redis host
REDIS_HOST=$(gcloud redis instances describe flowengine-cache \
  --region=us-central1 --format="value(host)")

# Update Cloud Run with Redis
gcloud run services update flowengine-api \
  --region=us-central1 \
  --set-env-vars="REDIS_ADDR=${REDIS_HOST}:6379" \
  --vpc-connector=YOUR_VPC_CONNECTOR
```

### Cloud Run Service YAML

```yaml
# cloudrun-service.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: flowengine-api
  annotations:
    run.googleapis.com/ingress: all
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/cloudsql-instances: YOUR_PROJECT:us-central1:flowengine-db
        autoscaling.knative.dev/minScale: "0"
        autoscaling.knative.dev/maxScale: "10"
    spec:
      containerConcurrency: 80
      timeoutSeconds: 300
      containers:
        - image: gcr.io/YOUR_PROJECT/flowengine:latest
          ports:
            - containerPort: 8080
          resources:
            limits:
              memory: 512Mi
              cpu: "1"
          env:
            - name: PORT
              value: "8080"
            - name: GIN_MODE
              value: "release"
            - name: LOG_FORMAT
              value: "json"
            - name: POSTGRES_HOST
              value: "/cloudsql/YOUR_PROJECT:us-central1:flowengine-db"
            - name: POSTGRES_PORT
              value: "5432"
            - name: POSTGRES_USER
              value: "flowuser"
            - name: POSTGRES_DB
              value: "flowengine"
            - name: POSTGRES_SSLMODE
              value: "disable"
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-password
                  key: latest
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: jwt-secret
                  key: latest
```

Deploy:

```bash
gcloud run services replace cloudrun-service.yaml --region=us-central1
```

---

## Kubernetes

### Prerequisites

- Kubernetes cluster (GKE, EKS, AKS, or local)
- kubectl configured
- Helm (optional)

### 1. Create Namespace

```bash
kubectl create namespace flowengine
```

### 2. Create Secrets

```bash
kubectl create secret generic flowengine-secrets \
  --namespace=flowengine \
  --from-literal=POSTGRES_PASSWORD=YOUR_DB_PASSWORD \
  --from-literal=JWT_SECRET=YOUR_JWT_SECRET
```

### 3. Create ConfigMap

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: flowengine-config
  namespace: flowengine
data:
  PORT: "8080"
  GIN_MODE: "release"
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
  POSTGRES_HOST: "postgres-service"
  POSTGRES_PORT: "5432"
  POSTGRES_USER: "flowuser"
  POSTGRES_DB: "flowengine"
  POSTGRES_SSLMODE: "disable"
  REDIS_ADDR: "redis-service:6379"
```

### 4. Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: flowengine-api
  namespace: flowengine
spec:
  replicas: 3
  selector:
    matchLabels:
      app: flowengine-api
  template:
    metadata:
      labels:
        app: flowengine-api
    spec:
      containers:
        - name: api
          image: gcr.io/YOUR_PROJECT/flowengine:latest
          ports:
            - containerPort: 8080
          envFrom:
            - configMapRef:
                name: flowengine-config
          env:
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: flowengine-secrets
                  key: POSTGRES_PASSWORD
            - name: JWT_SECRET
              valueFrom:
                secretKeyRef:
                  name: flowengine-secrets
                  key: JWT_SECRET
          resources:
            requests:
              memory: "256Mi"
              cpu: "250m"
            limits:
              memory: "512Mi"
              cpu: "500m"
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 10
---
apiVersion: v1
kind: Service
metadata:
  name: flowengine-api
  namespace: flowengine
spec:
  selector:
    app: flowengine-api
  ports:
    - port: 80
      targetPort: 8080
  type: LoadBalancer
```

### 5. Apply

```bash
kubectl apply -f configmap.yaml
kubectl apply -f deployment.yaml
```

### 6. Ingress (Optional)

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: flowengine-ingress
  namespace: flowengine
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
spec:
  tls:
    - hosts:
        - api.flowengine.example.com
      secretName: flowengine-tls
  rules:
    - host: api.flowengine.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: flowengine-api
                port:
                  number: 80
```

---

## Azure Container Apps

### Prerequisites

```bash
# Install Azure CLI
az login
az account set --subscription YOUR_SUBSCRIPTION_ID
```

### 1. Create Resources

```bash
# Resource group
az group create --name flowengine-rg --location eastus

# Container Apps environment
az containerapp env create \
  --name flowengine-env \
  --resource-group flowengine-rg \
  --location eastus

# Azure Database for PostgreSQL
az postgres flexible-server create \
  --name flowengine-db \
  --resource-group flowengine-rg \
  --location eastus \
  --admin-user flowuser \
  --admin-password YOUR_PASSWORD \
  --sku-name Standard_B1ms \
  --version 16

# Create database
az postgres flexible-server db create \
  --resource-group flowengine-rg \
  --server-name flowengine-db \
  --database-name flowengine
```

### 2. Deploy Container App

```bash
az containerapp create \
  --name flowengine-api \
  --resource-group flowengine-rg \
  --environment flowengine-env \
  --image ghcr.io/lafabric-linktic/flowengine:latest \
  --target-port 8080 \
  --ingress external \
  --min-replicas 0 \
  --max-replicas 10 \
  --cpu 0.5 \
  --memory 1.0Gi \
  --env-vars \
    "PORT=8080" \
    "GIN_MODE=release" \
    "LOG_FORMAT=json" \
    "POSTGRES_HOST=flowengine-db.postgres.database.azure.com" \
    "POSTGRES_PORT=5432" \
    "POSTGRES_USER=flowuser" \
    "POSTGRES_DB=flowengine" \
    "POSTGRES_SSLMODE=require" \
  --secrets \
    "db-password=YOUR_DB_PASSWORD" \
    "jwt-secret=YOUR_JWT_SECRET" \
  --secret-env-vars \
    "POSTGRES_PASSWORD=db-password" \
    "JWT_SECRET=jwt-secret"
```

---

## Database Migrations

### Run Migrations

```bash
# Local
make migrate-up

# Docker
docker-compose exec api /app/migrate -path /app/migrations -database "postgres://..." up

# Kubernetes
kubectl exec -it deployment/flowengine-api -n flowengine -- /app/migrate up
```

### Initialize Schema

The init script (`scripts/init.sql`) is automatically run when PostgreSQL container starts.

For production, apply migrations manually:

```bash
# Connect to Cloud SQL
gcloud sql connect flowengine-db --user=flowuser

# Run init script
\i scripts/init.sql
```

---

## Monitoring

### Health Check Endpoint

```bash
curl https://your-api-url/health
```

### Logging

- **Local**: stdout/stderr
- **Cloud Run**: Cloud Logging
- **Kubernetes**: kubectl logs
- **Azure**: Container Apps logs

### Metrics

Configure your APM tool (Datadog, New Relic, etc.) with these environment variables:

```yaml
DD_AGENT_HOST: datadog-agent
DD_TRACE_ENABLED: "true"
DD_SERVICE: flowengine
```

---

## Security Checklist

- [ ] Use strong JWT_SECRET (32+ bytes)
- [ ] Enable HTTPS/TLS
- [ ] Use Secret Manager for credentials
- [ ] Configure CORS properly
- [ ] Enable database SSL
- [ ] Set up firewall rules
- [ ] Use private networking for database
- [ ] Enable audit logging
- [ ] Regular security updates
