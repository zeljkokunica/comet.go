comet.go
===============

comet server implementation in golang. This is development version and not ready for other uses.

install:
===============

* install go
* clone app: git clone git@github.com:zeljkokunica/comet.go.git
* cd comet.go
* ./setup_and_build.sh
* ./comet_server --port=8080


Supported options:
===============

* data channels
* additional subscriptions/unsubscriptions
* multiple channel subscription
* simple channel persistance using files (can be restarted)
* data create, update and clear via http request (requires channel name and data - string, usualy containing json)
* web socket communication where available, and long poll as a fallback

Not yet supported, but planned
===============

* channel persistance using redis
* channel timeout - clear channel when and if not persistent


Includes:
===============

* js lib for web client (uses jQuery)
* go comet server
* client example page (http://localhost:8090/client.html)
* status page (http://localhost:8090/status.html)
* simple data feeder example page (http://localhost:8090/feeder.html)

Other TODOs
===============

* test and tune
* supervision of health?

Future plans :)
===============

* conduct all client-server communication through this server:
* all communication through web socket (if available)
* routing requests to other systems
* each client has it's own private channel
* other systems informs client through it's private channel (progress, results...)
