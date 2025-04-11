package concurrent

import (
	"context"
	"sync"
	"time"

	"payflow/internal/domain"
	"payflow/pkg/logger"
)

type TransactionProcessor = func(transaction *domain.Transaction) error

type WorkerPool struct {
	numWorkers     int
	jobQueue       chan *domain.Transaction
	processor      TransactionProcessor
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	logger         logger.Logger
	started        bool
	mutex          sync.Mutex
	statsCollector *StatsCollector
}

func NewWorkerPool(numWorkers int, queueSize int, processor TransactionProcessor, logger logger.Logger) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		numWorkers:     numWorkers,
		jobQueue:       make(chan *domain.Transaction, queueSize),
		processor:      processor,
		ctx:            ctx,
		cancel:         cancel,
		logger:         logger,
		started:        false,
		statsCollector: NewStatsCollector(),
	}
}

func (wp *WorkerPool) Start() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()

	if wp.started {
		return
	}

	wp.logger.Info("İşçi havuzu başlatılıyor", map[string]interface{}{
		"num_workers": wp.numWorkers,
		"queue_size":  cap(wp.jobQueue),
	})

	for i := 0; i < wp.numWorkers; i++ {
		wp.wg.Add(1)
		workerID := i
		go func() {
			defer wp.wg.Done()
			wp.worker(workerID)
		}()
	}

	wp.started = true
}

func (wp *WorkerPool) Stop() {
	wp.mutex.Lock()
	if !wp.started {
		wp.mutex.Unlock()
		return
	}
	wp.started = false
	wp.mutex.Unlock()

	wp.logger.Info("İşçi havuzu durduruluyor", map[string]interface{}{})
	wp.cancel()
	close(wp.jobQueue)
	wp.wg.Wait()
}

func (wp *WorkerPool) Submit(transaction *domain.Transaction) bool {
	wp.mutex.Lock()
	if !wp.started {
		wp.mutex.Unlock()
		return false
	}
	wp.mutex.Unlock()

	// Non-blocking send
	select {
	case wp.jobQueue <- transaction:
		wp.statsCollector.IncrementSubmitted()
		wp.logger.Info("İşlem kuyruğa eklendi", map[string]interface{}{
			"transaction_id": transaction.ID,
			"type":           transaction.Type,
			"amount":         transaction.Amount,
		})
		return true
	default:
		wp.statsCollector.IncrementRejected()
		wp.logger.Warn("İşlem kuyruğu dolu, işlem reddedildi", map[string]interface{}{
			"transaction_id": transaction.ID,
		})
		return false
	}
}

func (wp *WorkerPool) worker(id int) {
	wp.logger.Info("İşçi başlatıldı", map[string]interface{}{"worker_id": id})

	for {
		select {
		case <-wp.ctx.Done():
			wp.logger.Info("İşçi durduruldu", map[string]interface{}{"worker_id": id})
			return
		case transaction, ok := <-wp.jobQueue:
			if !ok {
				wp.logger.Info("İş kuyruğu kapatıldı, işçi durduruluyor", map[string]interface{}{"worker_id": id})
				return
			}

			startTime := time.Now()
			wp.logger.Info("İşlem işleniyor", map[string]interface{}{
				"worker_id":      id,
				"transaction_id": transaction.ID,
				"type":           transaction.Type,
				"amount":         transaction.Amount,
			})

			err := wp.processor(transaction)

			processingTime := time.Since(startTime)

			if err != nil {
				wp.statsCollector.IncrementFailed()
				wp.logger.Error("İşlem başarısız oldu", map[string]interface{}{
					"worker_id":       id,
					"transaction_id":  transaction.ID,
					"error":           err.Error(),
					"processing_time": processingTime.String(),
				})
			} else {
				wp.statsCollector.IncrementCompleted()
				wp.statsCollector.RecordProcessingTime(processingTime)
				wp.logger.Info("İşlem başarıyla tamamlandı", map[string]interface{}{
					"worker_id":       id,
					"transaction_id":  transaction.ID,
					"processing_time": processingTime.String(),
				})
			}
		}
	}
}

func (wp *WorkerPool) GetStats() Stats {
	return wp.statsCollector.GetStats()
}

func (wp *WorkerPool) QueueLength() int {
	return len(wp.jobQueue)
}

func (wp *WorkerPool) QueueCapacity() int {
	return cap(wp.jobQueue)
}
