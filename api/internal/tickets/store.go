package tickets

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Ticket struct {
	ServerUUID string
	UserID     uint64
	ExpiresAt  time.Time
}

type Store struct {
	mu   sync.Mutex
	data map[string]Ticket
}

func NewStore() *Store {
	s := &Store{data: make(map[string]Ticket)}
	go s.gcLoop()
	return s
}

func (s *Store) Issue(serverUUID string, userID uint64, ttl time.Duration) (string, time.Time, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", time.Time{}, err
	}
	id := hex.EncodeToString(b)
	exp := time.Now().UTC().Add(ttl)
	s.mu.Lock()
	s.data[id] = Ticket{ServerUUID: serverUUID, UserID: userID, ExpiresAt: exp}
	s.mu.Unlock()
	return id, exp, nil
}

func (s *Store) Consume(id, serverUUID string) (Ticket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.data[id]
	if !ok {
		return Ticket{}, false
	}
	delete(s.data, id)
	if time.Now().UTC().After(t.ExpiresAt) || t.ServerUUID != serverUUID {
		return Ticket{}, false
	}
	return t, true
}

func (s *Store) gcLoop() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for range t.C {
		now := time.Now().UTC()
		s.mu.Lock()
		for k, v := range s.data {
			if now.After(v.ExpiresAt) {
				delete(s.data, k)
			}
		}
		s.mu.Unlock()
	}
}
