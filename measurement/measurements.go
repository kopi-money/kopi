package measurement

import (
	"fmt"
	"sort"
	"time"
)

type Measurement struct {
	nanos map[string]int64
	count map[string]int64

	running map[string]time.Time
}

func NewMeasurements() *Measurement {
	return &Measurement{
		nanos:   make(map[string]int64),
		count:   make(map[string]int64),
		running: make(map[string]time.Time),
	}
}

func (m *Measurement) Print(printer func(string)) {
	m.PrintSkip(printer, nil)
}

func (m *Measurement) PrintSkip(printer func(string), skip []string) {
	keys := m.GetKeys()
	for _, key := range keys {
		if contains(skip, key) {
			continue
		}

		millis, avg := m.millis(key)
		printer(fmt.Sprintf("%v: %.4fms (%v, %v)", key, millis, avg, m.count[key]))
	}
}

func (m *Measurement) millis(key string) (float64, float64) {
	millis := float64(m.nanos[key]) / 1_000_000
	avg := millis / float64(m.count[key])
	return millis, avg
}

func (m *Measurement) GetKeys() (keys []string) {
	for key := range m.nanos {
		keys = append(keys, key)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	return
}

func (m *Measurement) Start(key string) {
	m.running[key] = time.Now()
}

func (m *Measurement) End(key string) {
	m.count[key] += 1
	m.nanos[key] += time.Since(m.running[key]).Nanoseconds()
}

func contains(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}

	return false
}
