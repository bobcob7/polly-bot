package internal

func NewMemoryHistory(maxLen int) *MemoryHistory {
	return &MemoryHistory{
		maxLen:  maxLen,
		records: make([]string, 0, maxLen),
	}
}

type MemoryHistory struct {
	maxLen  int
	records []string
}

// add will check if a record was already added and adds it if it's not.
func (m *MemoryHistory) Add(s string) bool {
	for _, v := range m.records {
		if v == s {
			return false
		}
	}
	m.records = append(m.records, s)
	return true
}

// Increment all record TTLs and delete them if they are too old.
func (m *MemoryHistory) Cleanup() {
	if len(m.records) > m.maxLen {
		m.records = m.records[:m.maxLen]
	}
}
