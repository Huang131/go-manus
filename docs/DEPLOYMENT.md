# 部署指南

## 1. 部署方式概览

| 部署方式 | 适用场景 | 推荐程度 |
|---------|---------|---------|
| Docker Compose (本地开发) | 本地开发、演示 | ⭐⭐⭐⭐⭐ |
| Docker Compose (生产) | 小规模生产部署 | ⭐⭐⭐⭐ |
| Kubernetes | 大规模生产部署 | ⭐⭐⭐⭐⭐ |
| 单机直接部署 | 最小化部署 | ⭐⭐⭐ |

## 2. 本地开发部署

### 2.1 前置要求

- Docker >= 20.10
- Docker Compose >= 2.0
- 至少 4GB 内存

### 2.2 快速启动

```bash
# 1. 进入项目目录
cd go-manus

# 2. 复制环境变量模板
cp api/.env.example .env

# 3. 配置环境变量（编辑 .env）
vim .env

# 4. 启动服务
docker-compose -f docker-compose.dev.yml up -d --build

# 5. 查看服务状态
docker-compose -f docker-compose.dev.yml ps
```

### 2.3 访问服务

- 前端 UI：http://localhost:3000
- API 服务：http://localhost:8080
- 健康检查：http://localhost:8080/api/v1/status

### 2.4 常用命令

```bash
# 查看日志
docker-compose -f docker-compose.dev.yml logs -f

# 查看特定服务日志
docker-compose -f docker-compose.dev.yml logs -f api

# 重启服务
docker-compose -f docker-compose.dev.yml restart

# 停止服务
docker-compose -f docker-compose.dev.yml down

# 清理所有数据（慎用）
docker-compose -f docker-compose.dev.yml down -v
```

## 3. 生产环境部署 (Docker)

### 3.1 环境要求

- Docker >= 24.0
- Docker Compose >= 2.20
- 至少 8GB 内存
- 50GB 磁盘空间

### 3.2 配置生产环境

创建 `docker-compose.prod.yml`：

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    restart: always
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    networks:
      - go-manus-prod

  redis:
    image: redis:7-alpine
    restart: always
    command: redis-server --appendonly yes --maxmemory 512mb
    volumes:
      - redis_data:/data
    networks:
      - go-manus-prod

  api:
    image: go-manus-api:latest
    restart: always
    environment:
      - DATABASE_HOST=postgres
      - REDIS_HOST=redis
      - ENV=production
      - LOG_LEVEL=info
    networks:
      - go-manus-prod
    deploy:
      resources:
        limits:
          memory: 2G

  ui:
    image: go-manus-ui:latest
    restart: always
    networks:
      - go-manus-prod

  sandbox:
    image: go-manus-sandbox:latest
    restart: always
    privileged: true
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    networks:
      - go-manus-prod

networks:
  go-manus-prod:
    driver: bridge

volumes:
  postgres_data:
  redis_data:
```

### 3.3 启动生产环境

```bash
# 构建并启动
docker-compose -f docker-compose.prod.yml up -d --build

# 配置健康检查
docker-compose -f docker-compose.prod.yml ps
```

## 4. Kubernetes 部署

### 4.1 环境要求

- Kubernetes >= 1.28
- Helm >= 3.12
- Ingress Controller (Traefik 或 Nginx)

### 4.2 目录结构

```
k8s/
├── namespace.yaml
├── postgres/
│   ├── deployment.yaml
│   └── service.yaml
├── redis/
│   ├── deployment.yaml
│   └── service.yaml
├── api/
│   ├── deployment.yaml
│   ├── service.yaml
│   └── hpa.yaml
├── ui/
│   ├── deployment.yaml
│   └── service.yaml
├── sandbox/
│   ├── deployment.yaml
│   └── service.yaml
└── ingress.yaml
```

### 4.3 部署步骤

```bash
# 1. 创建命名空间
kubectl apply -f k8s/namespace.yaml

# 2. 部署基础设施（PostgreSQL、Redis）
kubectl apply -f k8s/postgres/
kubectl apply -f k8s/redis/

# 3. 等待基础设施就绪
kubectl wait --for=condition=available deployment/postgres --timeout=300s
kubectl wait --for=condition=available deployment/redis --timeout=300s

# 4. 部署应用
kubectl apply -f k8s/api/
kubectl apply -f k8s/ui/
kubectl apply -f k8s/sandbox/

# 5. 配置 Ingress
kubectl apply -f k8s/ingress.yaml

# 6. 检查部署状态
kubectl get all -n go-manus
kubectl get ingress -n go-manus
```

### 4.4 关键配置

#### API Deployment

```yaml
# k8s/api/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: go-manus-api
  namespace: go-manus
spec:
  replicas: 2
  selector:
    matchLabels:
      app: api
  template:
    spec:
      containers:
      - name: api
        image: go-manus-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_HOST
          value: "postgres"
        - name: REDIS_HOST
          value: "redis"
        - name: ENV
          value: "production"
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

#### Ingress 配置

```yaml
# k8s/ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: go-manus
  namespace: go-manus
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "86400"
spec:
  ingressClassName: nginx
  rules:
  - host: api.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: ui
            port:
              number: 3000
      - path: /api/
        pathType: Prefix
        backend:
          service:
            name: api
            port:
              number: 8080
      - path: /sandbox/
        pathType: Prefix
        backend:
          service:
            name: sandbox
            port:
              number: 8080
  tls:
  - hosts:
    - api.example.com
    secretName: go-manus-tls
```

## 5. 环境变量配置

### 5.1 必需配置

```bash
# 数据库
DATABASE_HOST=postgres
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=your_secure_password
DATABASE_DATABASE=manus

# Redis
REDIS_HOST=redis
REDIS_PORT=6379

# LLM 配置（必需）
LLM_BASE_URL=https://api.openai.com
LLM_API_KEY=sk-your-api-key
LLM_MODEL_NAME=gpt-4

# COS 配置（文件存储）
COS_SECRET_ID=your_cos_secret_id
COS_SECRET_KEY=your_cos_secret_key
COS_REGION=ap-guangzhou
COS_BUCKET=your_bucket_name
```

### 5.2 可选配置

```bash
# 服务端口（默认）
SERVER_PORT=8080

# 日志级别（默认 info）
LOG_LEVEL=debug

# 环境（默认 development）
ENV=production

# 沙箱服务地址（默认从环境获取）
SANDBOX_ADDRESS=http://sandbox:8080
```

## 6. 数据持久化

### 6.1 PostgreSQL 数据

```bash
# 本地 Docker
-v postgres_data:/var/lib/postgresql/data

# K8s
# 使用 PVC
spec:
  volumes:
  - name: postgres-data
    persistentVolumeClaim:
      claimName: postgres-pvc
```

### 6.2 Redis 数据

```bash
# 本地 Docker
-v redis_data:/data

# K8s
# 使用 PVC 或空目录（Redis 可丢失）
```

### 6.3 备份策略

```bash
# PostgreSQL 备份
pg_dump -h postgres -U postgres -d manus > backup_$(date +%Y%m%d).sql

# 恢复
psql -h postgres -U postgres -d manus < backup_20240101.sql
```

## 7. 监控与日志

### 7.1 日志收集

```yaml
# K8s 配置日志收集
spec:
  containers:
  - name: api
    volumeMounts:
    - name: varlog
      mountPath: /var/log
```

### 7.2 健康检查

```yaml
livenessProbe:
  httpGet:
    path: /api/v1/status
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /api/v1/status
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

### 7.3 资源限制

建议配置：
- API：2CPU, 2GB 内存
- UI：1CPU, 512MB 内存
- Sandbox：2CPU, 2GB 内存
- PostgreSQL：2CPU, 4GB 内存
- Redis：1CPU, 1GB 内存

## 8. 安全配置

### 8.1 网络策略 (K8s)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: api-network-policy
spec:
  podSelector:
    matchLabels:
      app: api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: ingress
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    - podSelector:
        matchLabels:
          app: redis
```

### 8.2 Secret 管理

```bash
# 创建 Secret
kubectl create secret generic go-manus-secrets \
  --from-literal=LLM_API_KEY=sk-xxx \
  --from-literal=DATABASE_PASSWORD=xxx \
  -n go-manus

# 使用 Secret
env:
- name: LLM_API_KEY
  valueFrom:
    secretKeyRef:
      name: go-manus-secrets
      key: LLM_API_KEY
```

## 9. 故障排除

### 9.1 常见问题

| 问题 | 解决方案 |
|------|---------|
| 服务启动失败 | 检查日志：`docker-compose logs api` |
| 数据库连接失败 | 确认网络和凭据 |
| 前端无法访问 API | 检查 CORS 和代理配置 |
| 沙箱执行超时 | 增加超时时间或资源 |

### 9.2 日志查看

```bash
# Docker
docker-compose logs -f [service]

# K8s
kubectl logs -f deployment/api -n go-manus
kubectl describe pod [pod-name] -n go-manus
```

## 10. 性能优化

### 10.1 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_messages_session_id ON messages(session_id);

-- 连接池配置
ALTER SYSTEM SET max_connections = 100;
```

### 10.2 Redis 优化

```bash
# 限制内存
redis-server --maxmemory 512mb --maxmemory-policy allkeys-lru
```

### 10.3 API 优化

```yaml
# K8s HPA 自动扩缩容
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: go-manus-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```