package comet

import (
	"l"
	"time"
	"fmt"
	"net/http"
	"strings"
	"io/ioutil"
	"encoding/json"
	"code.google.com/p/go.net/websocket"
)
/**
* parse request, and return response
*/
type DataMediator interface {
	/**
	* read parameter from request
	*/
	ReadParameter(parameterName string) string
	
	/**
	* writes response
	*/
	WriteResponse(response interface{}, contentType string)
}

type HttpDataMediator struct {
	w http.ResponseWriter 
	r *http.Request
}

func (m *HttpDataMediator) ReadParameter(parameterName string) string {
	return m.r.FormValue(parameterName)
}

func (m *HttpDataMediator) WriteResponse(response interface{}, contentType string)  {
	if (contentType == "json") {
		jsonData, _ := json.Marshal(response)
		fmt.Fprintf(m.w, "%s", jsonData)
	} else {
		fmt.Fprintf(m.w, "%s", response)
	}
}

func (h *Hub) route(command string, mediator DataMediator) {
	if command == "ping" {
		mediator.WriteResponse("pong", "plain");		
	} else if command == "subscribe" {
		h.onSubscribeRequest(mediator)		
	} else if command == "addchannels" {
		h.onAddChannelsRequest(mediator)
	} else if command == "removechannels" {
		h.onRemoveChannelsRequest(mediator)
	} else if command == "data" {
		h.onGetDataRequest(mediator)
	} else if command == "create" {
		h.onCreateDataRequest(mediator)
	} else if command == "update" {
		h.onUpdateDataRequest(mediator)
	} else if command == "clear" {
		h.onClearDataRequest(mediator)
	} else {
		h.onServeFileRequest(mediator, command)
	}
}

func (h *Hub) onSubscribeRequest(m DataMediator) {
	var channels = strings.Split(m.ReadParameter("channels"), ",")
	var responseListener = make(chan Subscriber)
	defer close(responseListener)
	h.subscriberCommandListener <- HubSubscriberRequest{command: Subscribe, channels: channels, responseListener: responseListener}
	var subscriber = <- responseListener
	m.WriteResponse(map[string]interface{}{"command": "subscribe", "subscriberId": subscriber.id}, "json");
}

func (h *Hub) onAddChannelsRequest(m DataMediator) {
	var channels = strings.Split(m.ReadParameter("channels"), ",")
	var id = m.ReadParameter("id")
	var responseListener = make(chan Subscriber)
	defer close(responseListener)
	h.subscriberCommandListener <- HubSubscriberRequest{command: SubscribeToChannels, channels: channels, subscriberId: id, responseListener: responseListener}
	var subscriber = <- responseListener
	m.WriteResponse(subscriber.id, "plain");
}

func (h *Hub) onRemoveChannelsRequest(m DataMediator) {
	var channels = strings.Split(m.ReadParameter("channels"), ",")
	var id = m.ReadParameter("id")
	var responseListener = make(chan Subscriber)
	defer close(responseListener)
	h.subscriberCommandListener <- HubSubscriberRequest{command: UnsubscribeFromChannels, channels: channels, subscriberId: id, responseListener: responseListener}
	var subscriber = <- responseListener
	m.WriteResponse(subscriber.id, "plain");
}

func (h *Hub) onGetDataRequest(m DataMediator) {
	var id = m.ReadParameter("id")
	var subscriber = h.subscribers[id]
	// subscriber not found
	if (subscriber == nil) {
		var response = SubscriberResponse{Status: -1}
		m.WriteResponse(response, "json");
	} else {
		subscriber.lastRequest = time.Now().Unix()
		timeout := time.After(30 * time.Second)
		select {
			case command := <- subscriber.feedListener:
				var response SubscriberResponse
				if len(command.data) > 0 {
					response = SubscriberResponse{Status: 1, Commands: command.data}
				} else {
					response = SubscriberResponse{Status: 0}
				}
				m.WriteResponse(response, "json");
			case <- timeout:
				var response = SubscriberResponse{Status: 0}
				m.WriteResponse(response, "json");
		}
	}
}

func (h *Hub) onCreateDataRequest(m DataMediator) {
	var channel = m.ReadParameter("channel")
	var dataParam = m.ReadParameter("data")
	h.AddNewDataToChannel("create", channel, dataParam)
}

func (h *Hub) onUpdateDataRequest(m DataMediator) {
	var channel = m.ReadParameter("channel")
	var dataParam = m.ReadParameter("data")
	h.AddNewDataToChannel("update", channel, dataParam)
}

func (h *Hub) onClearDataRequest(m DataMediator) {
	var channel = m.ReadParameter("channel")
	h.AddNewDataToChannel("clear", channel, "")
}

func (h *Hub) onServeFileRequest(m DataMediator, file string) {
	b, err := ioutil.ReadFile("web/" + file)
	if err != nil {
		l.Wf("file not found %s", "web/" + file)
	}
	if err == nil {
		l.Df("found file %s", file)
		m.WriteResponse(string(b), "plain");
	}
}

/**
* Http request handler 
*/
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if x := recover(); x != nil {
    	l.Ef("request %s caused error from %s {%s}: %v", r.URL.Path, r.RemoteAddr, r.Form.Encode(), x)
    }
	}() 
	w.Header().Set("Access-Control-Allow-Origin", "*")
	parts := strings.Split(r.URL.Path, "/")
	command := parts[1]
	r.ParseForm()
	startTime := time.Now()
	l.If("http serving request %s from %s {%s}", r.URL.Path, r.RemoteAddr, r.Form.Encode())
	mediator := HttpDataMediator{w: w, r: r}
	h.route(command, &mediator);	
	delay := float64(time.Now().Sub(startTime).Nanoseconds()) / 1000000.0
	l.If("http Served request %s from %s {%s} took %f ms", r.URL.Path, r.RemoteAddr, r.Form.Encode(), delay)
}
/**
* Websocket handler 
*/
func (h *Hub) ServeWebsocket(ws *websocket.Conn) {
	l.W("WebSocket connection")
	handler := WebSocketHandler{ws: ws, send: make(chan WebSocketResponse, 255), hub: h, closeListener: make(chan bool)}
  handler.reader()
}
