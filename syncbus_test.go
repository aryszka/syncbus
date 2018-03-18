package syncbus

import (
	"errors"
	"testing"
	"time"
)

const testWaitTimeout = 120 * time.Millisecond

type testWait struct {
	n       int
	c       chan struct{}
	doneAll chan struct{}
}

var (
	token               = struct{}{}
	errUnexpectedlyDone = errors.New("unexpectedly done")
)

func newTestWait(n int) *testWait {
	return &testWait{
		n:       n,
		c:       make(chan struct{}, n),
		doneAll: make(chan struct{}),
	}
}

func (tw *testWait) wait() error {
	for {
		if tw.n <= 0 {
			close(tw.doneAll)
			return nil
		}

		select {
		case <-tw.c:
			tw.n--
		case <-time.After(testWaitTimeout):
			return ErrTimeout
		}
	}
}

func (tw *testWait) checkWaiting() error {
	select {
	case <-tw.doneAll:
		return errUnexpectedlyDone
	default:
		return nil
	}
}

func (tw testWait) done() {
	tw.c <- token
}

func TestNilWait(t *testing.T) {
	var bus *SyncBus
	if err := bus.Wait("test"); err != nil {
		t.Error(err)
	}
}

func TestNilSignal(t *testing.T) {
	var bus *SyncBus
	tw := newTestWait(1)
	go func() {
		bus.Signal("test")
		tw.done()
	}()

	if err := tw.wait(); err != nil {
		t.Error(err)
	}
}

func TestNilReset(t *testing.T) {
	var bus *SyncBus
	tw := newTestWait(1)
	go func() {
		bus.Signal("test")
		tw.done()
	}()

	bus.ResetSignals("test")
	if err := tw.wait(); err != nil {
		t.Error(err)
	}
}

func TestNilResetAll(t *testing.T) {
	var bus *SyncBus
	tw := newTestWait(1)
	go func() {
		bus.Signal("test")
		tw.done()
	}()

	bus.Reset()
	if err := tw.wait(); err != nil {
		t.Error(err)
	}
}

func TestNilClose(t *testing.T) {
	var bus *SyncBus
	bus.Close()
}

func TestEmptyWait(t *testing.T) {
	bus := New(120 * time.Millisecond)
	if err := bus.Wait(); err != nil {
		t.Error(err)
	}
}

func TestEmptySignal(t *testing.T) {
	bus := New(120 * time.Millisecond)
	tw := newTestWait(1)
	go func() {
		bus.Signal()
		tw.done()
	}()

	if err := tw.wait(); err != nil {
		t.Error(err)
	}
}

func TestEmptyRest(t *testing.T) {
	bus := New(120 * time.Millisecond)
	tw := newTestWait(1)
	go func() {
		bus.Signal()
		tw.done()
	}()

	bus.ResetSignals()
	if err := tw.wait(); err != nil {
		t.Error(err)
	}
}

func TestTimeout(t *testing.T) {
	bus := New(12 * time.Millisecond)
	defer bus.Close()

	if err := bus.Wait("test"); err != ErrTimeout {
		t.Error("failed to timeout")
	}
}

func TestTimeoutOneOfTwo(t *testing.T) {
	to := 12 * time.Millisecond
	bus := New(to)
	defer bus.Close()

	tw := newTestWait(2)

	go func() {
		if err := bus.Wait("test1"); err != ErrTimeout {
			t.Error("failed to timeout")
		}

		tw.done()
	}()

	go func() {
		time.Sleep(2 * to / 3)
		if err := bus.Wait("test2"); err != nil {
			t.Error("unexpected error:", err)
		}

		tw.done()
	}()

	time.Sleep(4 * to / 3)
	bus.Signal("test2")
	if err := tw.wait(); err != nil {
		t.Error(err)
	}
}

func TestSingleKeySignal(t *testing.T) {
	bus := New(120 * time.Millisecond)
	defer bus.Close()

	tw := newTestWait(2)

	go func() {
		if err := bus.Wait("test"); err != nil {
			t.Error(err)
		}

		tw.done()
	}()

	go func() {
		if err := bus.Wait("test"); err != nil {
			t.Error(err)
		}

		tw.done()
	}()

	bus.Signal("test")
	if err := tw.wait(); err != nil {
		t.Error(err)
	}
}

func TestMultiKeySignal(t *testing.T) {
	bus := New(120 * time.Millisecond)
	defer bus.Close()

	tw1 := newTestWait(1)
	go func() {
		if err := bus.Wait("foo", "bar"); err != nil {
			t.Error(err)
		}

		tw1.done()
	}()

	tw2 := newTestWait(1)
	go func() {
		if err := bus.Wait("bar", "baz"); err != nil {
			t.Error(err)
		}

		tw2.done()
	}()

	bus.Signal("foo")
	if err := tw1.checkWaiting(); err != nil {
		t.Error(err)
	}
	if err := tw2.checkWaiting(); err != nil {
		t.Error(err)
	}

	bus.Signal("bar")
	if err := tw1.wait(); err != nil {
		t.Error(err)
	}
	if err := tw2.checkWaiting(); err != nil {
		t.Error(err)
	}

	bus.Signal("baz")
	if err := tw2.wait(); err != nil {
		t.Error(err)
	}
}

func TestSignalBeforeWait(t *testing.T) {
	bus := New(120 * time.Millisecond)
	defer bus.Close()

	bus.Signal("foo")
	if err := bus.Wait("foo"); err != nil {
		t.Error(err)
	}
}

func TestReset(t *testing.T) {
	bus := New(12 * time.Millisecond)
	defer bus.Close()

	bus.Signal("foo")
	bus.ResetSignals("foo")
	if err := bus.Wait("foo"); err != ErrTimeout {
		t.Error("failed to timeout")
	}
}

func TestResetAll(t *testing.T) {
	bus := New(12 * time.Millisecond)
	defer bus.Close()

	bus.Signal("foo")
	bus.Signal("bar")
	bus.Signal("baz")

	bus.Reset()
	if err := bus.Wait("foo", "bar", "baz"); err != ErrTimeout {
		t.Error("failed to timeout")
	}
}
