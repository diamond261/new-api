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
	"github.com/QuantumNous/new-api/model"
)

// BackupDatabase streams a database backup for download.
// SQLite: sends the raw .db file.
// PostgreSQL / MySQL: sends a logical JSON dump (gzipped).
func BackupDatabase(c *gin.Context) {
	timestamp := time.Now().UTC().Format("20060102-150405")

	if common.UsingSQLite {
		dbFilePath := sqliteFilePath()
		absPath, err := filepath.Abs(dbFilePath)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		info, err := os.Stat(absPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": fmt.Sprintf("Database file not found: %s", err.Error())})
			return
		}
		filename := fmt.Sprintf("new-api-backup-%s.db", timestamp)
		c.Writer.Header().Set("Content-Description", "Database Backup")
		c.Writer.Header().Set("Content-Type", "application/octet-stream")
		c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Writer.Header().Set("Content-Transfer-Encoding", "binary")
		c.Writer.Header().Set("Expires", "0")
		c.Writer.Header().Set("Cache-Control", "must-revalidate")
		c.Writer.Header().Set("Pragma", "public")
		c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
		c.File(absPath)
		return
	}

	filename := fmt.Sprintf("new-api-backup-%s.json.gz", timestamp)
	c.Writer.Header().Set("Content-Type", "application/gzip")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Writer.Header().Set("Cache-Control", "no-store")

	if err := model.PerformLogicalBackup(c.Writer); err != nil {
		common.SysError("database backup failed: " + err.Error())
	}
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

// RestoreDatabase accepts a multipart `file` upload and restores the database.
// It auto-detects the file type by magic bytes:
//   - gzip magic (\x1f\x8b): logical JSON.gz restore (PostgreSQL, MySQL, SQLite)
//   - SQLite magic: file-swap restore (SQLite only)
func RestoreDatabase(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "No backup file uploaded (expected multipart field `file`)."})
		return
	}

	uploaded, err := fileHeader.Open()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer uploaded.Close()

	magic := make([]byte, 16)
	if _, err := io.ReadFull(uploaded, magic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Uploaded file is too small."})
		return
	}
	if _, err := uploaded.Seek(0, io.SeekStart); err != nil {
		common.ApiError(c, err)
		return
	}

	// Gzip magic bytes → logical JSON restore
	if magic[0] == 0x1f && magic[1] == 0x8b {
		if err := model.RestoreLogicalBackup(uploaded); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Restore failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "Database restored. The server will restart momentarily."})
		go func() {
			time.Sleep(1500 * time.Millisecond)
			os.Exit(0)
		}()
		return
	}

	// SQLite magic → file-swap restore (SQLite only)
	if string(magic) != "SQLite format 3\x00" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Unrecognized backup file format. Expected a .json.gz or .db backup."})
		return
	}
	if !common.UsingSQLite {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Cannot restore a SQLite backup to a PostgreSQL/MySQL database. Use a .json.gz backup instead."})
		return
	}

	dbFilePath := sqliteFilePath()
	absPath, err := filepath.Abs(dbFilePath)
	if err != nil {
		common.ApiError(c, err)
		return
	}
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

	timestamp := time.Now().UTC().Format("20060102-150405")
	safetyPath := fmt.Sprintf("%s.bak-%s", absPath, timestamp)
	if _, statErr := os.Stat(absPath); statErr == nil {
		if copyErr := copyFile(absPath, safetyPath); copyErr != nil {
			_ = os.Remove(tmpPath)
			common.ApiError(c, copyErr)
			return
		}
	}
	if err := os.Rename(tmpPath, absPath); err != nil {
		_ = os.Remove(tmpPath)
		common.ApiError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Database restored. The server will restart momentarily.",
		"data":    gin.H{"safety_backup": filepath.Base(safetyPath)},
	})
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
