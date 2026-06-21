package model

import (
	"compress/gzip"
	"fmt"
	"io"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// PerformLogicalBackup writes a gzipped JSON dump of all tables to w.
// Works with PostgreSQL, MySQL, and SQLite.
func PerformLogicalBackup(w io.Writer) error {
	dbType := "sqlite"
	if common.UsingPostgreSQL {
		dbType = "postgresql"
	} else if common.UsingMySQL {
		dbType = "mysql"
	}

	gz := gzip.NewWriter(w)
	defer gz.Close()

	tables, err := listBackupTables()
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	header := fmt.Sprintf(`{"version":"1","db_type":%q,"created_at":%q,"tables":{`,
		dbType, time.Now().UTC().Format(time.RFC3339))
	if _, err := io.WriteString(gz, header); err != nil {
		return err
	}

	first := true
	for _, table := range tables {
		var rows []map[string]interface{}
		if res := DB.Table(table).Find(&rows); res.Error != nil {
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
			return err
		}
	}

	_, err = io.WriteString(gz, "}}")
	return err
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

	rows, err := DB.Raw(query).Rows()
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
