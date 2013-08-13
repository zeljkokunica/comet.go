package comet

import (
	"strings"
	"time"
	"github.com/zeljkokunica/comet.go/l"
	"fmt"
	"io"
	"crypto/rand"
)

const (
	Subscribe = 1
	Unsubscribe = 2
	SubscribeToChannels = 3
	UnsubscribeFromChannels = 4
	CleanupSubscribers = 5
)

type HubSubscriberRequest struct {
	command int
	subscriberId string
	channels []string
	responseListener chan<- Subscriber
}

type Hub struct {
	subscribers map[string]*Subscriber
	repository *HubRepository
	subscriberFeedListener chan ChannelDataOperation
	subscriberCommandListener chan HubSubscriberRequest
}

func NewHub() *Hub {
	var hub = new(Hub)
	hub.subscribers = make(map[string]*Subscriber)
	hub.repository = NewHubRepository()
	hub.subscriberCommandListener = make(chan HubSubscriberRequest, expectedMaxSubscribers)
	hub.subscriberFeedListener = make(chan ChannelDataOperation, expectedMaxSubscribers)
	go hub.subscribersProcess()
	go hub.refreshStatusProcess()
	return hub
}

/**
* handles adding, removing, cleaning and feeding subscribers
*/
func (h *Hub) subscribersProcess() {
	l.I("subscribers process - starting") 
	for {
		select {
			case newData := <- h.subscriberFeedListener:
				l.If("subscribers process - newData", newData)
				for _, subscriber := range(h.subscribers) {
					if subscriber.channels[newData.channelData.ChannelName] != nil {
						select {
							case subscriber.commandListener <- SubscriberControlCommand{SubscriberFeed, CreateSingleFeedCommad(string(newData.operation), newData.channelData.ChannelName, newData.channelData.Data, newData.channelData.DataVersion)}:
							default:
								l.Wf("subscribers process - did not deliver new data to %s - queue full", subscriber.id) 
						}
					}
				}
			case subscriberCommand := <- h.subscriberCommandListener:
				switch subscriberCommand.command {
					case Subscribe: 
						l.I("subscribers process - subscribe")
						newSubscriber := h.createSubscriber(subscriberCommand.channels)
						subscriberCommand.responseListener <- newSubscriber
					case Unsubscribe: 
						l.If("subscribers process - %s - unsubscribe", subscriberCommand.subscriberId)
						h.deleteSubscriber(subscriberCommand.subscriberId)
						subscriberCommand.responseListener <- Subscriber{}		
					case SubscribeToChannels:
						l.If("subscribers process - %s - subscribe to channels", subscriberCommand.subscriberId, subscriberCommand.channels)
						if subscriber, found := h.subscribers[subscriberCommand.subscriberId]; found == true {
							h.addChannelsToSubscriber(subscriberCommand.channels, subscriber)
							subscriberCommand.responseListener <- *subscriber
						} else {
							subscriberCommand.responseListener <- Subscriber{}
						}
					case UnsubscribeFromChannels:
						l.If("subscribers process - %s - unsubscribe from channels", subscriberCommand.subscriberId, subscriberCommand.channels)
						if subscriber, found := h.subscribers[subscriberCommand.subscriberId]; found == true {
							h.removeChannelsFromSubscriber(subscriberCommand.channels, subscriber)
							subscriberCommand.responseListener <- *subscriber
						} else {
							subscriberCommand.responseListener <- Subscriber{}
						}
					case CleanupSubscribers: 
						var cleanupStartTime = time.Now().Unix()
						for id, subscriber := range(h.subscribers) {
							if cleanupStartTime - subscriber.lastRequest > secondsToKeepSubscriberAlive {
								l.If("subscribers process - found dead subscriber %s", id)
								h.deleteSubscriber(id)
							}
						}
				}
			case <- time.After(30 * time.Second):
    		l.If("subscribers process - alive")
		}
	}
	l.I("subscribers process - stopped")
}

func newUUID() (string, error) {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}

func (h *Hub) createSubscriber(channels []string) Subscriber{
	l.If("subscriber creating with channels %s", channels)
	var id, err = newUUID()
	if err != nil {
		l.Ef("Error creation new uuid %s", err.Error())
	}
	
	subscriber := Subscriber{
		id: id, 
		commandListener:  make(chan SubscriberControlCommand, 10), 
		feedListener: make(chan SubscriberFeedCommand, subscriberQueueSize),
		channels: make(map[string]*SubscriberChannel),
	}
	channels = append(channels, fmt.Sprintf("private_%s", id))
	h.addChannelsToSubscriber(channels, &subscriber)
	subscriber.lastRequest = time.Now().Unix()
	go subscriber.subscriberCommandProcess(h)
	h.subscribers[id] = &subscriber
	l.If("subscriber %s created with channels %s", id, channels)
	return subscriber
}

func (h *Hub) addChannelsToSubscriber(channels []string, s *Subscriber) {
	for i := range(channels) {
		var channelName = channels[i]
		if len(strings.Trim(channelName, "")) == 0 {
			continue
		}
		
		var channel = new (SubscriberChannel)
		channel.channelName = channelName
		channel.dataVersion = -1
		s.channels[channelName] = channel
		var responseReceiver = make(chan Channel)
		channelFeedsListener := make(chan HubRepositoryChannelFeeds)
		defer close(channelFeedsListener)
		h.repository.getChannelListener <- HubRepositoryChannelGetCommand{channelName: channelName, resultListener: channelFeedsListener}
		var channelFeeds = <- channelFeedsListener
		channelFeeds.getDataListener <- ChannelDataRequestCommand{channelName: channelName, lastDataVersion: -1, responseReceiver: responseReceiver}
		var data = <-responseReceiver
		var commands = make([]SubscriberResponseCommand,  len(data.Updates) + 1)
		commands[0] = SubscriberResponseCommand{DataCreate, data.ChannelName, data.Data, data.DataVersion}
		for i := 0; i < len(data.Updates); i++ {
			commands[ i + 1] = SubscriberResponseCommand{DataUpdate, data.Updates[i].ChannelName, data.Updates[i].Data, data.Updates[i].DataVersion}
		}
		s.feedListener <- SubscriberFeedCommand{data: commands}
		
		close(responseReceiver)
	}
}

func (h *Hub) removeChannelsFromSubscriber(channels []string, s *Subscriber) {
	for i := range(channels) {
		var channelName = channels[i]
		if len(strings.Trim(channelName, "")) == 0 {
			continue
		}
		if _, found := s.channels[channelName]; found == true {
			delete(s.channels, channelName)
		}
	}
}

func (h *Hub) deleteSubscriber(id string) {
	if subscriber, found := h.subscribers[id]; found == true {
		subscriber.commandListener <- SubscriberControlCommand{command: SubscriberStop}
		delete(h.subscribers, id)
		l.Df("subscriber deleted %s", id)
	}
}
