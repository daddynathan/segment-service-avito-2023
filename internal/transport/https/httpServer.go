package https

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func NewHTTPServer(httpHandler *HTTPHandlers, addr string) *http.Server {
	mux := http.NewServeMux()
	docsDir := filepath.Join(".", "docs")
	mux.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir(docsDir))))
	mux.HandleFunc("/swagger/index.html", httpSwagger.Handler(
		httpSwagger.URL("/swagger/swagger.json"),
	))
	mux.HandleFunc("/segments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			httpHandler.HandleAddSegment(w, r)
		case http.MethodGet:
			httpHandler.HandleGetAllSegments(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	mux.HandleFunc("/segments/history", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		httpHandler.HandleGetH(w, r)
	})
	mux.HandleFunc("/user/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			httpHandler.HandleAddUserToSegment(w, r)
		case http.MethodGet:
			httpHandler.HandleGetUserSegments(w, r)
		case http.MethodPatch:
			httpHandler.HandleUpdateUserSegments(w, r)
		default:
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	})
	return &http.Server{
		Handler: mux,
		Addr:    addr,
	}
}

func StartServer(ctx context.Context, srv *http.Server, db *sql.DB, shutdownTimeouts string) error {
	shutdownTimeouti, err := strconv.Atoi(shutdownTimeouts)
	if err != nil {
		return fmt.Errorf("convertation from .env file failed: %w", err)
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Default().Error("server ListenAndServe failed", "error", err)
		}
	}()
	slog.Default().Info("server ListenAndServe successfully", "addr", srv.Addr)
	<-ctx.Done()
	slog.Default().Info("shutting down server gracefully", "shutdownTimeout", shutdownTimeouti)
	shutdownTimeoutDuration := time.Second * time.Duration(shutdownTimeouti)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeoutDuration)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	slog.Default().Info("server successfully shut down")
	slog.Default().Info("closing database connection")
	if err := db.Close(); err != nil {
		return fmt.Errorf("error closing database: %w", err)
	}
	slog.Default().Info("database closed")
	return nil
}
