package service

import (
	"context"
	"fmt"
	"io"
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
			logger.LogInfo(context.Background(), "database auto-backup task started: runs daily at midnight local time")
			for {
				time.Sleep(durationUntilMidnight())
				runAutoBackup()
			}
		})
	})
}

func durationUntilMidnight() time.Duration {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
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
