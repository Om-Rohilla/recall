package context

import (
	"sync"
	"time"
)

type SessionEntry struct {
	Command   string
	Timestamp time.Time
	ExitCode  int
	Category  string
}

type Session struct {
	ID         string
	StartedAt  time.Time
	Recent     []SessionEntry
	mu         sync.Mutex
	maxRecent  int
}

var (
	currentSession *Session
	sessionMu      sync.Mutex
)

func GetSession(sessionID string) *Session {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	if currentSession != nil && currentSession.ID == sessionID {
		return currentSession
	}

	currentSession = &Session{
		ID:        sessionID,
		StartedAt: time.Now(),
		maxRecent: 20,
	}
	return currentSession
}

func (s *Session) AddCommand(cmd string, exitCode int, category string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry := SessionEntry{
		Command:   cmd,
		Timestamp: time.Now(),
		ExitCode:  exitCode,
		Category:  category,
	}

	s.Recent = append(s.Recent, entry)
	if len(s.Recent) > s.maxRecent {
		s.Recent = s.Recent[len(s.Recent)-s.maxRecent:]
	}
}

func (s *Session) RecentCommands(n int) []SessionEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	if n <= 0 || n > len(s.Recent) {
		n = len(s.Recent)
	}
	if n == 0 {
		return nil
	}

	start := len(s.Recent) - n
	result := make([]SessionEntry, n)
	copy(result, s.Recent[start:])
	return result
}

func (s *Session) RecentCategories() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cats := make(map[string]int)
	for _, e := range s.Recent {
		if e.Category != "" {
			cats[e.Category]++
		}
	}
	return cats
}
