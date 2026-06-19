package controller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
)

// BackupDatabase streams the underlying database for download.
// Currently only SQLite is supported (a copy of the .db file is sent).
// For MySQL / PostgreSQL the operator should use the native dump tools
// (`mysqldump`, `pg_dump`) — automated backup is not implemented for those
// drivers because the binaries are not guaranteed to be present in the
// runtime container.
func BackupDatabase(c *gin.Context) {
	if !common.UsingSQLite {
		driver := "MySQL"
		if common.UsingPostgreSQL {
			driver = "PostgreSQL"
		}
		c.JSON(http.StatusNotImplemented, gin.H{
			"success": false,
			"message": fmt.Sprintf(
				"Automated database backup is only supported for SQLite. Please use native dump tools (mysqldump / pg_dump) for %s.",
				driver,
			),
		})
		return
	}

	// Strip any SQLite DSN parameters (?_busy_timeout=...) to get the
	// real filesystem path of the database file.
	dbFilePath := common.SQLitePath
	if idx := strings.Index(dbFilePath, "?"); idx >= 0 {
		dbFilePath = dbFilePath[:idx]
	}

	absPath, err := filepath.Abs(dbFilePath)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": fmt.Sprintf("Database file not found: %s", err.Error()),
		})
		return
	}

	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("new-api-backup-%s.db", timestamp)

	c.Writer.Header().Set("Content-Description", "Database Backup")
	c.Writer.Header().Set("Content-Type", "application/octet-stream")
	c.Writer.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename),
	)
	c.Writer.Header().Set("Content-Transfer-Encoding", "binary")
	c.Writer.Header().Set("Expires", "0")
	c.Writer.Header().Set("Cache-Control", "must-revalidate")
	c.Writer.Header().Set("Pragma", "public")
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	c.File(absPath)
}

// sqliteFilePath returns the on-disk path of the SQLite database without DSN
// parameters (`?_busy_timeout=...`).
func sqliteFilePath() string {
	dbFilePath := common.SQLitePath
	if idx := strings.Index(dbFilePath, "?"); idx >= 0 {
		dbFilePath = dbFilePath[:idx]
	}
	return dbFilePath
}

// RestoreDatabase accepts a multipart `file` upload, validates that it's a
// SQLite database (magic header `SQLite format 3\000`), writes it to a temp
// path, copies the current DB to a `.bak-<timestamp>` safety copy, atomically
// renames the temp file over the live DB, and forces the process to exit so
// the container restart applies the restored database.
//
// SQLite only. Returns 501 for MySQL/PostgreSQL.
func RestoreDatabase(c *gin.Context) {
	if !common.UsingSQLite {
		driver := "MySQL"
		if common.UsingPostgreSQL {
			driver = "PostgreSQL"
		}
		c.JSON(http.StatusNotImplemented, gin.H{
			"success": false,
			"message": fmt.Sprintf(
				"Automated database restore is only supported for SQLite. Please restore manually for %s.",
				driver,
			),
		})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "No backup file uploaded (expected multipart field `file`).",
		})
		return
	}

	uploaded, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer uploaded.Close()

	// Validate SQLite magic header.
	header := make([]byte, 16)
	if _, err := io.ReadFull(uploaded, header); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Uploaded file is too small to be a SQLite database.",
		})
		return
	}
	if string(header) != "SQLite format 3\x00" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Uploaded file is not a valid SQLite database (bad magic header).",
		})
		return
	}
	if _, err := uploaded.Seek(0, io.SeekStart); err != nil {
		common.ApiError(c, err)
		return
	}

	dbFilePath := sqliteFilePath()
	absPath, err := filepath.Abs(dbFilePath)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	// Write the upload to a temp file in the same directory so the final
	// rename is atomic (same filesystem).
	dir := filepath.Dir(absPath)
	tmpFile, err := os.CreateTemp(dir, ".db-restore-*.tmp")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tmpPath := tmpFile.Name()
	if _, err := io.Copy(tmpFile, uploaded); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		common.ApiError(c, err)
		return
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		_ = os.Remove(tmpPath)
		common.ApiError(c, err)
		return
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		common.ApiError(c, err)
		return
	}

	// Copy the current DB to a timestamped safety backup.
	timestamp := time.Now().UTC().Format("20060102-150405")
	safetyPath := fmt.Sprintf("%s.bak-%s", absPath, timestamp)
	if _, statErr := os.Stat(absPath); statErr == nil {
		if copyErr := copyFile(absPath, safetyPath); copyErr != nil {
			_ = os.Remove(tmpPath)
			common.ApiError(c, copyErr)
			return
		}
	}

	// Atomic replace.
	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database restored. The server will restart momentarily.",
		"data": gin.H{
			"safety_backup": filepath.Base(safetyPath),
		},
	})

	// Force the process to exit after the response has flushed so the
	// container's restart policy reloads the DB cleanly. 1.5s delay gives
	// the HTTP write time to complete.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		os.Exit(0)
	}()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
