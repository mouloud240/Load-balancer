package upstreams

import (
	"fmt"
	"net/url"
	"sync"
	"time"
)
const SUCCESS_THRESHOLD = 3
const FAILURE_THRESHOLD = 3
const HALF_OPEN_TIMEOUT = 10 * time.Second
const OPENTHRESHOLD = 30 * time.Second //Time elapsed since last try to flip from open to half open
type upstreamState int
const (
	open upstreamState = iota
	closed
	halfOpen
)
//TODO:need a member to manage how many requests we send through the half-open stream , since we don't want to sennd all to it 
//Maybe keeping a config of percentage and tracking skipped and currrent?
//Will be done after state trasnitions are working correctly
type Upstream struct{
	mux sync.Mutex //TO allow state trasnitions in concurrent environemnet
	 Server url.URL
	 State upstreamState
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
	if u.State == open && time.Since(u.lastChecked) > HALF_OPEN_TIMEOUT {
		fmt.Print("Transitioning upstream: ", u.Server.String(), " to half open state\n")
		u.State = halfOpen
		u.successCount = 0
	} 
}
func (u *Upstream) updateState() {
	switch u.State{
	case closed:
		if u.failureCount >= FAILURE_THRESHOLD {
			u.State = open
			u.failureCount = 0
		}

	case halfOpen:
		if u.successCount >= SUCCESS_THRESHOLD {
			u.State = closed
			u.successCount = 0
		}
	default:
		panic("Unknown state passed")

	}
	fmt.Print("Updated_state: ", u.State, " for upstream: ", u.Server.String(), "\n")
	
}

func NewUpstream(server url.URL) *Upstream {
	return &Upstream{
		Server: server,
		State: closed,
		lastChecked: time.Now(),
	}
}





