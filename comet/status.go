package comet

import (
	"strconv"
	"runtime"
	"encoding/json"
	"time"
	"l"
)

type HubStatusSubscriber struct {
	Id string `json:"id"`
	Channels []string `json:"channels"`
}
type HubStatusChannel struct {
	ChannelName string `json:"channelName"`
	DataVersion int64 `json:"dataVersion"`
}
type HubStatus struct {
	Subscribers []HubStatusSubscriber `json:"subscribers"`
	Channels []HubStatusChannel `json:"channels"`
	Statistics map[string]string `json:"statistics"`
}

func createHubStatus(h *Hub) HubStatus{
	var subscribers = make([]HubStatusSubscriber, 0)
	
	for _, subscriber := range(h.subscribers) {
		subscribers = append(subscribers, HubStatusSubscriber{Id: subscriber.id})
	}
	
	var channels = make([]HubStatusChannel, 0)
	for _, channel := range(h.repository.channels) {
		channels = append(channels, HubStatusChannel{ChannelName: channel.ChannelName})
	}
	var stats = make(map[string]string, 0)
	stats["routines"] = strconv.Itoa(runtime.NumGoroutine())
	var result = HubStatus{Subscribers: subscribers, Channels: channels, Statistics: stats}
	return result 
}

func (h *Hub) refreshStatusProcess() {
	l.I("status process - start")
	defer l.I("status process - end")
	for {
			var data, _ = json.Marshal(createHubStatus(h))
			h.subscriberCommandListener <- HubSubscriberRequest{CleanupSubscribers, "", nil, nil}
			h.AddNewDataToChannel("create", "system", string(data))
			l.I("status process - checked system")
			time.Sleep(refreshStatusPeriod)
		}
}
