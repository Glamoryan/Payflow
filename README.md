# PayFlow


## Kurulum

```bash

# High availability stack'i başlatın
docker compose up -d


# Load balancer üzerinden health check
curl http://localhost/health
```

### Legacy Single Instance Kurulum

```bash
# Legacy profil ile tek instance çalıştırma
docker compose --profile legacy up -d

# Loglar
docker compose logs -f app
```

### High Availability Yapısı

```
                    ┌─────────────┐
                    │    NGINX    │
                    │Load Balancer│
                    └─────────────┘
                           │
                    ┌──────┴──────┐
                    │             │
              ┌─────────┐   ┌─────────┐
              │  App1   │   │  App2   │
              │Instance │   │Instance │
              └─────────┘   └─────────┘
                    │             │
                    └──────┬──────┘
                           │
           ┌───────────────┼───────────────┐
           │               │               │
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │ DB Master   │ │DB Replica 1 │ │DB Replica 2 │
    │(Write)      │ │(Read)       │ │(Read)       │
    └─────────────┘ └─────────────┘ └─────────────┘
           │               │               │
    ┌─────────────┐ ┌─────────────┐ ┌─────────────┐
    │Redis Master │ │Redis Replica│ │   Cache     │
    │             │ │             │ │  Manager    │
    └─────────────┘ └─────────────┘ └─────────────┘
```

## API Kullanımı

### Sistem Health Checks

```bash
# Genel health check
curl http://localhost/health

# NGINX status
curl http://localhost:8080/nginx_status
```

### Authentication ve Başlangıç

```bash
# Yeni Kullanıcı Oluşturma
curl -X POST http://localhost/api/users -H "Content-Type: application/json" -d '{
  "username": "admin",
  "email": "admin@example.com",
  "password": "securepassword",
  "role": "admin"
}'

# Giriş yapma ve API anahtarı alma
curl -X POST http://localhost/api/login -H "Content-Type: application/json" -d '{
  "username": "admin",
  "password": "securepassword"
}'

# API anahtarı yenileme
curl -X POST http://localhost/api/users/api-key -H "X-API-Key: <your_api_key>"
```

### Kullanıcı İşlemleri

```bash
# Kullanıcı Bilgilerini Görüntüleme
curl -X GET "http://localhost/api/users?id=1" -H "X-API-Key: <your_api_key>"

# Kullanıcı Güncelleme
curl -X PUT http://localhost/api/users -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"id": 1, "username": "updateduser", "email": "updated@example.com", "role": "admin"}'

# Kullanıcı Silme
curl -X DELETE "http://localhost/api/users?id=1" -H "X-API-Key: <your_api_key>"
```

### Bakiye İşlemleri

```bash
# Bakiye Oluşturma
curl -X POST "http://localhost/api/balances/initialize?user_id=1" -H "X-API-Key: <your_api_key>"

# Bakiye Görüntüleme
curl -X GET "http://localhost/api/balances?user_id=1" -H "X-API-Key: <your_api_key>"

# Bakiye Geçmişi Görüntüleme
curl -X GET "http://localhost/api/balances/history?user_id=1&limit=10&offset=0" -H "X-API-Key: <your_api_key>"
```

### Para Transferi

```bash
# Para yatırma
curl -X POST http://localhost/api/transactions/deposit -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"user_id": 1, "amount": 100.50, "description": "Para yatırma"}'

# Para çekme
curl -X POST http://localhost/api/transactions/withdraw -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"user_id": 1, "amount": 50.25, "description": "Para çekme"}'

# Para transferi
curl -X POST http://localhost/api/transactions/transfer -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"from_user_id": 1, "to_user_id": 2, "amount": 25.00, "description": "Transfer"}'

# Toplu İşlem (Batch Transaction)
curl -X POST http://localhost/api/transactions/batch -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{
       "transactions": [
         {"sender_id": 1, "receiver_id": 2, "amount": 100, "description": "Test işlem 1"},
         {"sender_id": 1, "receiver_id": 3, "amount": 150, "description": "Test işlem 2"}
       ]
     }'
```


### Monitoring Dashboards
- **NGINX Load Balancer**: http://localhost
- **Uygulama Health Check**: http://localhost/health
- **Prometheus Metrics**: http://localhost:9090
- **Grafana Dashboard**: http://localhost:3000 (admin/admin)
- **Jaeger Tracing**: http://localhost:16686

### Service Ports
- **80**: NGINX Load Balancer (HTTP)
- **443**: NGINX Load Balancer (HTTPS)
- **5432**: PostgreSQL Master
- **5433**: PostgreSQL Replica 1
- **5434**: PostgreSQL Replica 2
- **6379**: Redis Master
- **6380**: Redis Replica

### Operasyonel Endpointler

```bash
# Database connection stats
curl http://localhost/health | jq '.services.connection_manager'

# Load balancer stats
curl http://localhost/health | jq '.services.load_balancer'

# Fallback mechanism stats  
curl http://localhost/health | jq '.services.fallback_manager'

# Cache istatistikleri
curl http://localhost/api/cache/stats

# Cache warm-up
curl -X POST http://localhost/api/cache/warmup

# İşlem istatistikleri (Admin yetkisi gerekir)
curl -X GET http://localhost/api/transactions/stats -H "X-API-Key: <admin_api_key>"
```

## Yüksek Erişilebilirlik Özellikleri

### Database Replication
- **Master-Slave Setup**: Yazma işlemleri master'da, okuma işlemleri replica'larda
- **Automatic Failover**: Read replica hataları durumunda master'a otomatik geçiş
- **Load Balancing**: Weighted round-robin ile read replicas arasında yük dağılımı

### Circuit Breaker Pattern
- **Database Operations**: Veritabanı hataları için circuit breaker koruması
- **External Services**: Dış servis çağrıları için otomatik koruma
- **Configurable Thresholds**: Yapılandırılabilir hata eşikleri

### Fallback Mechanisms
- **Cache Strategy**: Cache miss durumlarında yedek data sources
- **Retry Strategy**: Geçici hatalar için exponential backoff ile retry
- **Degraded Mode**: Kritik olmayan özelliklerin devre dışı bırakılması
- **Default Values**: Hizmet hataları durumunda varsayılan değerler

### Load Balancing
- **NGINX Upstream**: Multiple application instances
- **Health Checks**: Automatic unhealthy instance detection
- **Rate Limiting**: API endpoint protection
- **SSL Termination**: HTTPS support

## Konfigürasyon

### Environment Variables

```bash
# Database Master
DB_HOST=db-master
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=payflow
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=300

# Database Read Replicas
DB_READ_HOST_1=db-replica1
DB_READ_PORT_1=5432
DB_READ_WEIGHT_1=1

DB_READ_HOST_2=db-replica2
DB_READ_PORT_2=5432
DB_READ_WEIGHT_2=1

# Redis Configuration
REDIS_HOST=redis-master
REDIS_PORT=6379
REDIS_CLUSTER=false
REDIS_POOL_SIZE=20
REDIS_MIN_IDLE_CONNS=5

# Load Balancer
LB_ENABLED=false
LB_ALGORITHM=round_robin
LB_HEALTH_CHECK_PATH=/health/ready
LB_HEALTH_CHECK_INTERVAL=30
```


### Load Testing

```bash
# Apache Bench ile basit load test
ab -n 1000 -c 10 http://localhost/health

# Wrk ile advanced load test
wrk -t12 -c400 -d30s http://localhost/api/users/1
```

### Circuit Breaker Testing

```bash
# Database'i kapatarak circuit breaker test etme
docker compose stop db-master

# Health check ile durumu kontrol etme
curl http://localhost/health

# Database'i yeniden başlatma
docker compose start db-master
```
# Implement Event Sourcing

Örnek test flowu

```bash
# Add user
curl -X POST -H "Content-Type: application/json" -d '{"username": "testuser", "email": "test@example.com", "password": "password123"}' http://localhost:8080/api/users

# Init user
curl -X POST http://localhost:8080/api/balances/initialize?user_id=1

# Transactions
curl -X POST -H "Content-Type: application/json" -d '{"user_id": 1, "amount": 100}' http://localhost:8080/api/transactions/deposit

curl -X POST -H "Content-Type: application/json" -d '{"user_id": 1, "amount": 30}' http://localhost:8080/api/transactions/withdraw

# Balance check
curl http://localhost:8080/api/balances?user_id=1

# Replay
curl -X POST http://localhost:8080/api/balances/replay?user_id=1

# Rebuild
curl -X POST http://localhost:8080/api/balances/rebuild?user_id=1

# Balance check again
curl http://localhost:8080/api/balances?user_id=1
```

# Add Caching Layer

```bash
# User cache
curl -X POST "http://localhost:8080/api/cache/warmup" -H "Content-Type: application/json" -d '{"type": "user", "user_id": 1}' | jq

curl -X GET "http://localhost:8080/api/cache/keys?pattern=user:*" | jq

# Cache invalidation
curl -X POST "http://localhost:8080/api/cache/invalidate" -H "Content-Type: application/json" -d '{"user_id": 1}' | jq

curl -X POST "http://localhost:8080/api/cache/invalidate" -H "Content-Type: application/json" -d '{"pattern": "user:*"}' | jq

curl -X GET "http://localhost:8080/api/cache/keys?pattern=user:*" | jq
```