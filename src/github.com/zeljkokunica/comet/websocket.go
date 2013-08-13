package comet

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"github.com/zeljkokunica/l"
	"fmt"
	"strings"
	"time"
)

type WebSocketCommand struct {
	RequestId int64 `json:"requestId"`
	Command string `json:"command"`
	Parameters map[string]interface{}  `json:"parameters"`
}

type WebSocketResponse struct {
	RequestId int64 `json:"requestId"`
	Data interface{} `json:"data"`
}

type WebSocketDataMediator struct {
  RequestId int64
	send chan WebSocketResponse
	command WebSocketCommand
}

func (m *WebSocketDataMediator) ReadParameter(parameterName string) string {
	return fmt.Sprint(m.command.Parameters[parameterName])
}

func (m *WebSocketDataMediator) WriteResponse(response interface{}, responseType string)  {
	m.send <- WebSocketResponse{RequestId: m.RequestId, Data: response}
}

type WebSocketHandler struct {
	ws *websocket.Conn
	hub *Hub
	send chan WebSocketResponse
	closeListener chan bool
	subscriber Subscriber
}

func (m *WebSocketHandler) reader() {
	l.I("wsreader process - starting")
	var subscriberId string = "anonymous"
	for {
  	var message string
    err := websocket.Message.Receive(m.ws, &message)
    if err != nil {
    	l.If("wsreader process - %s - error reading from socket: %s", subscriberId, err.Error())
    	break
    }
    l.If("wsreader process - read from socket %s", string(message))
    var command = new (WebSocketCommand)
    err = json.Unmarshal([]byte(message), command)
    if err != nil {
    	l.Ef("wsreader process - %s - error unmarshal commad %s", subscriberId, err.Error())
    	break
    }
    var mediator = WebSocketDataMediator{send: m.send, command: *command, RequestId: command.RequestId}
    // special commands - keep alive and subscribe
    if command.Command == "keepAlive" {
  	  var subscriber = m.hub.subscribers[m.subscriber.id]
			// subscriber not found
			if (subscriber == nil) {
				l.Wf("wsreader process - %s - timed out!", subscriberId)
    		break
    	} else {
    		subscriber.lastRequest = time.Now().Unix()
    	}	
    } else if command.Command == "subscribe" {
    	var channels = strings.Split(mediator.ReadParameter("channels"), ",")
			var responseListener = make(chan Subscriber)
			m.hub.subscriberCommandListener <- HubSubscriberRequest{command: Subscribe, channels: channels, responseListener: responseListener}
			m.subscriber = <- responseListener
			subscriberId = m.subscriber.id
			go m.writer()
			mediator.WriteResponse(map[string]interface{}{"command": "subscribe", "subscriberId": m.subscriber.id}, "json");
			close(responseListener)
    } else {
    	m.hub.route(command.Command, &mediator)
    }
  }
  m.closeListener <- true
  l.If("wsreader process - %s - stopped", subscriberId)
  m.ws.Close()
}

func (m *WebSocketHandler) writer() {
	l.If("wswriter process - %s - starting", m.subscriber.id)
	
	defer close(m.send);
	defer close(m.closeListener);
  defer m.ws.Close()
  defer l.If("wswriter process - %s - stopped", m.subscriber.id)
	
	for {
		select {
			case message := <- m.send:
				jsonData, _ := json.Marshal(message)
				l.If("wswriter process - %s - serve json %s", m.subscriber.id, jsonData)
  			err := websocket.Message.Send(m.ws, string(jsonData))
    		if err != nil {
    			l.Ef("wswriter process - %s - error writing to socket: %s", m.subscriber.id, err.Error())
    		}
			case command := <- m.subscriber.feedListener:
				var response = SubscriberResponse{Status: 1, Commands: command.data}
				jsonData, _ := json.Marshal(response)
				l.If("wswriter process - %s - serve data: %s", m.subscriber.id, string(jsonData))
				err := websocket.Message.Send(m.ws, string(jsonData))
    		if err != nil {
    			l.Ef("wswriter process - %s - error writing to socket: %s", m.subscriber.id, err.Error())
    		}
    	case <- m.closeListener:
    		l.If("wswriter process - %s - received close command", m.subscriber.id)
    		return
    	case <- time.After(30 * time.Second):
    		l.If("wswriter process - %s - alive", m.subscriber.id)
		}
	}
	
}