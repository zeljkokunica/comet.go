package comet

import (
	"time"
)

var expectedMaxSubscribers = 20000
var subscriberQueueSize = 500
var subscriberDeleteListenerQueueSize = 50
var refreshStatusPeriod = 30 * time.Second
var secondsToKeepSubscriberAlive int64 = 120
var restoreData = true
func init() {
}