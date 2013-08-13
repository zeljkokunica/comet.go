package main

import (
	"net/http"
	"os"
	"runtime"
	"github.com/zeljkokunica/comet.go/comet"
	"code.google.com/p/go.net/websocket"
	"flag"
	"fmt"
)


func httpServerProcess(hub *comet.Hub, restartListener chan string) {
	var ip = flag.String("ip", "0.0.0.0", "ip address to listen on")
	var port = flag.Int("port", 8080, "port to listen on")
	flag.Parse()
	var addr = fmt.Sprintf("%s:%d", *ip, *port);
	l.If("listening on %s", addr)
	http.HandleFunc("/", hub.ServeHTTP)
	http.Handle("/ws", websocket.Handler(hub.ServeWebsocket))
	err := http.ListenAndServe(addr, nil); 
  restartListener <- err.Error()
}

func main() {
	path, err := os.Getwd()
	if err != nil {
    panic(err)
	}
	l.If("current app path: %s", path)
	var processes = runtime.NumCPU()
	runtime.GOMAXPROCS(processes)
	l.If("using processes %d", processes)
	l.I("server started.")
	restartLisnener := make(chan string)
	hub := comet.NewHub()
	for {
		go httpServerProcess(hub, restartLisnener)
		reason := <-restartLisnener
		l.Wf("web server restarted: %s", reason) 
	}
	
}
