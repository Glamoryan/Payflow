package concurrent

import (
	"sync"
	"sync/atomic"
	"time"
)

type Stats struct {
	Submitted      int64
	Completed      int64
	Failed         int64
	Rejected       int64
	AvgProcessTime time.Duration
}

type StatsCollector struct {
	submitted      int64
	completed      int64
	failed         int64
	rejected       int64
	totalProcTime  int64
	processedCount int64
	mutex          sync.RWMutex
}

func NewStatsCollector() *StatsCollector {
	return &StatsCollector{}
}

func (sc *StatsCollector) IncrementSubmitted() {
	atomic.AddInt64(&sc.submitted, 1)
}

func (sc *StatsCollector) IncrementCompleted() {
	atomic.AddInt64(&sc.completed, 1)
}

func (sc *StatsCollector) IncrementFailed() {
	atomic.AddInt64(&sc.failed, 1)
}

func (sc *StatsCollector) IncrementRejected() {
	atomic.AddInt64(&sc.rejected, 1)
}

func (sc *StatsCollector) RecordProcessingTime(d time.Duration) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.totalProcTime += d.Nanoseconds()
	sc.processedCount++
}

func (sc *StatsCollector) GetStats() Stats {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()

	stats := Stats{
		Submitted: atomic.LoadInt64(&sc.submitted),
		Completed: atomic.LoadInt64(&sc.completed),
		Failed:    atomic.LoadInt64(&sc.failed),
		Rejected:  atomic.LoadInt64(&sc.rejected),
	}

	if sc.processedCount > 0 {
		stats.AvgProcessTime = time.Duration(sc.totalProcTime / sc.processedCount)
	}

	return stats
}

func (sc *StatsCollector) Reset() {
	atomic.StoreInt64(&sc.submitted, 0)
	atomic.StoreInt64(&sc.completed, 0)
	atomic.StoreInt64(&sc.failed, 0)
	atomic.StoreInt64(&sc.rejected, 0)

	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.totalProcTime = 0
	sc.processedCount = 0
}
