package portalstate

import (
	"sort"
	"strings"
	"time"

	"everest.local/data-agent-policy-resolver/internal/contracts"
)

type DataPassportFilter struct {
	ExecutionID string
	FileID      string
	DataStoreID string
	Limit       int
}

func (s *Store) UpsertDataPassport(passport contracts.DataPassport) contracts.DataPassport {
	nowTime := time.Now().UTC()
	passport.PassportID = clean(firstNonEmpty(passport.PassportID, "passport-"+strings.TrimPrefix(contracts.StableHash(passport.ExecutionID, passport.Subject.URI, passport.Subject.Path), "sha256:")[:20]))
	passport = passport.WithDefaults(nowTime)

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, existing := range s.state.DataPassports {
		if existing.PassportID == passport.PassportID {
			passport.CreatedAt = firstNonEmpty(existing.CreatedAt, passport.CreatedAt)
			passport = passport.WithDefaults(nowTime)
			s.state.DataPassports[i] = passport
			s.sortPassportsLocked()
			s.saveLocked()
			return passport
		}
	}

	s.state.DataPassports = append(s.state.DataPassports, passport)
	s.sortPassportsLocked()
	if len(s.state.DataPassports) > 1000 {
		s.state.DataPassports = s.state.DataPassports[:1000]
	}
	s.saveLocked()
	return passport
}

func (s *Store) GetDataPassport(passportID string) (contracts.DataPassport, bool) {
	passportID = clean(passportID)
	if passportID == "" {
		return contracts.DataPassport{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, passport := range s.state.DataPassports {
		if passport.PassportID == passportID {
			return passport, true
		}
	}
	return contracts.DataPassport{}, false
}

func (s *Store) ListDataPassports(filter DataPassportFilter) []contracts.DataPassport {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	executionID := clean(filter.ExecutionID)
	fileID := clean(filter.FileID)
	dataStoreID := clean(filter.DataStoreID)

	out := make([]contracts.DataPassport, 0, minInt(len(s.state.DataPassports), limit))
	for _, passport := range s.state.DataPassports {
		if executionID != "" && passport.ExecutionID != executionID {
			continue
		}
		if dataStoreID != "" && passport.DataStoreID != dataStoreID {
			continue
		}
		if fileID != "" && passport.Subject.URI != fileID && passport.Subject.Path != fileID {
			continue
		}
		out = append(out, passport)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (s *Store) sortPassportsLocked() {
	sort.Slice(s.state.DataPassports, func(i, j int) bool {
		return s.state.DataPassports[i].UpdatedAt > s.state.DataPassports[j].UpdatedAt
	})
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
