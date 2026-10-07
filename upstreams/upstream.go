package upstreams

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)
const SUCCESS_THRESHOLD = 3
const HALF_OPEN_SKIPPED_REQUEST=5 //This means that from 6 requests we give 1 to the half open channel 
//Formula of percentage perc=1-((H/H+1)/100)
const FAILURE_THRESHOLD = 3
const HALF_OPEN_TIMEOUT = 10 * time.Second
const OPENTHRESHOLD = 30 * time.Second //Time elapsed since last try to flip from open to half open
type upstreamState int
const (
	open upstreamState = iota
	closed
	halfOpen
)
type Upstream struct{
	mux sync.Mutex //To allow state trasnitions in concurrent environemnet
	 Server url.URL
	 State upstreamState
	 weight int //Weight for weighted round robin defaults to 1
	 requestsLeft int //The requests that is left before moving to the next one , starts with weight and goes down to zero
	 skippedCount int //For now this is a very basic member used to just send a few requests to the half_open 
	 failureCount int //For the closed state, so we can flip it to open if we have enough failures
	 successCount int //For half open state , we flip it to close if we have enough successes
	 lastChecked time.Time //For the open state , we flip it to half open if threshold of time has passed
}

func (u *Upstream) recordFailure() {
	u.mux.Lock()
	defer u.mux.Unlock()
	u.failureCount++
	u.lastChecked = time.Now()
	u.updateState()
}
func (u *Upstream) recordSuccess() {
	//Only record success if we are in half open state, otherwise we don't care about successes
	if u.State != halfOpen {
		return
	}
	u.mux.Lock()
	defer u.mux.Unlock()
	u.successCount++
	u.lastChecked = time.Now()
	u.updateState()
}
func (u *Upstream) TransitionToHalfOpen() {
	u.mux.Lock()
	defer u.mux.Unlock()
	u.transitionToHalfOpenLocked()
}

func (u *Upstream) transitionToHalfOpenLocked() {
	if u.State == open && time.Since(u.lastChecked) > HALF_OPEN_TIMEOUT {
		fmt.Print("Transitioning upstream: ", u.Server.String(), " to half open state\n")
		u.State = halfOpen
		u.successCount = 0
		u.skippedCount = 0
	}
}

// ClaimRequest checks whether this upstream can take a request and decrements quota if closed.
// It returns admitted=true if the request can be sent to this upstream.
// It returns exhausted=true if this upstream has exhausted its current quota (or completed a half-open probe),
// signaling the load balancer to advance the round-robin index.
func (u *Upstream) ClaimRequest() (admitted bool, exhausted bool) {
	u.mux.Lock()
	defer u.mux.Unlock()

	u.transitionToHalfOpenLocked()

	if u.State == open {
		return false, false
	}

	if u.State == halfOpen {
		if u.skippedCount >= HALF_OPEN_SKIPPED_REQUEST {
			u.skippedCount = 0
			return true, true
		}
		u.skippedCount++
		return false, false
	}

	// State is closed
	if u.requestsLeft <= 0 {
		u.requestsLeft = u.weight
	}
	u.requestsLeft--
	if u.requestsLeft == 0 {
		u.requestsLeft = u.weight
		return true, true
	}
	return true, false
}
func (u *Upstream) updateState() {
	switch u.State{
	case closed:
		if u.failureCount >= FAILURE_THRESHOLD {
			u.State = open
			u.skippedCount=0
			u.failureCount = 0
		}

	case halfOpen:
		if u.successCount >= SUCCESS_THRESHOLD {
			u.State = closed
			u.skippedCount=0
			u.successCount = 0
		}
	default:
		panic("Unknown state passed")

	}
	fmt.Print("Updated_state: ", u.State, " for upstream: ", u.Server.String(), "\n")
	
}

func NewUpstream(server url.URL, weight int) *Upstream {
	if weight <= 0 {
		weight = 1
	}
	return &Upstream{
		Server: server,
		State: closed,
		weight: weight,
		requestsLeft: weight,
		lastChecked: time.Now(),
	}
}





