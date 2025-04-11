# PayFlow

- Katmanlı mimari yapısı
- Factory design pattern kullanımı
- Düzenli Go package yapısı
- Go modülleri ile dependency yönetimi
- Environment değişkenleri için sistem yapısı
- Zerolog ile loglama
- Graceful shutdown handling
- Thread-safe bakiye güncellemeleri
- Eşzamanlı işlem sistemi (Worker Pool)
- Rol tabanlı yetkilendirme
- API Anahtarı tabanlı kimlik doğrulama
- İşlem geri alma mekanizması
- Bakiye geçmişi izleme

## Başlangıç

```bash
# Environment değişkenlerini yapılandırın
cp .env.example .env
# Gerekli değişiklikleri .env dosyasında yapın

# Uygulamayı başlatın
go run cmd/server/main.go
```

## Authentication

```bash
# Yeni Kullanıcı Oluşturma
curl -X POST http://localhost:8080/api/users -H "Content-Type: application/json" -d '{
  "username": "admin",
  "email": "admin@example.com",
  "password": "securepassword",
  "role": "admin"
}'

# Normal kullanıcı oluşturma
curl -X POST http://localhost:8080/api/users -H "Content-Type: application/json" -d '{
  "username": "user1",
  "email": "user1@example.com",
  "password": "userpassword",
  "role": "user"
}'

# Giriş yapma ve API anahtarı alma
curl -X POST http://localhost:8080/api/login -H "Content-Type: application/json" -d '{
  "username": "admin",
  "password": "securepassword"
}'

# API anahtarı yenileme
curl -X POST http://localhost:8080/api/users/api-key -H "X-API-Key: <your_api_key>"
```

## Kullanıcı İşlemleri

```bash
# Kullanıcı Bilgilerini Görüntüleme
curl -X GET "http://localhost:8080/api/users?id=1" -H "X-API-Key: <your_api_key>"

# Kullanıcı Güncelleme
curl -X PUT http://localhost:8080/api/users -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"id": 1, "username": "updateduser", "email": "updated@example.com", "role": "admin"}'

# Kullanıcı Silme
curl -X DELETE "http://localhost:8080/api/users?id=1" -H "X-API-Key: <your_api_key>"
```

## Bakiye İşlemleri

```bash
# Bakiye Oluşturma
curl -X POST "http://localhost:8080/api/balances/initialize?user_id=1" -H "X-API-Key: <your_api_key>"

# Bakiye Görüntüleme
curl -X GET "http://localhost:8080/api/balances?user_id=1" -H "X-API-Key: <your_api_key>"

# Bakiye Geçmişi Görüntüleme
curl -X GET "http://localhost:8080/api/balances/history?user_id=1&limit=10&offset=0" -H "X-API-Key: <your_api_key>"

# Tarih Aralığına Göre Bakiye Geçmişi
curl -X GET "http://localhost:8080/api/balances/history/date-range?user_id=1&start_date=2023-01-01T00:00:00Z&end_date=2023-12-31T23:59:59Z" -H "X-API-Key: <your_api_key>"

# Bakiyeyi Yeniden Hesaplama (Optimizasyon)
curl -X POST "http://localhost:8080/api/balances/recalculate?user_id=1" -H "X-API-Key: <your_api_key>"
```

## Para İşlemleri

```bash
# Para Yatırma
curl -X POST http://localhost:8080/api/transactions/deposit -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"user_id": 1, "amount": 1000, "description": "İlk bakiye yükleme"}'

# Para Çekme
curl -X POST http://localhost:8080/api/transactions/withdraw -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"user_id": 1, "amount": 200, "description": "Test para çekme"}'

# Para Transferi
curl -X POST http://localhost:8080/api/transactions/transfer -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"from_user_id": 1, "to_user_id": 2, "amount": 300, "description": "Test transfer"}'

# Toplu İşlem (Batch Transaction)
curl -X POST http://localhost:8080/api/transactions/batch -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{
       "transactions": [
         {"sender_id": 1, "receiver_id": 2, "amount": 100, "description": "Test işlem 1"},
         {"sender_id": 1, "receiver_id": 3, "amount": 150, "description": "Test işlem 2"},
         {"sender_id": 2, "receiver_id": 1, "amount": 75, "description": "Test işlem 3"}
       ]
     }'
```

## İşlem Geçmişi ve Yönetimi

```bash
# Kullanıcının İşlem Geçmişini Görüntüleme
curl -X GET "http://localhost:8080/api/user-transactions?user_id=1" -H "X-API-Key: <your_api_key>"

# Belirli Bir İşlemi Görüntüleme
curl -X GET "http://localhost:8080/api/transactions?id=1" -H "X-API-Key: <your_api_key>"

# İşlem İstatistiklerini Görüntüleme (Admin yetkisi gerektirir)
curl -X GET http://localhost:8080/api/transactions/stats -H "X-API-Key: <admin_api_key>"

# İşlemi Geri Alma (Rollback) (Admin yetkisi gerektirir)
curl -X POST "http://localhost:8080/api/transactions/rollback?id=1" -H "X-API-Key: <admin_api_key>"
```

## Denetim Günlükleri

```bash
# Tüm Denetim Günlüklerini Görüntüleme
curl -X GET "http://localhost:8080/api/audit-logs" -H "X-API-Key: <your_api_key>"

# Belirli Bir Varlık İçin Denetim Günlüklerini Görüntüleme
curl -X GET "http://localhost:8080/api/entity-logs?entity_type=balance&entity_id=1" -H "X-API-Key: <your_api_key>"
```

## Thread-Safe Bakiye İşlemleri
```bash
# Thread-Safe Para Yatırma (Paralel çalıştırılabilir)
curl -X POST http://localhost:8080/api/transactions/deposit -H "Content-Type: application/json" -H "X-API-Key: <your_api_key>" \
     -d '{"user_id": 1, "amount": 500, "description": "Paralel yükleme 1"}'
```

## Worker Pool ve İşlem Kuyruğu

## Örnek Akış

1. Sistemi ilk kez başlattığınızda:

```bash
# Veritabanını temizleme
rm -f payflow.db

# Uygulamayı başlatma
go run cmd/server/main.go
```

2. Kullanıcı oluşturma ve giriş yapma:

```bash
# Admin kullanıcısı oluşturma
curl -X POST http://localhost:8080/api/users -H "Content-Type: application/json" -d '{
  "username": "admin",
  "email": "admin@example.com",
  "password": "securepassword",
  "role": "admin"
}'

# Giriş yapma
curl -X POST http://localhost:8080/api/login -H "Content-Type: application/json" -d '{
  "username": "admin",
  "password": "securepassword"
}'
# Bu işlem size bir API anahtarı döndürecektir, bunu not edin
```

3. API anahtarı ile işlem yapma:

```bash
# Para yatırma
curl -X POST http://localhost:8080/api/transactions/deposit -H "Content-Type: application/json" -H "X-API-Key: <admin_api_key>" \
     -d '{"user_id": 1, "amount": 1000, "description": "İlk bakiye yükleme"}'

# İşlem istatistiklerini görüntüleme
curl -X GET http://localhost:8080/api/transactions/stats -H "X-API-Key: <admin_api_key>"
```