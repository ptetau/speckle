package server

import (
	"context"
	"sync"
)

// Submission is what a browser POSTs to /submit and what `speckle await`
// receives on /await.
type Submission struct {
	SpecVersion     int                       `json:"spec_version"`
	Decisions       map[string]DecisionAnswer `json:"decisions"`
	SectionComments map[string]string         `json:"section_comments,omitempty"`
	Notes           string                    `json:"notes,omitempty"`
}

type DecisionAnswer struct {
	Selected string `json:"selected"`
	Comment  string `json:"comment,omitempty"`
}

// queue holds at most one pending submission. Push replaces any prior
// unconsumed value (so a user submitting twice in a row only sends
// the latest); Await blocks until one arrives or ctx is cancelled.
type queue struct {
	mu      sync.Mutex
	pending chan Submission
}

func newQueue() *queue {
	return &queue{pending: make(chan Submission, 1)}
}

func (q *queue) Push(s Submission) {
	q.mu.Lock()
	defer q.mu.Unlock()
	select {
	case <-q.pending:
	default:
	}
	q.pending <- s
}

func (q *queue) Await(ctx context.Context) (Submission, bool) {
	select {
	case s := <-q.pending:
		return s, true
	case <-ctx.Done():
		return Submission{}, false
	}
}
