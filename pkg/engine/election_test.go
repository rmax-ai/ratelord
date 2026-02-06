package engine

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rmax-ai/ratelord/pkg/store"
)

// MockLeaseStore is a mock implementation of store.LeaseStore for testing.
type MockLeaseStore struct {
	mu sync.Mutex

	acquireResult bool
	acquireError  error
	renewError    error
	releaseError  error
	getResult     *store.Lease
	getError      error

	acquireCalled bool
	renewCalled   bool
	releaseCalled bool
	getCalled     bool
}

func (m *MockLeaseStore) Acquire(ctx context.Context, name, holderID string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acquireCalled = true
	return m.acquireResult, m.acquireError
}

func (m *MockLeaseStore) Renew(ctx context.Context, name, holderID string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renewCalled = true
	return m.renewError
}

func (m *MockLeaseStore) Release(ctx context.Context, name, holderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.releaseCalled = true
	return m.releaseError
}

func (m *MockLeaseStore) Get(ctx context.Context, name string) (*store.Lease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalled = true
	return m.getResult, m.getError
}

func (m *MockLeaseStore) resetCalls() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.acquireCalled = false
	m.renewCalled = false
	m.releaseCalled = false
	m.getCalled = false
}

func TestElectionManager_Promotion(t *testing.T) {
	mockStore := &MockLeaseStore{
		acquireResult: true,
		acquireError:  nil,
	}

	promoteCh := make(chan bool, 1)
	demoteCh := make(chan bool, 1)

	em := NewElectionManager(
		mockStore,
		"test-holder",
		"test-lease",
		50*time.Millisecond,
		func() { promoteCh <- true },
		func() { demoteCh <- true },
	)

	ctx, cancel := context.WithCancel(context.Background())
	em.Start(ctx)

	// Wait for promotion
	select {
	case <-promoteCh:
		// Good
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OnPromote not called")
	}

	if !em.IsLeader() {
		t.Error("Expected to be leader after promotion")
	}

	em.Stop(ctx)
	cancel()

	select {
	case <-demoteCh:
		t.Fatal("OnDemote should not be called on stop if leader")
	default:
		// Good, no demotion
	}
}

func TestElectionManager_Demotion(t *testing.T) {
	mockStore := &MockLeaseStore{
		acquireResult: true,
		acquireError:  nil,
		renewError:    errors.New("renew failed"),
	}

	promoteCh := make(chan bool, 1)
	demoteCh := make(chan bool, 1)

	em := NewElectionManager(
		mockStore,
		"test-holder",
		"test-lease",
		50*time.Millisecond,
		func() { promoteCh <- true },
		func() { demoteCh <- true },
	)

	ctx, cancel := context.WithCancel(context.Background())
	em.Start(ctx)

	// Wait for promotion
	select {
	case <-promoteCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("OnPromote not called")
	}

	// Wait for demotion (renew will fail)
	select {
	case <-demoteCh:
		// Good
	case <-time.After(200 * time.Millisecond):
		t.Fatal("OnDemote not called after renew failure")
	}

	if em.IsLeader() {
		t.Error("Expected not to be leader after demotion")
	}

	em.Stop(ctx)
	cancel()
}

func TestElectionManager_Renewal(t *testing.T) {
	mockStore := &MockLeaseStore{
		acquireResult: true,
		acquireError:  nil,
		renewError:    nil,
	}

	em := NewElectionManager(
		mockStore,
		"test-holder",
		"test-lease",
		50*time.Millisecond,
		func() {},
		func() {},
	)

	ctx, cancel := context.WithCancel(context.Background())
	em.Start(ctx)

	// Wait a bit for initial acquisition and some renewals
	time.Sleep(150 * time.Millisecond)

	em.Stop(ctx)
	cancel()

	mockStore.mu.Lock()
	renewCalled := mockStore.renewCalled
	mockStore.mu.Unlock()

	if !renewCalled {
		t.Error("Renew should have been called periodically")
	}
}

func TestElectionManager_IsLeader(t *testing.T) {
	mockStore := &MockLeaseStore{
		acquireResult: false, // Initially not leader
		acquireError:  nil,
	}

	em := NewElectionManager(
		mockStore,
		"test-holder",
		"test-lease",
		50*time.Millisecond,
		func() {},
		func() {},
	)

	if em.IsLeader() {
		t.Error("Should not be leader initially")
	}

	// Change mock to allow acquisition
	mockStore.mu.Lock()
	mockStore.acquireResult = true
	mockStore.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	em.Start(ctx)

	// Wait for election
	time.Sleep(100 * time.Millisecond)

	if !em.IsLeader() {
		t.Error("Should be leader after successful acquisition")
	}

	em.Stop(ctx)
	cancel()
}
