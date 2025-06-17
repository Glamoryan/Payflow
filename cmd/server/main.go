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

	shutdownTracing, err := tracing.InitTracer(
		"payflow",
		"1.0.0",
		"jaeger:4317",
	)
	if err != nil {
		log.Error("Tracing başlatılamadı", map[string]interface{}{"error": err.Error()})
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

				metrics.UpdateWorkerPoolStats(
					stats.QueueLength,
					3,
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

	log.Info("Tüm route'lar register edildi", map[string]interface{}{
		"user_routes":        "✓",
		"transaction_routes": "✓",
		"balance_routes":     "✓",
		"audit_routes":       "✓",
	})

	mux.HandleFunc("/debug/routes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("Registered routes:\n"))
			w.Write([]byte("GET /\n"))
			w.Write([]byte("GET /metrics\n"))
			w.Write([]byte("GET /debug/routes\n"))
			w.Write([]byte("Balance routes:\n"))
			w.Write([]byte("POST /api/balances/initialize\n"))
			w.Write([]byte("GET /api/balances/history\n"))
			w.Write([]byte("POST /api/balances/replay\n"))
			w.Write([]byte("POST /api/balances/rebuild\n"))
			w.Write([]byte("GET /api/balances\n"))
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			promhttp.Handler().ServeHTTP(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("PayFlow API is healthy!"))
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	var handler http.Handler = mux
	handler = middleware.TracingMiddleware(handler)
	handler = middleware.MetricsMiddleware(handler)

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
