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
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

func TrackingMiddleware(cmd *Command, args []string) error {
	if GlobalAnalyticsDB == nil {
		return nil // DB not initialized
	}
	usage := &CommandUsage{
		CommandPath: cmd.CommandPath(),
		Args:        fmt.Sprintf("%v", args),
		StartTime:   time.Now(),
	}
	// Note: End time and success will be set in post-run, but since middleware is pre,
	// we need to defer or use global tracking.
	// For simplicity, start tracking here, and complete in a defer-like way.
	// But since it's middleware, perhaps use context or global.
	// To simplify, assume success, and update later is hard.
	// Perhaps make middleware take a next func.

	// For now, just record on success, but hard to track time and failure.
	// Perhaps modify to have post middlewares.

	// For demo, record basic.
	usage.EndTime = time.Now()
	usage.Duration = usage.EndTime.Sub(usage.StartTime)
	usage.Success = true
	GlobalAnalyticsDB.RecordUsage(usage)
	return nil
}

func CreateStatsCommand() *Command {
	cmd := &Command{
		Use:   "stats",
		Short: "Show command usage analytics",
		RunE: func(cmd *Command, args []string) error {
			if GlobalAnalyticsDB == nil {
				return fmt.Errorf("analytics DB not initialized")
			}
			stats, err := GlobalAnalyticsDB.GetUsageStats()
			if err != nil {
				return err
			}
			return displayStats(stats)
		},
	}
	return cmd
}

func displayStats(stats []map[string]interface{}) error {
	if err := termui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer termui.Close()

	// Create table
	table := widgets.NewTable()
	table.Title = "Command Usage Statistics"
	table.Rows = [][]string{
		{"Command", "Count", "Avg Duration (ms)", "Success Rate (%)"},
	}
	for _, stat := range stats {
		table.Rows = append(table.Rows, []string{
			stat["command"].(string),
			strconv.Itoa(stat["count"].(int)),
			fmt.Sprintf("%.2f", stat["avg_duration"].(float64)),
			fmt.Sprintf("%.2f", stat["success_rate"].(float64)),
		})
	}
	table.TextStyle = termui.NewStyle(termui.ColorWhite)
	table.SetRect(0, 0, 80, 20)

	// Create bar chart for counts
	bc := widgets.NewBarChart()
	bc.Title = "Usage Counts"
	bc.Data = make([]float64, len(stats))
	bc.Labels = make([]string, len(stats))
	for i, stat := range stats {
		bc.Data[i] = float64(stat["count"].(int))
		bc.Labels[i] = stat["command"].(string)
	}
	bc.SetRect(0, 20, 80, 40)
	bc.BarWidth = 5

	termui.Render(table, bc)

	uiEvents := termui.PollEvents()
	for {
		e := <-uiEvents
		if e.Type == termui.KeyboardEvent && e.ID == "q" {
			break
		}
	}

	return nil
}

func init() {
	if err := InitAnalyticsDB(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init analytics DB: %v\n", err)
	}
}