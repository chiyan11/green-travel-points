// Package eventbus 定义领域事件与事务性 Outbox 的契约。
//
// 选题要求"事件驱动架构（EDA）"。教学场景不引入 Kafka/RabbitMQ，
// 采用 Outbox 表 + 轮询 Worker 的最小实现：
//
//  1. 业务事务内：写业务表 + INSERT event_outbox（同一事务，保证原子性）
//  2. Worker 轮询未投递事件，调用处理器，成功后标记 dispatched_at
//  3. 处理失败重试，超过次数进入死信（dead_letter_at），可人工重放
package eventbus

import (
	"context"
	"log/slog"
	"time"
)

// 事件主题常量。命名采用"聚合 + 过去式动词"，表示已经发生的事实。
const (
	TopicTripVerified    = "TripVerified"    // 行程核验通过，已算碳减排
	TopicPointsGranted   = "PointsGranted"   // 积分已发放
	TopicRedeemOrderPaid = "RedeemOrderPaid" // 兑换订单已支付（待发货）
	TopicOrderShipped    = "OrderShipped"    // 订单已发货
	TopicOrderConfirmed  = "OrderConfirmed"  // 商户已确认订单（运营后台）
	TopicOrderPaid       = "OrderPaid"       // 兑换下单成功（待商户确认）
	TopicOrderCancelled  = "OrderCancelled"  // 订单取消，积分原路退回
	TopicOrderCompleted  = "OrderCompleted"  // 订单已完成核销（运营后台）
)

// Event 对应 event_outbox 表的一行。
type Event struct {
	ID            int64
	Topic         string
	AggregateType string
	AggregateID   string
	Payload       []byte
	Attempts      int
	AvailableAt   time.Time
	CreatedAt     time.Time
}

// Handler 消费单条事件；返回 error 时由 Worker 决定是否重试。
type Handler func(e Event) error

// Dispatcher 由 repo 实现；Worker 不依赖具体的 SQL 实现。
type Dispatcher interface {
	Dispatch(ctx context.Context, limit, maxAttempts int, handler Handler) (int, error)
}

// Worker 轮询并投递 Outbox 事件。处理成功后标记已投递，失败会退避重试或进入死信。
type Worker struct {
	dispatcher   Dispatcher
	handler      Handler
	logger       *slog.Logger
	pollInterval time.Duration
	batchSize    int
	maxAttempts  int
}

const (
	defaultPollInterval = 2 * time.Second
	defaultBatchSize    = 50
	defaultMaxAttempts  = 5
)

func NewWorker(dispatcher Dispatcher, handler Handler, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		dispatcher:   dispatcher,
		handler:      handler,
		logger:       logger,
		pollInterval: defaultPollInterval,
		batchSize:    defaultBatchSize,
		maxAttempts:  defaultMaxAttempts,
	}
}

// Run 直到 ctx 取消。首次立即执行，避免服务启动后无谓等待一个轮询周期。
func (w *Worker) Run(ctx context.Context) {
	w.ProcessOnce(ctx)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.ProcessOnce(ctx)
		}
	}
}

// ProcessOnce 投递当前一批待处理事件；错误只记录，下一轮继续尝试。
func (w *Worker) ProcessOnce(ctx context.Context) {
	if w.dispatcher == nil || w.handler == nil {
		return
	}
	processed, err := w.dispatcher.Dispatch(ctx, w.batchSize, w.maxAttempts, w.handler)
	if err != nil {
		w.logger.Error("outbox 投递失败", "err", err)
		return
	}
	if processed > 0 {
		w.logger.Debug("outbox 已处理事件", "count", processed)
	}
}
