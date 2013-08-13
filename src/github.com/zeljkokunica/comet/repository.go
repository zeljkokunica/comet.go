package comet

import (
	"github.com/zeljkokunica/l"
	"io/ioutil"
	"encoding/json"
	"log"
	"time"
	"strings"
)
type DataOperation string

const (
	DataClear = "clear"
	DataCreate = "create"
	DataUpdate = "update"
)

type HubRepositoryChannelFeeds struct {
	newDataListener chan ChannelDataInputCommand
	getDataListener chan ChannelDataRequestCommand
}

type HubRepositoryChannelGetCommand struct {
	channelName string
	resultListener chan HubRepositoryChannelFeeds
}

/***
* holds data state
*/
type HubRepository struct {
	channels map[string]*Channel
	newDataListener map[string]chan ChannelDataInputCommand
	getDataListener map[string]chan ChannelDataRequestCommand
	getChannelListener chan HubRepositoryChannelGetCommand
}

func NewHubRepository() *HubRepository {
	var repository = new (HubRepository)
	repository.channels = make(map[string]*Channel)
	repository.getChannelListener = make(chan HubRepositoryChannelGetCommand, expectedMaxSubscribers)
	repository.newDataListener = make(map[string]chan ChannelDataInputCommand, expectedMaxSubscribers)
	repository.getDataListener = make(map[string]chan ChannelDataRequestCommand, expectedMaxSubscribers)
	if restoreData {
		l.I("restoring channels")
		// read previous data
		var files, err = ioutil.ReadDir("data")
		if err == nil {
			for i := 0; i < len(files); i++ {
				var file = files[i]
				if !strings.HasSuffix(file.Name(), ".json") {
					continue
				}
				var data, err = ioutil.ReadFile("data/" + file.Name())
				if err == nil {
					l.If("restoring channel %s", file.Name())
					var channel = new (Channel)
					var err = json.Unmarshal(data, channel)
					if err == nil && strings.Trim(channel.ChannelName, "") != "" {
						repository.addCreatedChannel(channel)
					}
				}
			}
		}
		l.I("restoring channels completed.")
	}
	// start channel process
	go repository.channelProcess()

	return repository
}
/**
* create a dataProcess for a channel
*/
func (r *HubRepository) addCreatedChannel(channel *Channel) {
	if strings.Trim(channel.ChannelName, "") == "" {
		return
	}
	
	r.channels[channel.ChannelName] = channel
	r.newDataListener[channel.ChannelName] = make(chan ChannelDataInputCommand, 10)
	r.getDataListener[channel.ChannelName] = make(chan ChannelDataRequestCommand, expectedMaxSubscribers)
	go r.dataProcess(channel.ChannelName)
}

func (r *HubRepository) channelProcess() {
	l.I("channels process - start")
	for {
		var channelRequest = <- r.getChannelListener
		var channel = r.channels[channelRequest.channelName]
		if channel == nil {
			l.If("channels process - add channel %s", channelRequest.channelName);
			channel = new (Channel)
			channel.ChannelName = channelRequest.channelName
			channel.DataVersion = 0 
			channel.Data = "" 
			r.addCreatedChannel(channel)
		}
		l.If("channels process - served channel %s", channelRequest.channelName);
		channelRequest.resultListener <- HubRepositoryChannelFeeds{newDataListener: r.newDataListener[channel.ChannelName], getDataListener: r.getDataListener[channel.ChannelName]}
	}
	
	l.I("channels process - end")
}

/**
* receive and feed data from a channel
**/
func (r *HubRepository) dataProcess(channelName string) {
	l.If("data process - %s - start", channelName)
	for {
		select {
			// on new data
			case newData := <- r.newDataListener[channelName]: 
				l.If("data process - %s - new data: %s ", channelName, newData.ToJson())
				var channel = r.channels[newData.ChannelName]
				if channel == nil {
					channel = new (Channel)
					channel.ChannelName = newData.ChannelName
					channel.DataVersion = 0 
					channel.Data = "" 
					r.channels[newData.ChannelName] = channel
				}
				channel.addNewData(DataOperation(newData.Command), newData.Data, newData.responseListener)
				if !strings.HasPrefix("private", channel.ChannelName) {
					// persist data
					var js, _ = json.Marshal(channel)
					var err = ioutil.WriteFile("data/" + channel.ChannelName + ".json", js, 0644)
			    if err != nil { 
			    	log.Panic(err) 
			    }
			   }
		  // on data feed request
		  case newRequest := <- r.getDataListener[channelName]:
		  	l.If("data process - %s - get data", channelName)
				var channel = r.channels[newRequest.channelName]
				var result Channel
				if channel == nil {
					result = Channel{newRequest.channelName, 0, "", time.Now(), make([]ChannelData, 0)}
				} else {
					result = channel.Copy()
				}
				newRequest.responseReceiver <- result
		}	 
	}
	l.If("data process - %s - stop", channelName)
}

/**
* stores new data and informs sunscribers
*/
func (h *Hub) AddNewDataToChannel(command string, channel string, data string) {
	if strings.Trim(channel, "") == "" {
		return
	}
	// get channel (created if necessary)
	channelFeedsListener := make(chan HubRepositoryChannelFeeds)
	h.repository.getChannelListener <- HubRepositoryChannelGetCommand{channelName: channel, resultListener: channelFeedsListener}
	var channelFeeds = <- channelFeedsListener
	close(channelFeedsListener)
	// feed data to channel/repository
	var responseListener = make(chan ChannelDataOperation)
	channelFeeds.newDataListener <- ChannelDataInputCommand{Command: command, ChannelName: channel, Data: data, responseListener: responseListener}
	var dataResponse = <- responseListener
	// send data to subscribers
	h.subscriberFeedListener <- dataResponse
}

