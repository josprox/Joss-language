package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jossecurity/joss/pkg/parser"
)

var (
	cronTickerStarted bool
	cronTickerMutex   sync.Mutex
	scheduledTasks    = make(map[string]*parser.BlockStatement)
	scheduledTasksMu  sync.RWMutex
)

// EnsureCronTable creates the cron table if it doesn't exist
func (r *Runtime) EnsureCronTable() {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "cron"

	dbDriver := "mysql"
	if val, ok := r.Env["DB"]; ok {
		dbDriver = val
	}

	var query string
	if dbDriver == "mysql" {
		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INT AUTO_INCREMENT PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE,
			schedule VARCHAR(255) NOT NULL,
			last_run_at DATETIME,
			is_running BOOLEAN DEFAULT 0,
			status VARCHAR(50)
		);
		`, tableName)
	} else {
		query = fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name VARCHAR(255) NOT NULL UNIQUE,
			schedule VARCHAR(255) NOT NULL,
			last_run_at DATETIME,
			is_running BOOLEAN DEFAULT 0,
			status VARCHAR(50)
		);
		`, tableName)
	}

	_, err := r.GetDB().Exec(query)
	if err != nil {
		fmt.Printf("[Cron] Error creando tabla %s: %v\n", tableName, err)
	}
}

// StartCronTicker starts the background evaluation loop
func (r *Runtime) StartCronTicker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	fmt.Println("[Cron] Ticker de tareas programadas iniciado.")

	for range ticker.C {
		r.TickCron()
	}
}

// TickCron checks scheduled tasks against the current time
func (r *Runtime) TickCron() {
	if r.GetDB() == nil {
		return
	}

	prefix := r.dbPrefix()
	tableName := prefix + "cron"

	rows, err := r.GetDB().Query(fmt.Sprintf("SELECT name, schedule, is_running FROM %s", tableName))
	if err != nil {
		return
	}
	defer rows.Close()

	now := time.Now()

	for rows.Next() {
		var name, expr string
		var isRunning bool
		if err := rows.Scan(&name, &expr, &isRunning); err != nil {
			continue
		}

		if isRunning {
			continue
		}

		// Match cron expression
		if MatchCron(expr, now) {
			scheduledTasksMu.RLock()
			block, exists := scheduledTasks[name]
			scheduledTasksMu.RUnlock()

			if exists {
				fmt.Printf("[Cron] Disparando tarea '%s' programada (%s)...\n", name, expr)
				r.RunCronTask(name, block)
			}
		}
	}
}

// RunCronTask handles execution safety and locking
func (r *Runtime) RunCronTask(name string, block *parser.BlockStatement) {
	prefix := r.dbPrefix()
	tableName := prefix + "cron"

	// Lock task
	_, err := r.GetDB().Exec(fmt.Sprintf("UPDATE %s SET is_running = 1, status = 'running' WHERE name = ?", tableName), name)
	if err != nil {
		return
	}

	newR := r.Fork()
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				fmt.Printf("[Cron] Error en tarea %s: %v\n", name, rec)
				if r.GetDB() != nil {
					_, _ = r.GetDB().Exec(fmt.Sprintf("UPDATE %s SET is_running = 0, status = 'error', last_run_at = CURRENT_TIMESTAMP WHERE name = ?", tableName), name)
				}
			} else {
				if r.GetDB() != nil {
					_, _ = r.GetDB().Exec(fmt.Sprintf("UPDATE %s SET is_running = 0, status = 'completed', last_run_at = CURRENT_TIMESTAMP WHERE name = ?", tableName), name)
				}
			}
		}()
		newR.executeBlock(block)
	}()
}

// MatchCron evaluates standard cron syntax
func MatchCron(expr string, t time.Time) bool {
	// Support friendly aliases
	expr = strings.TrimSpace(strings.ToLower(expr))
	if expr == "hourly" {
		expr = "0 * * * *"
	} else if expr == "daily" {
		expr = "0 0 * * *"
	} else if expr == "weekly" {
		expr = "0 0 * * 0"
	} else if expr == "monthly" {
		expr = "0 0 1 * *"
	}

	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return false
	}

	min := t.Minute()
	hour := t.Hour()
	dom := t.Day()
	month := int(t.Month())
	dow := int(t.Weekday()) // 0 = Sunday, 1 = Monday, ...

	return matchCronField(parts[0], min, 0, 59) &&
		matchCronField(parts[1], hour, 0, 23) &&
		matchCronField(parts[2], dom, 1, 31) &&
		matchCronField(parts[3], month, 1, 12) &&
		matchCronField(parts[4], dow, 0, 6)
}

func matchCronField(field string, val int, minVal, maxVal int) bool {
	if field == "*" {
		return true
	}

	// Step (*/X)
	if strings.HasPrefix(field, "*/") {
		var step int
		_, err := fmt.Sscanf(field, "*/%d", &step)
		if err == nil && step > 0 {
			return val%step == 0
		}
	}

	// List (e.g. 0,12)
	parts := strings.Split(field, ",")
	if len(parts) > 1 {
		for _, p := range parts {
			var item int
			if _, err := fmt.Sscanf(p, "%d", &item); err == nil {
				if item == val {
					return true
				}
			}
		}
		return false
	}

	// Exact number
	var item int
	if _, err := fmt.Sscanf(field, "%d", &item); err == nil {
		return item == val
	}

	return false
}
