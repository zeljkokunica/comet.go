export GOPATH=$PWD
go get code.google.com/p/go.net/websocket
go build github.com/zeljokunica/comet_server
mkdir $GOPATH/bin/data
cp $GOPATH/web $GOPATH/bin/