package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	http_handler "github.com/vantutran2k1/orbit/internal/adapter/handler/http"
	"github.com/vantutran2k1/orbit/internal/adapter/handler/socket"
	"github.com/vantutran2k1/orbit/internal/adapter/storage/postgres"
	"github.com/vantutran2k1/orbit/internal/adapter/storage/redis"
	"github.com/vantutran2k1/orbit/internal/core/service"
)

func main() {
	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	dbPool, err := pgxpool.New(context.Background(), dbUrl)
	if err != nil {
		log.Fatal("unable to connect to database:", err)
	}
	defer dbPool.Close()

	redis.Init()

	jobRepo := postgres.NewJobRepository(dbPool)
	schedulerSvc := service.NewSchedulerService(jobRepo)
	jobHandler := http_handler.NewJobHandler(schedulerSvc)

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.SetHeader("Content-Type", "application/json"))

	r.Post("/jobs", jobHandler.Create)
	r.Get("/ws", socket.GlobalHub.HandleConnection)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("orbit api listening on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
