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

	"payflow/internal/api"
	"payflow/internal/database"
	"payflow/pkg/factory"
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

	migrationService := database.NewMigrationService(db, log)
	if err := migrationService.RunMigrations(); err != nil {
		log.Fatal("Migrationlar uygulanamadı", map[string]interface{}{"error": err.Error()})
	}

	userService := appFactory.GetUserService()
	transactionService := appFactory.GetTransactionService()
	balanceService := appFactory.GetBalanceService()
	auditLogService := appFactory.GetAuditLogService()

	userHandler := api.NewUserHandler(userService, log)
	transactionHandler := api.NewTransactionHandler(transactionService, log)
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

	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
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
