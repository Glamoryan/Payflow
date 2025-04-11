# PayFlow

- Katmanlı mimari yapısı
- Factory design pattern kullanımı
- Düzenli Go package yapısı
- Go modülleri ile dependency yönetimi
- Environment değişkenleri için sistem yapısı
- Zerolog ile loglama
- Graceful shutdown handling

## Başlangıç

Projeyi çalıştırmak için:

```bash
# Environment değişkenlerini yapılandırın
cp .env.example .env
# Gerekli değişiklikleri .env dosyasında yapın

# Uygulamayı başlatın
go run cmd/server/main.go
```

## Requests

Aşağıdaki curl komutlarını kullanarak API'yi test edebilirsiniz:

### Kullanıcı İşlemleri

```bash
# Yeni Kullanıcı Oluşturma
curl --location 'http://localhost:8080/api/users' \
--header 'Content-Type: application/json' \
--data-raw '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "test123",
    "role": "user"
}'

# Kullanıcı Bilgilerini Görüntüleme
curl -X GET "http://localhost:8080/api/users?id=1"

# Kullanıcı Güncelleme
curl -X PUT http://localhost:8080/api/users -H "Content-Type: application/json" \
     -d '{"id": 1, "username": "updateduser", "email": "updated@example.com", "role": "admin"}'

# Kullanıcı Silme
curl -X DELETE "http://localhost:8080/api/users?id=1"
```

### Bakiye İşlemleri

```bash
# Bakiye Oluşturma
curl -X POST "http://localhost:8080/api/balances/initialize?user_id=1"

# Bakiye Görüntüleme
curl -X GET "http://localhost:8080/api/balances?user_id=1"
```

### Para İşlemleri

```bash
# Para Yatırma
curl -X POST http://localhost:8080/api/transactions/deposit -H "Content-Type: application/json" \
     -d '{"user_id": 1, "amount": 1000}'

# Para Çekme
curl -X POST http://localhost:8080/api/transactions/withdraw -H "Content-Type: application/json" \
     -d '{"user_id": 1, "amount": 200}'

# Para Transferi
curl -X POST http://localhost:8080/api/transactions/transfer -H "Content-Type: application/json" \
     -d '{"from_user_id": 1, "to_user_id": 2, "amount": 300}'
```

### İşlem Geçmişi

```bash
# Kullanıcının İşlem Geçmişini Görüntüleme
curl -X GET "http://localhost:8080/api/user-transactions?user_id=1"

# Belirli Bir İşlemi Görüntüleme
curl -X GET "http://localhost:8080/api/transactions?id=1"
```

### Denetim Günlükleri

```bash
# Tüm Denetim Günlüklerini Görüntüleme
curl -X GET "http://localhost:8080/api/audit-logs"

# Belirli Bir Varlık İçin Denetim Günlüklerini Görüntüleme
curl -X GET "http://localhost:8080/api/entity-logs?entity_type=balance&entity_id=1"

# Yeni Denetim Günlüğü Ekleme (genellikle sistem tarafından otomatik eklenir)
curl -X POST http://localhost:8080/api/audit-logs -H "Content-Type: application/json" \
     -d '{"entity_type": "user", "entity_id": 1, "action": "update", "details": "Manuel güncelleme işlemi"}'
```

# Thread-Safe & Multiple Transactions

```bash
# Thread-Safe Deposit
curl --location 'http://localhost:8080/api/transactions/deposit' \
--header 'Content-Type: application/json' \
--data '{
    "user_id": 1,
    "amount": 500,
    "description": "Paralel yükleme 1"
}'

# Transaction stats
curl --location 'http://localhost:8080/api/transactions/stats' \
--header 'x-api-key: 4aa9c8d94d3aeccb09ca9811b41975f596ce3f9995e062cd129becf6434d3ed6'

# Multiple Transactions
curl --location 'http://localhost:8080/api/transactions/batch' \
--header 'Content-Type: application/json' \
--data '{
  "transactions": [
    {"sender_id": 1, "receiver_id": 2, "amount": 100, "description": "Test işlem 1"},
    {"sender_id": 1, "receiver_id": 3, "amount": 150, "description": "Test işlem 2"},
    {"sender_id": 2, "receiver_id": 1, "amount": 75, "description": "Test işlem 3"},
    {"sender_id": 3, "receiver_id": 2, "amount": 200, "description": "Test işlem 4"}
  ]
}'
```

# Auth

Remove the payflow.db first

```bash
# Create admin user
   curl -X POST http://localhost:8080/api/users -H "Content-Type: application/json" -d '{
     "username": "admin",
     "email": "admin@example.com",
     "password": "securepassword",
     "role": "admin"
   }'

# Login and get the API key
   curl -X POST http://localhost:8080/api/login -H "Content-Type: application/json" -d '{
     "username": "admin",
     "password": "securepassword"
   }'

# Check stats (admin)
curl -X GET http://localhost:8080/api/transactions/stats -H "X-API-Key: <api-key>"
```