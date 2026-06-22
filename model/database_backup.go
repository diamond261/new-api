package model

import (
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/QuantumNous/new-api/common"
)

// RestoreLogicalBackup reads a gzipped JSON dump (produced by PerformLogicalBackup)
// and restores all tables inside a single transaction.
func RestoreLogicalBackup(r io.Reader) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("not a valid gzip backup file: %w", err)
	}
	defer gz.Close()

	var dump struct {
		Tables map[string][]map[string]interface{} `json:"tables"`
	}
	if err := common.DecodeJson(gz, &dump); err != nil {
		return fmt.Errorf("decode backup: %w", err)
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		sess := tx.Session(&gorm.Session{PrepareStmt: false})
		switch {
		case common.UsingPostgreSQL:
			sess.Exec("SET LOCAL session_replication_role = 'replica'")
		case common.UsingMySQL:
			sess.Exec("SET FOREIGN_KEY_CHECKS = 0")
		}
		for table, rows := range dump.Tables {
			if err := sess.Exec("DELETE FROM " + quoteIdent(table)).Error; err != nil {
				return fmt.Errorf("clear %s: %w", table, err)
			}
			if len(rows) == 0 {
				continue
			}
			if err := sess.Table(table).CreateInBatches(rows, 200).Error; err != nil {
				return fmt.Errorf("insert %s: %w", table, err)
			}
		}
		return nil
	})
}

// PerformLogicalBackup writes a gzipped JSON dump of all tables to w.
func PerformLogicalBackup(w io.Writer) error {
	dbType := "sqlite"
	if common.UsingPostgreSQL {
		dbType = "postgresql"
	} else if common.UsingMySQL {
		dbType = "mysql"
	}

	tables, err := listBackupTables()
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	gz := gzip.NewWriter(w)

	header := fmt.Sprintf(`{"version":"1","db_type":%q,"created_at":%q,"tables":{`,
		dbType, time.Now().UTC().Format(time.RFC3339))
	if _, err := io.WriteString(gz, header); err != nil {
		gz.Close()
		return err
	}

	// Bypass PrepareStmt cache — dynamic SELECT * queries can't be prepared.
	sess := DB.Session(&gorm.Session{PrepareStmt: false})

	first := true
	for _, table := range tables {
		var rows []map[string]interface{}
		if res := sess.Raw("SELECT * FROM " + quoteIdent(table)).Scan(&rows); res.Error != nil {
			continue
		}
		encoded, err := common.Marshal(rows)
		if err != nil {
			continue
		}
		sep := ","
		if first {
			sep = ""
			first = false
		}
		if _, err := fmt.Fprintf(gz, "%s%q:%s", sep, table, encoded); err != nil {
			gz.Close()
			return err
		}
	}

	if _, err := io.WriteString(gz, "}}"); err != nil {
		gz.Close()
		return err
	}
	return gz.Close()
}

func quoteIdent(name string) string {
	if common.UsingMySQL {
		return "`" + strings.ReplaceAll(name, "`", "") + "`"
	}
	return `"` + strings.ReplaceAll(name, `"`, "") + `"`
}

func listBackupTables() ([]string, error) {
	var query string
	switch {
	case common.UsingPostgreSQL:
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'"
	case common.UsingMySQL:
		query = "SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'"
	default:
		query = "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'"
	}

	rows, err := DB.Session(&gorm.Session{PrepareStmt: false}).Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			tables = append(tables, name)
		}
	}
	return tables, rows.Err()
}
