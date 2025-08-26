package job

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryEventBus implements the EventBus interface using in-memory channels
type MemoryEventBus struct {
	mu             sync.RWMutex
	subscribers    map[string]*EventSubscription
	jobSubscribers map[string]map[string]*EventSubscription
	nextSubID      int
}

// EventSubscription represents an active event subscription
type EventSubscription struct {
	ID         string
	EventTypes []JobEventType
	Channel    chan *JobEvent
	JobID      string // Empty for global subscriptions
	Active     bool
}

// NewMemoryEventBus creates a new in-memory event bus
func NewMemoryEventBus() *MemoryEventBus {
	return &MemoryEventBus{
		subscribers:    make(map[string]*EventSubscription),
		jobSubscribers: make(map[string]map[string]*EventSubscription),
	}
}

// Publish publishes a job event to all relevant subscribers
func (eb *MemoryEventBus) Publish(ctx context.Context, event *JobEvent) error {
	if event == nil {
		return fmt.Errorf("event is nil")
	}

	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Send to global subscribers
	for _, sub := range eb.subscribers {
		if sub.Active && eb.matchesEventTypes(event.Type, sub.EventTypes) {
			select {
			case sub.Channel <- event:
				// Event sent successfully
			case <-ctx.Done():
				return ctx.Err()
			default:
				// Channel is full, skip this subscriber
				// In a production system, you might want to log this
			}
		}
	}

	// Send to job-specific subscribers
	if jobSubs, exists := eb.jobSubscribers[event.JobID]; exists {
		for _, sub := range jobSubs {
			if sub.Active && eb.matchesEventTypes(event.Type, sub.EventTypes) {
				select {
				case sub.Channel <- event:
					// Event sent successfully
				case <-ctx.Done():
					return ctx.Err()
				default:
					// Channel is full, skip this subscriber
				}
			}
		}
	}

	return nil
}

// Subscribe subscribes to job events of specified types
func (eb *MemoryEventBus) Subscribe(ctx context.Context, eventTypes []JobEventType) (<-chan *JobEvent, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subID := fmt.Sprintf("sub-%d", eb.nextSubID)
	eb.nextSubID++

	channel := make(chan *JobEvent, 100) // Buffered channel

	subscription := &EventSubscription{
		ID:         subID,
		EventTypes: eventTypes,
		Channel:    channel,
		Active:     true,
	}

	eb.subscribers[subID] = subscription

	return channel, nil
}

// SubscribeToJob subscribes to events for a specific job
func (eb *MemoryEventBus) SubscribeToJob(ctx context.Context, jobID string) (<-chan *JobEvent, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subID := fmt.Sprintf("job-sub-%s-%d", jobID, eb.nextSubID)
	eb.nextSubID++

	channel := make(chan *JobEvent, 100) // Buffered channel

	subscription := &EventSubscription{
		ID:         subID,
		EventTypes: nil, // All event types for this job
		Channel:    channel,
		JobID:      jobID,
		Active:     true,
	}

	// Initialize job subscribers map if needed
	if eb.jobSubscribers[jobID] == nil {
		eb.jobSubscribers[jobID] = make(map[string]*EventSubscription)
	}

	eb.jobSubscribers[jobID][subID] = subscription

	return channel, nil
}

// Unsubscribe removes a subscription
func (eb *MemoryEventBus) Unsubscribe(ctx context.Context, subscriptionID string) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Check global subscribers
	if sub, exists := eb.subscribers[subscriptionID]; exists {
		sub.Active = false
		close(sub.Channel)
		delete(eb.subscribers, subscriptionID)
		return nil
	}

	// Check job-specific subscribers
	for jobID, jobSubs := range eb.jobSubscribers {
		if sub, exists := jobSubs[subscriptionID]; exists {
			sub.Active = false
			close(sub.Channel)
			delete(jobSubs, subscriptionID)

			// Clean up empty job subscriber maps
			if len(jobSubs) == 0 {
				delete(eb.jobSubscribers, jobID)
			}

			return nil
		}
	}

	return fmt.Errorf("subscription not found: %s", subscriptionID)
}

// Close closes the event bus and all active subscriptions
func (eb *MemoryEventBus) Close() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Close all global subscriptions
	for subID, sub := range eb.subscribers {
		if sub.Active {
			sub.Active = false
			close(sub.Channel)
		}
		delete(eb.subscribers, subID)
	}

	// Close all job-specific subscriptions
	for jobID, jobSubs := range eb.jobSubscribers {
		for subID, sub := range jobSubs {
			if sub.Active {
				sub.Active = false
				close(sub.Channel)
			}
			delete(jobSubs, subID)
		}
		delete(eb.jobSubscribers, jobID)
	}

	return nil
}

// GetActiveSubscriptions returns the number of active subscriptions
func (eb *MemoryEventBus) GetActiveSubscriptions() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	count := len(eb.subscribers)

	for _, jobSubs := range eb.jobSubscribers {
		count += len(jobSubs)
	}

	return count
}

// PublishJobEvent is a convenience method for publishing job events
func (eb *MemoryEventBus) PublishJobEvent(ctx context.Context, jobID string, eventType JobEventType, status JobStatus, message string, data map[string]interface{}) error {
	event := &JobEvent{
		ID:        fmt.Sprintf("%s-%d", eventType, time.Now().UnixNano()),
		JobID:     jobID,
		Type:      eventType,
		Status:    status,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}

	return eb.Publish(ctx, event)
}

// Helper methods

// matchesEventTypes checks if an event type matches the subscription filter
func (eb *MemoryEventBus) matchesEventTypes(eventType JobEventType, filterTypes []JobEventType) bool {
	// If no filter types specified, match all events
	if len(filterTypes) == 0 {
		return true
	}

	for _, filterType := range filterTypes {
		if eventType == filterType {
			return true
		}
	}

	return false
}

// CleanupJobSubscriptions removes all subscriptions for a completed job
func (eb *MemoryEventBus) CleanupJobSubscriptions(jobID string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if jobSubs, exists := eb.jobSubscribers[jobID]; exists {
		for subID, sub := range jobSubs {
			if sub.Active {
				sub.Active = false
				close(sub.Channel)
			}
			delete(jobSubs, subID)
		}
		delete(eb.jobSubscribers, jobID)
	}
}

// EventBusStats represents event bus statistics
type EventBusStats struct {
	ActiveSubscriptions int `json:"active_subscriptions"`
	GlobalSubscriptions int `json:"global_subscriptions"`
	JobSubscriptions    int `json:"job_subscriptions"`
	EventsPublished     int `json:"events_published"`
}

// GetStats returns event bus statistics
func (eb *MemoryEventBus) GetStats() *EventBusStats {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	globalSubs := len(eb.subscribers)
	jobSubs := 0

	for _, jobSubMap := range eb.jobSubscribers {
		jobSubs += len(jobSubMap)
	}

	return &EventBusStats{
		ActiveSubscriptions: globalSubs + jobSubs,
		GlobalSubscriptions: globalSubs,
		JobSubscriptions:    jobSubs,
		// EventsPublished would be tracked separately
	}
}
