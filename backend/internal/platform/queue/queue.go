package queue

// QueueProcessor is a placeholder for Redis-backed jobs used by
// DNS verification polling, webhook fan-out, and quota aggregation.
type QueueProcessor struct{}

func New() *QueueProcessor {
	return &QueueProcessor{}
}
