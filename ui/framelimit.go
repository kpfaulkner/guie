package ui

import (
	"sync"
	"time"
)

// DefaultBackgroundFrameRate is how many frames a second Invalidate (and so Do
// and Quit) may ask for once its burst is used up, when the refresh rate of the
// window's display is not known. When it is known (the driver is a
// render.RefreshRater) the limit follows it, as more than one frame a refresh
// is never seen. See WithBackgroundFrameRate.
const DefaultBackgroundFrameRate = 60

// refreshCheckEvery is how often the display's refresh rate is asked for, so a
// window moved to a display with another rate follows it.
const refreshCheckEvery = time.Second

// frameBurst is how many frames Invalidate asks for at once before the rate
// applies. A key press is drawn by the input frame, and its echo and a prompt
// redraw come back from another goroutine a few milliseconds later; the burst
// lets those through at once, so only a stream of requests is held to the rate.
const frameBurst = 4

// frameLimiter holds frames asked for from outside input to a steady rate.
//
// On-demand frames are presented as soon as they are asked for, with no vsync
// to pace them, so a background goroutine that asks after every change (a
// terminal reading a program's output, say) draws as fast as the app can
// draw: hundreds of frames a second, most of them never seen. Over the burst,
// a request starts one timer for when the next frame is due, and requests
// before then are folded into it. Only the latest state is drawn anyway, so
// nothing is lost but frames nobody would see.
//
// It is a token bucket: tokens refill at rate per second up to frameBurst, and
// each frame asked for takes one. Frames that come from input do not pass
// through it and take none.
type frameLimiter struct {
	mu      sync.Mutex
	rate    float64 // frames per second; 0 means no limit
	tokens  float64
	last    time.Time // when tokens was last refilled
	pending bool      // a timer is set and will ask for the frame

	now   func() time.Time
	after func(time.Duration, func())
}

func newFrameLimiter(rate int) *frameLimiter {
	return &frameLimiter{
		rate:   float64(max(rate, 0)),
		tokens: frameBurst,
		now:    time.Now,
		after:  func(d time.Duration, f func()) { time.AfterFunc(d, f) },
	}
}

// request calls ask now if a frame is allowed, or once when one next is, unless
// a call is already due then. It is safe from any goroutine.
func (l *frameLimiter) request(ask func()) {
	l.mu.Lock()
	if l.rate == 0 {
		l.mu.Unlock()
		ask()
		return
	}
	if l.pending {
		l.mu.Unlock()
		return
	}
	l.refill()
	if l.tokens >= 1 {
		l.tokens--
		l.mu.Unlock()
		ask()
		return
	}
	l.pending = true
	wait := time.Duration((1 - l.tokens) / l.rate * float64(time.Second))
	l.mu.Unlock()
	l.after(wait, func() {
		l.mu.Lock()
		l.pending = false
		l.refill()
		l.tokens = max(l.tokens-1, 0)
		l.mu.Unlock()
		ask()
	})
}

// setRate changes the rate, keeping the tokens earned so far. fps 0 or less
// means no limit.
func (l *frameLimiter) setRate(fps int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.rate != 0 {
		l.refill() // at the old rate, up to now
	}
	l.rate = float64(max(fps, 0))
}

// rateNow is the rate in frames per second, 0 for no limit.
func (l *frameLimiter) rateNow() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return int(l.rate)
}

// refill adds the tokens earned since the last refill. l.mu must be held.
func (l *frameLimiter) refill() {
	t := l.now()
	if !l.last.IsZero() {
		l.tokens = min(l.tokens+t.Sub(l.last).Seconds()*l.rate, frameBurst)
	}
	l.last = t
}
