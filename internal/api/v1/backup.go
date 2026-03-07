package v1

import (
	"database/sql"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// BackupHandler returns an http.HandlerFunc that streams the SQLite database
// as a file download. It uses VACUUM INTO to create a consistent, defragmented
// copy of the database before streaming, then removes the temp file.
//
// Registered directly on the chi router (not via huma) because huma wraps all
// responses in JSON, which is unsuitable for binary file downloads.
func BackupHandler(db *sql.DB, dbPath string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Create a temp file in the same directory as the DB so that the
		// VACUUM INTO path is on the same filesystem.
		dir := filepath.Dir(dbPath)
		tmp, err := os.CreateTemp(dir, "luminarr-backup-*.db")
		if err != nil {
			logger.ErrorContext(r.Context(), "backup: failed to create temp file", slog.Any("error", err))
			http.Error(w, "failed to create backup", http.StatusInternalServerError)
			return
		}
		tmpPath := tmp.Name()
		tmp.Close()
		defer func() { _ = os.Remove(tmpPath) }()

		// VACUUM INTO produces a consistent, compacted copy even under concurrent writes.
		if _, err := db.ExecContext(r.Context(), fmt.Sprintf("VACUUM INTO '%s'", tmpPath)); err != nil {
			logger.ErrorContext(r.Context(), "backup: VACUUM INTO failed", slog.Any("error", err))
			http.Error(w, "failed to create backup", http.StatusInternalServerError)
			return
		}

		f, err := os.Open(tmpPath)
		if err != nil {
			logger.ErrorContext(r.Context(), "backup: failed to open temp file", slog.Any("error", err))
			http.Error(w, "failed to read backup", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		filename := fmt.Sprintf("luminarr-backup-%s.db", time.Now().UTC().Format("2006-01-02"))
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, f); err != nil {
			logger.WarnContext(r.Context(), "backup: error streaming response", slog.Any("error", err))
		}
	}
}

// RestoreHandler returns an http.HandlerFunc that accepts a raw SQLite database
// file upload (application/octet-stream) and writes it to a staging path.
// On the next startup, Luminarr will detect the staging file and swap it in.
func RestoreHandler(dbPath string, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Override the global 1 MiB body limit — restore files are full DB copies.
		const maxRestoreSize = 500 << 20 // 500 MiB
		r.Body = http.MaxBytesReader(w, r.Body, maxRestoreSize)

		stagingPath := dbPath + ".restore"

		f, err := os.Create(stagingPath)
		if err != nil {
			logger.ErrorContext(r.Context(), "restore: failed to create staging file", slog.Any("error", err))
			http.Error(w, "failed to write restore file", http.StatusInternalServerError)
			return
		}
		defer f.Close()

		if _, err := io.Copy(f, r.Body); err != nil {
			// Clean up partial staging file on write failure.
			_ = os.Remove(stagingPath)
			logger.ErrorContext(r.Context(), "restore: failed to write body", slog.Any("error", err))
			http.Error(w, "failed to write restore file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"message":"Restore file saved. Restart Luminarr to complete the restore."}`))
	}
}
