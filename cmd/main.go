package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"
	_ "progression1/docs"
	"progression1/internal/repository"
	"progression1/internal/service"
	"progression1/internal/transport/https"
	"syscall"

	"github.com/joho/godotenv"
)

// @title Сервис динамической сегментации пользователей
// @version 1.0
// @host localhost:8080
// @BasePath /
func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := godotenv.Load(); err != nil {
		slog.Default().Warn("Could not load .env file. Using OS environment variables.", "err", err)
	}
	db, err := repository.ConnectToBase()
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}
	npsri := repository.NewPgxSegmentRepo(db)
	userService := service.NewUserService(npsri)
	httpHandlers := https.NewHTTPHandlers(userService)
	port := os.Getenv("APP_PORT")
	if port == "" {
		log.Fatal("APP_PORT not set in environment or .env file.")
	}
	srv := https.NewHTTPServer(httpHandlers, port)
	if err := repository.RunMigrations(db); err != nil {
		log.Fatal("Failed to run database migrations: ", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	shutdownTimeout := os.Getenv("SHUTDOWN_TIMEOUT")
	if shutdownTimeout == "" {
		log.Fatal("SHUTDOWN_TIMEOUT not set in environment or .env file.")
	}
	if err := https.StartServer(ctx, srv, db, shutdownTimeout); err != nil {
		log.Fatal(err)
	}
}
