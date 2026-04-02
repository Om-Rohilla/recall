package vault

import (
	"database/sql"
	"fmt"
	"time"
)

// WrappedStats contains derived weekly productivity metrics.
type WrappedStats struct {
	TotalThisWeek       int
	TotalGrowth         int // compared to last week (percentage)
	TopCommands         []Command
	TopCategory         string
	BusiestHour         int // 0-23
	BusiestDay          time.Weekday
	MergeConflictsFixed int
}

// GetWrappedStats builds a statistical breakdown of the last 7 days.
func (s *Store) GetWrappedStats() (*WrappedStats, error) {
	stats := &WrappedStats{}
	now := time.Now().UTC()
	oneWeekAgo := now.AddDate(0, 0, -7).Format(time.RFC3339)
	twoWeeksAgo := now.AddDate(0, 0, -14).Format(time.RFC3339)

	// Total this week via context to capture all executions, not just distinct commands
	err := s.db.QueryRow(`SELECT COUNT(*) FROM contexts WHERE timestamp >= ?`, oneWeekAgo).Scan(&stats.TotalThisWeek)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("getting total this week: %w", err)
	}

	var totalLastWeek int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM contexts WHERE timestamp >= ? AND timestamp < ?`, twoWeeksAgo, oneWeekAgo).Scan(&totalLastWeek)
	if err == nil && totalLastWeek > 0 {
		stats.TotalGrowth = ((stats.TotalThisWeek - totalLastWeek) * 100) / totalLastWeek
	}

	// Top commands this week
	rows, err := s.db.Query(`
		SELECT c.raw, COUNT(ctx.id) as freq 
		FROM commands c 
		JOIN contexts ctx ON c.id = ctx.command_id 
		WHERE ctx.timestamp >= ? 
		GROUP BY c.id ORDER BY freq DESC LIMIT 3`, oneWeekAgo)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cmd Command
			if err := rows.Scan(&cmd.Raw, &cmd.Frequency); err == nil {
				stats.TopCommands = append(stats.TopCommands, cmd)
			}
		}
	}

	// top category
	err = s.db.QueryRow(`
		SELECT c.category 
		FROM commands c 
		JOIN contexts ctx ON c.id = ctx.command_id 
		WHERE ctx.timestamp >= ? AND c.category != '' 
		GROUP BY c.category ORDER BY COUNT(ctx.id) DESC LIMIT 1`, oneWeekAgo).Scan(&stats.TopCategory)
	if err != nil && err != sql.ErrNoRows {
		// Category might be empty
	}

	// Busiest hour (00-23 in UTC)
	err = s.db.QueryRow(`
		SELECT CAST(substr(timestamp, 12, 2) AS INTEGER) as hr, COUNT(*) as cnt 
		FROM contexts 
		WHERE timestamp >= ? 
		GROUP BY hr ORDER BY cnt DESC LIMIT 1`, oneWeekAgo).Scan(&stats.BusiestHour)
	if err != nil && err != sql.ErrNoRows {
		// Ignore
	}

	// Busiest day
	var dayInt int
	err = s.db.QueryRow(`
        SELECT CAST(strftime('%w', timestamp) AS INTEGER) as dy, COUNT(*) as cnt 
        FROM contexts 
        WHERE timestamp >= ? 
        GROUP BY dy ORDER BY cnt DESC LIMIT 1`, oneWeekAgo).Scan(&dayInt)
	if err == nil {
		stats.BusiestDay = time.Weekday(dayInt)
	}

	// Merge conflicts surviving
	err = s.db.QueryRow(`
		SELECT COUNT(*) 
		FROM contexts ctx 
		JOIN commands c ON c.id = ctx.command_id 
		WHERE ctx.timestamp >= ? AND (c.raw LIKE '%merge --abort%' OR c.raw LIKE '%merge --continue%')`, oneWeekAgo).Scan(&stats.MergeConflictsFixed)
	if err != nil && err != sql.ErrNoRows {
		// Ignore
	}

	return stats, nil
}
