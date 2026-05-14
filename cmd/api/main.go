package main

import (
	"log"
	"net/http"
	"os"

	"shadow_api/internal/config"
	"shadow_api/internal/db"
	"shadow_api/internal/httpapi"
	"shadow_api/internal/repository/sqlrepo"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Open(cfg)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	repo := sqlrepo.New(database, cfg.Database.Driver)
	handler := httpapi.New(repo, cfg)

	addr := ":" + cfg.App.Port
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	log.Printf("listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Printf("server error: %v", err)
		os.Exit(1)
	}
}
