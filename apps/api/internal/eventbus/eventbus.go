// Package eventbus 定义领域事件与事务性 Outbox 的契约。
//
// 选题要求"事件驱动架构（EDA）"。教学场景不引入 Kafka/RabbitMQ，
// 采用 Outbox 表 + 轮询 Worker 的最小实现：
//
//  1. 业务事务内：写业务表 + INSERT event_outbox（同一事务，保证原子性）
//  2. Worker 轮询未投递事件，调用处理器，成功后标记 dispatched_at
//  3. 处理失败重试，超过次数进入死信（dead_letter_at），可人工重放
package eventbus

import "time"

// 事件主题常量。命名采用"聚合 + 过去式动词"，表示已经发生的事实。
const (
	TopicTripVerified    = "TripVerified"    // 行程核验通过，已算碳减排
	TopicPointsGranted   = "PointsGranted"   // 积分已发放
	TopicRedeemOrderPaid = "RedeemOrderPaid" // 兑换订单已支付（待发货）
	TopicOrderShipped    = "OrderShipped"    // 订单已发货
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
