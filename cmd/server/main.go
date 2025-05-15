package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"payflow/internal/api"
	"payflow/internal/api/middleware"
	"payflow/internal/database"
	"payflow/pkg/factory"
	"payflow/pkg/metrics"
	"payflow/pkg/tracing"
)

func main() {
	appFactory, err := factory.NewFactory()
	if err != nil {
		fmt.Printf("Factory oluşturulamadı: %v\n", err)
		os.Exit(1)
	}

	log := appFactory.GetLogger()
	cfg := appFactory.GetConfig()
	db := appFactory.GetDB()

	defer db.Close()

	log.Info("Uygulama başlatılıyor", map[string]interface{}{"env": cfg.AppEnv})

	// Tracing başlat
	shutdownTracing, err := tracing.InitTracer(
		"payflow",     // Servis adı
		"1.0.0",       // Servis sürümü
		"jaeger:4317", // Jaeger OTLP endpoint
	)
	if err != nil {
		log.Error("Tracing başlatılamadı", map[string]interface{}{"error": err.Error()})
		// Kritik bir hata olmadığı için devam edebiliriz
	} else {
		defer shutdownTracing()
	}

	migrationService := database.NewMigrationService(db, log)
	if err := migrationService.RunMigrations(); err != nil {
		log.Fatal("Migrationlar uygulanamadı", map[string]interface{}{"error": err.Error()})
	}

	userService := appFactory.GetUserService()
	transactionService := appFactory.GetTransactionService()
	balanceService := appFactory.GetBalanceService()
	auditLogService := appFactory.GetAuditLogService()

	defer func() {
		log.Info("TransactionService kapatılıyor...", map[string]interface{}{})
		transactionService.Shutdown()
	}()

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				stats, err := transactionService.GetWorkerPoolStats()
				if err != nil {
					log.Error("Worker pool istatistikleri alınamadı", map[string]interface{}{"error": err.Error()})
					continue
				}

				// Prometheus worker pool metriklerini güncelle
				metrics.UpdateWorkerPoolStats(
					stats.QueueLength,
					3, // Sabit bir değer (active workers sayısı)
				)

				log.Info("Worker Pool İstatistikleri", map[string]interface{}{
					"submitted":        stats.Submitted,
					"completed":        stats.Completed,
					"failed":           stats.Failed,
					"rejected":         stats.Rejected,
					"avg_process_time": stats.AvgProcessTime.String(),
					"queue_length":     stats.QueueLength,
					"queue_capacity":   stats.QueueCapacity,
				})
			}
		}
	}()

	userHandler := api.NewUserHandler(userService, log)
	transactionHandler := api.NewTransactionHandler(transactionService, userService, log)
	balanceHandler := api.NewBalanceHandler(balanceService, log)
	auditLogHandler := api.NewAuditLogHandler(auditLogService, log)

	mux := http.NewServeMux()

	userHandler.RegisterRoutes(mux)
	transactionHandler.RegisterRoutes(mux)
	balanceHandler.RegisterRoutes(mux)
	auditLogHandler.RegisterRoutes(mux)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("PayFlow API'ye Hoş Geldiniz!"))
	})

	// Prometheus metrik endpoint'i
	mux.Handle("GET /metrics", promhttp.Handler())

	// Middleware zinciri oluştur
	var handler http.Handler = mux
	handler = middleware.TracingMiddleware(handler) // Tracing önce
	handler = middleware.MetricsMiddleware(handler) // Metrics sonra

	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: handler,
	}

	go func() {
		log.Info("HTTP sunucusu başlatılıyor", map[string]interface{}{"port": cfg.Server.Port})

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP sunucusu başlatılamadı", map[string]interface{}{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Sunucu kapatılıyor...", map[string]interface{}{})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Sunucu kapatılırken hata oluştu", map[string]interface{}{"error": err.Error()})
	}

	log.Info("Sunucu başarıyla kapatıldı", map[string]interface{}{})
}
