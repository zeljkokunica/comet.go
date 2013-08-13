package comet

import (
	"time"
	"encoding/json"
)

type ChannelData struct {
	ChannelName string `json:"channelName"`
	DataVersion int64 `json:"dataVersion"`
	Data string `json:"data"`
	DataTime time.Time `json:"dataTime"`
}

func (cd ChannelData) Copy() ChannelData {
	return ChannelData{cd.ChannelName, cd.DataVersion, cd.Data, cd.DataTime}
}

type ChannelDataOperation struct {
	operation DataOperation
	channelData ChannelData
}

/**
* Represents channel and its data
*/
type Channel struct {
	ChannelName string `json:"channelName"`
	DataVersion int64 `json:"dataVersion"`
	Data string `json:"data"`
	DataTime time.Time `json:"dataTime"`
	Updates []ChannelData `json:"updates"`
}

func (c Channel) Copy() Channel {
	updates := make([]ChannelData, len(c.Updates))
	copy(updates, c.Updates)
	return Channel{c.ChannelName, c.DataVersion, c.Data, c.DataTime, updates}
}

func (channel *Channel) GetLastVersion() int64 {
	if len(channel.Updates) > 0 {
		return channel.Updates[len(channel.Updates) - 1].DataVersion
	}	
	return channel.DataVersion
}

func (channel Channel) ToJson() string {
	result, err := json.Marshal(channel)
	if (err == nil) {	
		return string(result)
	}
	return ""
}

func (channel *Channel) addNewData(command DataOperation, data string, responseListener chan ChannelDataOperation) {
	var response ChannelData
	if command == DataClear {
		channel.DataVersion = 0
		channel.Data = "" 
		channel.DataTime = time.Now()
		channel.Updates = make([]ChannelData, 0)
		response = ChannelData{ChannelName: channel.ChannelName, DataVersion: channel.GetLastVersion(), Data: channel.Data, DataTime: channel.DataTime}
	}	else if command == DataCreate {
		channel.DataVersion = channel.GetLastVersion() + 1
		channel.Data = data 
		channel.DataTime = time.Now()
		channel.Updates = make([]ChannelData, 0)
		response = ChannelData{ChannelName: channel.ChannelName, DataVersion: channel.GetLastVersion(), Data: channel.Data, DataTime: channel.DataTime}
	} else if command == DataUpdate {
			var version = ChannelData{ChannelName: channel.ChannelName, DataVersion: channel.GetLastVersion() + 1, Data: data, DataTime: time.Now()}
			channel.Updates = append(channel.Updates, version)
			response = version
	}
	if (responseListener != nil) {
		responseListener <- ChannelDataOperation{operation: command, channelData: response}
	}
}

func (channel *Channel) addUpdateData(data string, responseListener chan Channel) {
	channel.DataVersion = channel.DataVersion + 1
	channel.Data = data 
	channel.DataTime = time.Now()
	if (responseListener != nil) {
		responseListener <- channel.Copy()
	}
}

/**
* Command to feed channel a new data
*/
type ChannelDataInputCommand struct {
	Command string
	ChannelName string
	DataVersion int64
	Data string
	responseListener chan ChannelDataOperation
	
}

func (c ChannelDataInputCommand) ToJson() string {
	result, err := json.Marshal(c)
	if (err == nil) {	
		return string(result)
	}
	return ""
}

/** 
* Command to request data from Channel
*/
type ChannelDataRequestCommand struct {
	channelName string
	lastDataVersion int64
	responseReceiver chan Channel
}