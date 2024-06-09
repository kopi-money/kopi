package measurement

import (
	"cosmossdk.io/log"
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

func (m *Measurement) Print(logger log.Logger) {
	keys := m.GetKeys()
	for _, key := range keys {
		millis, avg := m.millis(key)
		logger.Info(fmt.Sprintf("%v: %.4fms (%v, %v)", key, millis, avg, m.count[key]))
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
