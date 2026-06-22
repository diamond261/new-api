package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gopkg/util/gopool"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
)

const (
	autoBackupDir      = "/data/backups"
	autoBackupKeepLast = 7
)

var backupTaskOnce sync.Once

func StartDatabaseAutoBackupTask() {
	backupTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("database auto-backup task started: runs daily at %02d:00 UTC", common.AutoBackupHour))
			for {
				time.Sleep(durationUntilHour(common.AutoBackupHour))
				runAutoBackup()
			}
		})
	})
}

func durationUntilHour(hour int) time.Duration {
	now := time.Now().UTC()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return time.Until(next)
}

func runAutoBackup() {
	ctx := context.Background()
	if err := os.MkdirAll(autoBackupDir, 0o755); err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("auto-backup: cannot create backup dir: %v", err))
		return
	}

	timestamp := time.Now().Format("20060102-150405")

	if common.UsingSQLite {
		path := filepath.Join(autoBackupDir, fmt.Sprintf("new-api-backup-%s.db", timestamp))
		src := common.SQLitePath
		if i := strings.IndexByte(src, '?'); i >= 0 {
			src = src[:i]
		}
		if err := copyFileToPath(src, path); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("auto-backup (sqlite) failed: %v", err))
			return
		}
		logger.LogInfo(ctx, fmt.Sprintf("auto-backup saved: %s", path))
		notifyBackup(ctx, path)
	} else {
		path := filepath.Join(autoBackupDir, fmt.Sprintf("new-api-backup-%s.json.gz", timestamp))
		f, err := os.Create(path)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("auto-backup: cannot create file: %v", err))
			return
		}
		err = model.PerformLogicalBackup(f)
		f.Close()
		if err != nil {
			_ = os.Remove(path)
			logger.LogWarn(ctx, fmt.Sprintf("auto-backup (logical) failed: %v", err))
			return
		}
		logger.LogInfo(ctx, fmt.Sprintf("auto-backup saved: %s", path))
		notifyBackup(ctx, path)
	}

	pruneOldBackups(ctx)
}

func copyFileToPath(src, dst string) error {
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
	_, err = io.Copy(out, in)
	return err
}

func notifyBackup(ctx context.Context, path string) {
	if common.AutoBackupTelegramEnabled && common.AutoBackupTelegramBotToken != "" {
		if err := sendTelegramDocument(common.AutoBackupTelegramBotToken, path); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("auto-backup: telegram notify failed: %v", err))
		}
	}
}

func sendTelegramDocument(token, filePath string) error {
	chatID := common.AutoBackupTelegramChatID
	if chatID == "" {
		return fmt.Errorf("AutoBackupTelegramChatID not configured")
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("chat_id", chatID)
	_ = mw.WriteField("caption", fmt.Sprintf("new-api auto-backup %s", time.Now().UTC().Format("2006-01-02 15:04 UTC")))
	fw, err := mw.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return err
	}
	if _, err = io.Copy(fw, f); err != nil {
		return err
	}
	mw.Close()

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", token)
	resp, err := http.Post(url, mw.FormDataContentType(), &buf) //nolint:noctx
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func pruneOldBackups(ctx context.Context) {
	entries, err := os.ReadDir(autoBackupDir)
	if err != nil {
		return
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, filepath.Join(autoBackupDir, e.Name()))
		}
	}
	if len(files) <= autoBackupKeepLast {
		return
	}
	sort.Strings(files)
	for _, f := range files[:len(files)-autoBackupKeepLast] {
		if err := os.Remove(f); err == nil {
			logger.LogInfo(ctx, fmt.Sprintf("auto-backup: pruned old backup: %s", filepath.Base(f)))
		}
	}
}
