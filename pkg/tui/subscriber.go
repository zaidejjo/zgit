package tui

import (
	"context"
	"sync"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/git"
	"github.com/zaidejjo/zgit/pkg/core/models"
)

// teaMsg is the generic message type sent from subscriber to Bubble Tea.
type teaMsg struct {
	view int         // target view index
	data interface{} // typed payload (statusEvent, logEvent, etc.)
}

// Event types sent from subscriber to Bubble Tea event loop.
// Each carries typed data for a specific view.

type statusEvent struct {
	status *models.Status
	err    error
}

type logEvent struct {
	commits []*models.Commit
	err     error
}

type branchEvent struct {
	branches []*models.Branch
	err      error
}

type refreshEvent struct{}

// Subscriber bridges git engine events to Bubble Tea messages.
// Runs a background poller on a ticker — never blocks the UI.
type Subscriber struct {
	git    *git.NativeExec
	msgs   chan teaMsg
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	tick   *time.Ticker
}

// NewSubscriber creates a subscriber that polls git state periodically.
func NewSubscriber(gitExec *git.NativeExec, msgs chan teaMsg) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())
	return &Subscriber{
		git:    gitExec,
		msgs:   msgs,
		ctx:    ctx,
		cancel: cancel,
		tick:   time.NewTicker(5 * time.Second),
	}
}

// Start begins background polling.
func (s *Subscriber) Start() {
	s.wg.Add(1)
	go s.pollLoop()
}

// Stop terminates the background poller and waits for cleanup.
func (s *Subscriber) Stop() {
	s.tick.Stop()
	s.cancel()
	s.wg.Wait()
}

// Refresh triggers an immediate poll from the UI.
func (s *Subscriber) Refresh() {
	s.send(0, refreshEvent{})
}

func (s *Subscriber) pollLoop() {
	defer s.wg.Done()

	// Initial load
	s.fetchAll()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-s.tick.C:
			s.fetchAll()
		}
	}
}

func (s *Subscriber) fetchAll() {
	// Fetch all state in parallel-ish: sequential calls with short timeouts

	// Status
	s.fetchStatus()

	// Log
	s.fetchLog()

	// Branches
	s.fetchBranches()
}

func (s *Subscriber) fetchStatus() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	status, err := s.git.Status(ctx)
	if err != nil {
		s.send(0, statusEvent{err: err})
		return
	}
	s.send(0, statusEvent{status: status})
}

func (s *Subscriber) fetchLog() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	commits, err := s.git.Log(ctx, git.LogOptions{Count: 50})
	if err != nil {
		s.send(1, logEvent{err: err})
		return
	}
	s.send(1, logEvent{commits: commits})
}

func (s *Subscriber) fetchBranches() {
	ctx, cancel := context.WithTimeout(s.ctx, 10*time.Second)
	defer cancel()

	branches, err := s.git.Branches(ctx)
	if err != nil {
		s.send(2, branchEvent{err: err})
		return
	}
	s.send(2, branchEvent{branches: branches})
}

// send does a non-blocking send to the msg channel.
func (s *Subscriber) send(view int, data interface{}) {
	select {
	case s.msgs <- teaMsg{view: view, data: data}:
	default:
		// Channel full — UI will catch up on next tick
	}
}
