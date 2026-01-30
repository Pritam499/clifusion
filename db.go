// Copyright 2013-2023 The Cobra Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cobra

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type CommandUsage struct {
	ID          int
	CommandPath string
	Args        string
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Success     bool
	ErrorMsg    string
}

type AnalyticsDB struct {
	db *sql.DB
}

var GlobalAnalyticsDB *AnalyticsDB

func InitAnalyticsDB() error {
	dbPath := filepath.Join(os.Getenv("HOME"), ".cobra", "analytics.db")
	os.MkdirAll(filepath.Dir(dbPath), 0755)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	GlobalAnalyticsDB = &AnalyticsDB{db: db}
	return GlobalAnalyticsDB.initSchema()
}

func (a *AnalyticsDB) initSchema() error {
	_, err := a.db.Exec(`
		CREATE TABLE IF NOT EXISTS command_usage (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command_path TEXT NOT NULL,
			args TEXT,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration_ms INTEGER,
			success BOOLEAN NOT NULL,
			error_msg TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_command_path ON command_usage(command_path);
		CREATE INDEX IF NOT EXISTS idx_start_time ON command_usage(start_time);
	`)
	return err
}

func (a *AnalyticsDB) RecordUsage(usage *CommandUsage) error {
	_, err := a.db.Exec(`
		INSERT INTO command_usage (command_path, args, start_time, end_time, duration_ms, success, error_msg)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, usage.CommandPath, usage.Args, usage.StartTime, usage.EndTime, usage.Duration.Milliseconds(), usage.Success, usage.ErrorMsg)
	return err
}

func (a *AnalyticsDB) GetUsageStats() ([]map[string]interface{}, error) {
	rows, err := a.db.Query(`
		SELECT command_path, COUNT(*) as count, AVG(duration_ms) as avg_duration,
		       SUM(CASE WHEN success THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as success_rate
		FROM command_usage
		GROUP BY command_path
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []map[string]interface{}
	for rows.Next() {
		var cmd string
		var count int
		var avgDur float64
		var successRate float64
		err := rows.Scan(&cmd, &count, &avgDur, &successRate)
		if err != nil {
			return nil, err
		}
		stats = append(stats, map[string]interface{}{
			"command":      cmd,
			"count":        count,
			"avg_duration": avgDur,
			"success_rate": successRate,
		})
	}
	return stats, nil
}

func (a *AnalyticsDB) GetRecentCommands(prefix string, limit int) ([]string, error) {
	query := `
		SELECT DISTINCT command_path
		FROM command_usage
		WHERE command_path LIKE ?
		ORDER BY start_time DESC
		LIMIT ?
	`
	rows, err := a.db.Query(query, prefix+"%", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cmds []string
	for rows.Next() {
		var cmd string
		err := rows.Scan(&cmd)
		if err != nil {
			return nil, err
		}
		cmds = append(cmds, cmd)
	}
	return cmds, nil
}

func (a *AnalyticsDB) Close() error {
	return a.db.Close()
}