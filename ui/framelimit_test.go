package ui

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/kpfaulkner/guie/render"
)

// fakeLimiter is a frameLimiter on a clock the test moves, whose timers fire
// only when the test fires them.
type fakeLimiter struct {
	*frameLimiter
	t      time.Time
	timers []func()
	waits  []time.Duration
	asked  int
}

func newFakeLimiter(rate int) *fakeLimiter {
	f := &fakeLimiter{frameLimiter: newFrameLimiter(rate), t: time.Unix(1000, 0)}
	f.now = func() time.Time { return f.t }
	f.after = func(d time.Duration, fn func()) {
		f.waits = append(f.waits, d)
		f.timers = append(f.timers, fn)
	}
	return f
}

func (f *fakeLimiter) ask()                    { f.request(func() { f.asked++ }) }
func (f *fakeLimiter) advance(d time.Duration) { f.t = f.t.Add(d) }
func (f *fakeLimiter) fire() {
	fns := f.timers
	f.timers = nil
	for _, fn := range fns {
		fn()
	}
}

// A few requests together, like a key's echo and the prompt after it, are
// all asked for at once.
func TestFrameLimitLetsABurstThrough(t *testing.T) {
	f := newFakeLimiter(120)
	for range frameBurst {
		f.ask()
	}
	if f.asked != frameBurst || len(f.timers) != 0 {
		t.Errorf("asked %d frames and set %d timers, want %d and none", f.asked, len(f.timers), frameBurst)
	}
}

// Past the burst, requests wait for one timer, and every request before it
// fires is folded into the one frame it asks for.
func TestFrameLimitFoldsAStreamIntoOneTimer(t *testing.T) {
	f := newFakeLimiter(120)
	for range frameBurst {
		f.ask()
	}
	for range 50 {
		f.ask()
	}
	if f.asked != frameBurst {
		t.Fatalf("asked %d frames over the burst, want none until the timer", f.asked-frameBurst)
	}
	if len(f.timers) != 1 {
		t.Fatalf("set %d timers, want 1", len(f.timers))
	}
	if want := time.Second / 120; f.waits[0] < want-time.Millisecond || f.waits[0] > want+time.Millisecond {
		t.Errorf("timer set for %v, want about %v", f.waits[0], want)
	}
	f.advance(f.waits[0])
	f.fire()
	if f.asked != frameBurst+1 {
		t.Errorf("timer asked for %d frames, want 1", f.asked-frameBurst)
	}
}

// A request that keeps coming is held to the rate over time.
func TestFrameLimitHoldsASteadyRate(t *testing.T) {
	f := newFakeLimiter(100)
	// One request every millisecond for a second, the timers firing when due.
	var due time.Time
	for range 1000 {
		f.advance(time.Millisecond)
		if len(f.timers) > 0 && !f.t.Before(due) {
			f.fire()
		}
		set := len(f.waits)
		f.ask()
		if len(f.waits) > set { // this request set a timer
			due = f.t.Add(f.waits[set])
		}
	}
	if f.asked < 100 || f.asked > 100+frameBurst+1 {
		t.Errorf("asked %d frames in a second at 100 a second, want about 100", f.asked)
	}
}

// After a pause the burst is back: typing after a burst of output is not
// held behind it.
func TestFrameLimitRefillsAfterAPause(t *testing.T) {
	f := newFakeLimiter(120)
	for range frameBurst + 3 {
		f.ask()
	}
	f.advance(f.waits[0]) // the stream's last frame, when it is due
	f.fire()
	f.advance(time.Second)
	before := f.asked
	for range frameBurst {
		f.ask()
	}
	if f.asked-before != frameBurst || len(f.timers) != 0 {
		t.Errorf("after a pause asked %d frames at once, want %d", f.asked-before, frameBurst)
	}
}

// Rate 0 is no limit at all.
func TestFrameLimitZeroIsOff(t *testing.T) {
	f := newFakeLimiter(0)
	for range 100 {
		f.ask()
	}
	if f.asked != 100 || len(f.timers) != 0 {
		t.Errorf("asked %d frames with %d timers, want 100 and none", f.asked, len(f.timers))
	}
}

// Invalidate goes through the limit: the last request of a stream is still
// drawn, by the timer, once the rate allows.
func TestInvalidateIsLimitedButNotLost(t *testing.T) {
	d := &countingDriver{}
	a := NewApp(WithDriver(d), WithBackgroundFrameRate(1000))
	if err := a.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	for range 20 {
		a.Invalidate()
	}
	if got := d.requests.Load(); got > frameBurst {
		t.Fatalf("20 Invalidates asked for %d frames at once, want at most %d", got, frameBurst)
	}
	deadline := time.Now().Add(time.Second)
	for d.requests.Load() <= frameBurst && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond) // room for a wrong second timer to fire
	if got := d.requests.Load(); got != frameBurst+1 {
		t.Errorf("after the timer, %d frames asked for, want %d", got, frameBurst+1)
	}
}

// countingDriver counts RequestFrame calls, which the limiter's timer makes
// from its own goroutine.
type countingDriver struct{ requests atomic.Int64 }

func (d *countingDriver) Run(render.Config, render.Hooks) error { return nil }
func (d *countingDriver) SetContinuousFrames(bool)              {}
func (d *countingDriver) RequestFrame()                         { d.requests.Add(1) }

// raterDriver reports a refresh rate the test sets, and counts how often it is
// asked.
type raterDriver struct {
	pacingDriver
	hz    int
	asked int
}

func (d *raterDriver) RefreshRate() int { d.asked++; return d.hz }

// raterApp returns a running app on a raterDriver at hz, on a clock the test
// moves.
func raterApp(t *testing.T, hz int, opts ...AppOption) (*App, *raterDriver, *time.Time) {
	t.Helper()
	d := &raterDriver{hz: hz}
	a := NewApp(append([]AppOption{WithDriver(d)}, opts...)...)
	now := time.Unix(1000, 0)
	a.now = func() time.Time { return now }
	if err := a.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	return a, d, &now
}

// The limit follows the display: more than one frame a refresh is never seen.
func TestFrameLimitFollowsTheRefreshRate(t *testing.T) {
	a, _, _ := raterApp(t, 144)
	frame(t, a)
	if got := a.limit.rateNow(); got != 144 {
		t.Errorf("limit is %d frames a second on a 144 Hz display, want 144", got)
	}
}

// A display whose rate is not known gets the fixed default.
func TestFrameLimitUnknownRateUsesTheDefault(t *testing.T) {
	a, _, _ := raterApp(t, 0)
	frame(t, a)
	if got := a.limit.rateNow(); got != DefaultBackgroundFrameRate {
		t.Errorf("limit is %d with the rate unknown, want %d", got, DefaultBackgroundFrameRate)
	}
}

// A rate the app chose is kept whatever the display does.
func TestFrameLimitFixedRateIsKept(t *testing.T) {
	a, d, _ := raterApp(t, 144, WithBackgroundFrameRate(30))
	frame(t, a)
	if got := a.limit.rateNow(); got != 30 {
		t.Errorf("limit is %d, want the 30 the app set", got)
	}
	if d.asked != 0 {
		t.Errorf("asked the display's rate %d times with the rate fixed, want 0", d.asked)
	}
}

// The rate is asked for at most once a second however many frames run, and a
// change (the window moved to another display) is followed.
func TestFrameLimitChecksTheRateOnceASecond(t *testing.T) {
	a, d, now := raterApp(t, 60)
	for range 50 {
		frame(t, a)
		*now = now.Add(10 * time.Millisecond)
	}
	if d.asked != 1 {
		t.Fatalf("asked the rate %d times in half a second, want 1", d.asked)
	}
	d.hz = 120
	*now = now.Add(refreshCheckEvery)
	frame(t, a)
	if got := a.limit.rateNow(); got != 120 {
		t.Errorf("limit is %d after the display changed to 120 Hz, want 120", got)
	}
}

// setRate keeps the tokens earned at the old rate.
func TestFrameLimitSetRateKeepsTokens(t *testing.T) {
	f := newFakeLimiter(60)
	for range frameBurst {
		f.ask()
	}
	f.advance(time.Second/60 + time.Millisecond) // one token at 60 a second
	f.setRate(240)
	f.ask()
	if f.asked != frameBurst+1 || len(f.timers) != 0 {
		t.Errorf("after a rate change asked %d frames with %d timers, want %d and none", f.asked, len(f.timers), frameBurst+1)
	}
}
