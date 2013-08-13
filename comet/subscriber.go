package comet

import (
	"time"
	"github.com/zeljkokunica/comet.go/l"
)

const (
	SubscriberFeed = 1
	SubscriberStop = 2
)
/** 
*	control messages for subscriberProcess
**/
type SubscriberControlCommand struct {
	command int
	SubscriberFeedCommand
}

/** 
*	control messages for serving data to a subscriber
**/
type SubscriberFeedCommand struct {
	data []SubscriberResponseCommand
}

func CreateSingleFeedCommad(command string, channel string, data string, dataVersion int64) SubscriberFeedCommand {
	result := SubscriberFeedCommand{}
	result.data = make([]SubscriberResponseCommand, 1)
	result.data[0] = SubscriberResponseCommand{command, channel, data, dataVersion}
	return result
} 

/**
* channel to which the subscriber is subscribed 
*/
type SubscriberChannel struct {
	channelName string
	dataVersion int64
}

type Subscriber struct {
	id string
	channels map[string]*SubscriberChannel
	commandListener chan SubscriberControlCommand
	feedListener chan SubscriberFeedCommand
	lastRequest int64
}

/**
* single command to send to a client as a response
*/
type SubscriberResponseCommand struct {
	Command string `json:"command"`
	Channel string `json:"channel"`
	Data string `json:"data"`
	DataVersion int64 `json:"version"`
}
/**
* complete response to client
*/
type SubscriberResponse struct {
	Status int  `json:"status"`
	Commands []SubscriberResponseCommand `json:"commands"`	 
}
/**
* process subscriber commands
* when feed command is received, data must be received within 30 seconds or subscriber will unsubscribe
*/
func (s *Subscriber) subscriberCommandProcess(h *Hub) {
	l.If("subscriber process - %s - start", s.id)
  defer l.If("subscriber process - %s - end", s.id)
  
	for {
		select {
			case subscriberCommand := <-s.commandListener:
				switch subscriberCommand.command {
					case SubscriberStop:
						l.If("subscriber process - %s - stop", s.id)
						return
					case SubscriberFeed:
						l.If("subscriber process - %s - feed", s.id)
						select {
							case s.feedListener <- subscriberCommand.SubscriberFeedCommand:
							case <- time.After(30 * time.Second):
								l.Wf("subscriber process - %s - feed timedout", s.id) 
								h.subscriberCommandListener <- HubSubscriberRequest{Unsubscribe, s.id, nil, nil}
								return
						}
				}
			case <- time.After(30 * time.Second):
    		l.If("subscriber process - %s - alive", s.id)
		}
	}
	
} 
