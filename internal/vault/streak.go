package vault

import (
	"fmt"
	"time"
)

// StreakInfo contains the user's command capture streak data.
type StreakInfo struct {
	CurrentStreak int    `json:"current_streak"` // Consecutive days with captures
	LongestStreak int    `json:"longest_streak"` // All-time longest streak
	TodayCount    int    `json:"today_count"`    // Commands captured today
	IsActiveToday bool   `json:"is_active_today"`
	StreakEmoji   string `json:"streak_emoji"`   // Visual indicator
}

// GetCurrentStreak calculates the user's current consecutive capture day streak.
func (s *Store) GetCurrentStreak() (StreakInfo, error) {
	// Get distinct active dates ordered descending
	rows, err := s.db.Query(`
		SELECT DISTINCT date(last_seen) as d
		FROM commands
		WHERE last_seen IS NOT NULL AND last_seen != ''
		ORDER BY d DESC
		LIMIT 365
	`)
	if err != nil {
		return StreakInfo{}, fmt.Errorf("querying streak: %w", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			continue
		}
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		dates = append(dates, t)
	}

	if len(dates) == 0 {
		return StreakInfo{StreakEmoji: "💤"}, nil
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	yesterday := today.Add(-24 * time.Hour)

	// Check if active today
	todayCount := 0
	if err := s.db.QueryRow(`
		SELECT COUNT(*) FROM commands
		WHERE date(last_seen) = date('now')
	`).Scan(&todayCount); err != nil {
		todayCount = 0
	}
	isActiveToday := todayCount > 0

	// Calculate current streak
	currentStreak := 0
	expectedDate := today

	// If not active today, check from yesterday
	if !isActiveToday {
		expectedDate = yesterday
	}

	for _, d := range dates {
		day := d.Truncate(24 * time.Hour)
		if day.Equal(expectedDate) {
			currentStreak++
			expectedDate = expectedDate.Add(-24 * time.Hour)
		} else if day.Before(expectedDate) {
			break
		}
	}

	// Calculate longest streak (scan all dates)
	longestStreak := 0
	if len(dates) > 0 {
		streak := 1
		for i := 1; i < len(dates); i++ {
			diff := dates[i-1].Sub(dates[i])
			if diff >= 23*time.Hour && diff <= 25*time.Hour {
				streak++
			} else {
				if streak > longestStreak {
					longestStreak = streak
				}
				streak = 1
			}
		}
		if streak > longestStreak {
			longestStreak = streak
		}
	}

	if currentStreak > longestStreak {
		longestStreak = currentStreak
	}

	emoji := streakEmoji(currentStreak)

	return StreakInfo{
		CurrentStreak: currentStreak,
		LongestStreak: longestStreak,
		TodayCount:    todayCount,
		IsActiveToday: isActiveToday,
		StreakEmoji:    emoji,
	}, nil
}

func streakEmoji(days int) string {
	switch {
	case days >= 100:
		return "💎"
	case days >= 30:
		return "🏆"
	case days >= 14:
		return "⚡"
	case days >= 7:
		return "🔥"
	case days >= 3:
		return "✨"
	case days >= 1:
		return "🌱"
	default:
		return "💤"
	}
}
