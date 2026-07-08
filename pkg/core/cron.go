package core

import (
	"database/sql"
	"fmt"

	"github.com/jossecurity/joss/pkg/parser"
)

// Cron Implementation (Daemon mode simulation)
func (r *Runtime) executeCronMethod(instance *Instance, method string, args []interface{}) interface{} {
	if method == "schedule" {
		if len(args) >= 3 {
			name := args[0].(string)
			schedule := args[1].(string)

			// 1. Register/Update Task in DB
			if r.GetDB() != nil {
				prefix := r.dbPrefix()
				tableName := prefix + "cron"

				var id int
				err := r.GetDB().QueryRow(fmt.Sprintf("SELECT id FROM %s WHERE name = ?", tableName), name).Scan(&id)
				if err == sql.ErrNoRows {
					_, err = r.GetDB().Exec(fmt.Sprintf("INSERT INTO %s (name, schedule, status) VALUES (?, ?, 'idle')", tableName), name, schedule)
				} else if err == nil {
					_, err = r.GetDB().Exec(fmt.Sprintf("UPDATE %s SET schedule = ? WHERE id = ?", tableName), schedule, id)
				}
				if err != nil {
					fmt.Printf("[Cron] Error registrando tarea %s: %v\n", name, err)
				}
			}

			if block, ok := args[2].(*parser.BlockStatement); ok {
				scheduledTasksMu.Lock()
				scheduledTasks[name] = block
				scheduledTasksMu.Unlock()

				cronTickerMutex.Lock()
				if !cronTickerStarted {
					cronTickerStarted = true
					// Start ticker in background
					go r.StartCronTicker()
				}
				cronTickerMutex.Unlock()
			}
		}
	}
	return nil
}
