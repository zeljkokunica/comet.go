package comet_client

import (
	"net/http"
	"io/ioutil"
	"encoding/json"
	"time"
	"fmt"
)
const (
	ClientCommandAddChannel = 1
	ClientCommandRemoveChannel = 2
	ClientCommandClose = 3
)

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
	
type cometClientCommand struct {
	command int
	data []string
}

type CometClient struct {
	serverIp string
	channels []string
	subscriberId string
	OnDataFeed chan SubscriberResponseCommand
	clientCommand chan cometClientCommand
}

func NewCometClient(serverIp string, channels []string) CometClient {
	var client = CometClient{serverIp, channels, "", make (chan SubscriberResponseCommand, 10), make(chan cometClientCommand, 10)}
	go client.cometClientProcess()
	return client
} 

func (c *CometClient) channelsToParam() string {
	result := ""
	for i:=0; i < len(c.channels); i++ {
		if result == "" {
			result = c.channels[i]
		} else {
			result += "," + c.channels[i]
		}
			
	}
	return result
}

func FeedData(serverIp string, channel string, command string, data string) {
	for ok := false; !ok; {
		_, err := http.Get("http://" + serverIp + "/" + command + "?channel=" + channel + "&data=" + data)
		ok = err == nil
		if err != nil {
			fmt.Printf("\nFeedData got error %s", err.Error())
		}
	}
}

func (c *CometClient) subscribe() {
	var id string = ""
	var err error = nil
	var resp *http.Response
	for ; id == ""; {
		resp, err = http.Get("http://" + c.serverIp + "/subscribe?channels=" + c.channelsToParam())
		if err == nil {
//			fmt.Printf("\nsubscribe got response %s", id)
			var idBytes []byte
			idBytes, err = ioutil.ReadAll(resp.Body);
			resp.Body.Close()	
			if err == nil {
				id = string(idBytes)
				break
			} else {
				fmt.Printf("subscribe parse response failed %s", err.Error())
			}
		} else {
			fmt.Printf("subscribe failed %s", err.Error())
		}
		<-time.After(time.Second)
	}
	c.subscriberId = id
	fmt.Printf("\nsubscribed %s", id)
}

func (c *CometClient) getData() {
	var err error = nil
	var resp *http.Response
	for {
		if c.subscriberId == "" {
			c.subscribe()
		}
		resp, err = http.Get("http://" + c.serverIp + "/data?id=" + c.subscriberId)
		if err != nil {
			continue
		}
		var dataBytes []byte
		dataBytes, err = ioutil.ReadAll(resp.Body);
		if err != nil {
			continue
		}
		resp.Body.Close()
		var result = new(SubscriberResponse)
		err = json.Unmarshal(dataBytes, result) 
		if err != nil {
			continue
		}
		switch(result.Status) {
			case 0:
//				fmt.Printf("\nno new data %s", c.subscriberId)
				continue
			case 1:
//				fmt.Printf("\ngot new data %d", len(result.Commands))
				for i := 0; i < len(result.Commands); i++ {
					c.OnDataFeed <- result.Commands[i]
				}
			case -1:
				fmt.Printf("\nlost subscription %s", c.subscriberId)
				c.subscriberId = ""
			default:
				fmt.Printf("\nunknown result %d", result.Status)
				c.subscriberId = ""
							
		}
	}
}


func (c *CometClient) cometClientProcess() {
	for {
		if c.subscriberId == "" {
			c.subscribe()
			continue
		}
//		select {
//			case newCommand := <- c.clientCommand:
//				if newCommand.command == ClientCommandFeedData {
//					c.feedData(newCommand.data[0], newCommand.data[1], newCommand.data[2])
//					continue
//				}
//			default:
//		}
		c.getData()
		// get data
	}
}

