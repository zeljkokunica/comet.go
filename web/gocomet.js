/**
 * js client for gocomet - uses WebSocket where possible, or long poll otherwise.
 * options:
 * 	channels: array<string> - array of channel names to subscribe
 * 	onDataListener: function(command, channel, version, data) - called when data from channel is received
 * 	onSubscribed: function(id) - called when client is successfully subscribed
 * 	onClosed: function() - called when connection is closed
 * 	reconnect: boolean = true - reconnect if connection gets closed
 * 	crossDomain: boolean = true - make cross domain requests (cors)
 * 	forceLongPoll: boolean = false - force using of LongPoll over WebSockets
 *  debug: boolean = false - debug console output
 *  ip: string - ip with port - 127.0.0.1:8080
 *  useSSL: boolean = false
 * Example usage:
 * var comet = GoComet({channels: ["global"], onDataListener: onNewData});
 * 
 * public methods:
 * addChannels(channels) - subscribe to channels (array of channel names)
 * removeChannels(channels) - unsubscribe to channels (array of channel names)
 * connectionType() - returns WebSocket or LongPoll
 */

GoComet = function(options) {
	options = options || {};
	options.channels = options.channels || [];
	options.ip = options.ip || window.location.host;
	if (typeof(options.reconnect) === "undefined" || options.reconnect === null) {
		options.reconnect = true;
	}
	if (typeof(options.crossDomain) === "undefined" || options.crossDomain === null) {
		options.crossDomain= true;
	}
	if (typeof(options.useSSL) === "undefined" || options.useSSL === null) {
		options.useSSL= false;
	}
	
	if (typeof(WebSocket) === "undefined" || options.forceLongPoll) {
		return GoCometLongPoll(options);
	}
	else {
		return GoCometWS(options);
	}
}

GoCometWS = function(options) {
	var debug = true,
		ws = null,
		requestId = 0,
		responseHandlers = {},
		id = null,
		channels = [],
		keepAliveId = null,
		isConnecting = false,
		channelVersion,
		// methods
		reconnect,
		create,
		onMessage,
		send,
		subscribe,
		_addChannels,
		_removeChannels;
	
	if (typeof(WebSocket) === "undefined") {
		return GoCometLongPoll(options);
	}
	
	reconnect = function() {
		if (!options.reconnect) {
			return;
		}
		responseHandlers = {};
		setTimeout(
			function(){
				create();
			}, 500);
	}
	
	create = function() {
		var wsUrl = "ws://";
		if (options.useSSL) {
			wsUrl = "wss://";
		}
		wsUrl +=  options.ip + "/ws"
		if (options.debug) console.log("Connecting ws...");
		if (isConnecting) {
			return;
		}
		isConnecting = true;
		if (keepAliveId) {
			clearInterval(keepAliveId);
		}
		channels = options.channels;
		
  	  	ws = new WebSocket(wsUrl);
		ws.onclose = function(evt) {
			if (options.onClosed) {
				options.onClosed();
			} 
			reconnect();
		}
		ws.onmessage = onMessage;
		
		keepAliveId = setInterval(
			function(){
		    	send("keepAlive", {});
		    }, 
		    15000);
		subscribe();
	};
	
	
	send = function(command, params, success, fail) {
	  if (ws.readyState == 0) {
		  if (options.debug) console.log("ws connecting..." + ws.readyState);
		  setTimeout(function(){
			  send(command, params, success, fail);
			  }, 
			  500);
		  return;
	  }
	  else if (ws.readyState != 1) {
		  isConnecting = false;
		  if (options.debug) console.log("ws not ready..." + ws.readyState);
		  reconnect();
		  return
	  }
	  isConnecting = false;
	  requestId++;
	  responseHandlers[requestId] = {success: success, fail: fail};
	  var request = {requestId: requestId, command: command, parameters: params};
	  var requestJson = JSON.stringify(request);
	  ws.send(requestJson);
	};
		  
	onMessage = function(evt) {
		var event = JSON.parse(evt.data);
		var requestId = event.requestId;
		if (responseHandlers[requestId]) {
			if (responseHandlers[requestId].success) {
				responseHandlers[requestId].success(event.data);
		    }
		  	delete responseHandlers[requestId];
		}
		else {
			jQuery.each(event.commands, function(index, data){
				if (options.onDataListener) {
					options.onDataListener(data.command, data.channel, data.version, data.data);
				}
            });
		}
	};
		
	request = function(command, params, success, error) {
		var paramsMap = {};
		jQuery.each(params, function(index, param){
			paramsMap[param.name] = param.value;
		});
		send(command, paramsMap, success, error);
	};
	
	_addChannels = function(channels) {
		var newChannels = "";
		jQuery.each(channels, function(index, channel){
			if (newChannels == "") {
				newChannels = channel;
			}
			else {
				newChannels += "," + channel;
			}
		});
		request(
			"addchannels", 
			[{name: "id", value: id}, {name: "channels", value: newChannels}], 
			function(data){
				jQuery.each(channels, function(index, channel){
			   		channels.push(channels);
				});
			},
			null);
	};
	
	_removeChannels = function(channels) {
		var newChannels = "";
		jQuery.each(channels, function(index, channel){
			if (newChannels == "") {
				newChannels = channel;
			}
			else {
				newChannels += "," + channel;
			}
		});
		
		request(
			"removechannels",
			[{name: "id", value: id}, {name: "channels", value: newChannels}],
			function(data) {
				jQuery.each(channels, function(index, channel){
				// TODO
				});
			},
			null);
	};
	
	subscribe = function() {
		var channelsParam = "";
		jQuery.each(options.channels, function(index, channel){
			if (channelsParam == "") {
				channelsParam = channel;
			}
			else {
				channelsParam += "," + channel;
			}
		});
		request(
			"subscribe",
			[{name: "channels", value: channelsParam}],
			function(data) {
				if (options.debug) console.log("subscribed: " + data.subscriberId);
				id = data.subscriberId;
				channelVersion = {};
				if (options.onSubscribed) {
					options.onSubscribed(id);
				}
			},
			function(){
				reconnect();
			});
	}
	
	create();
	
	return {
		addChannels: function(channels) {
			_addChannels(channels);
		},
		removeChannels: function(channels) {
			_removeChannels(channels);
		},
		connectionType: function() {
			return "WebSocket";
		}
	}
}

GoCometLongPoll = function(options) {
	var debug = true,
		id = null,
		channels = [],
		keepAliveId = null,
		channelVersion,
		// methods
		getData,
		request,
		subscribe,
		_addChannels,
		_removeChannels,
		url = "http";

	if (options.useSSL) {
		url+= "s";
	}
	url += "://" + options.ip + "/";
	
	request = function(command, params, success, error) {
		var requestUrl = url + command;
		var paramsUrl = "";
		// add timestamp to skip cache on IE
		params.push({name: "__ts", value: new Date().getTime()});
		jQuery.each(params, function(index, param){
			if (paramsUrl != "") {
				paramsUrl += "&";
			}  
			paramsUrl += param.name + "=" + param.value;
		});
		if (paramsUrl != "") {
			requestUrl += "?" + paramsUrl;
		}
		if (options.debug) console.log("Ajax requerst url: " + requestUrl);
		jQuery.ajax({
			url: requestUrl,
		    context: document.body,
		    crossDomain: options.crossDomain,
		    timeout: 60000
		 }).success(function(data, status, jqxhr) {
			if (options.debug) console.log("Ajax response: " + data);
			if (success) {
				success(JSON.parse(data));
			}
		 }).error(function(qXHR, status, errorThrown){
			if (error) {
				error();
			}
		 });
	};
	_addChannels = function(channels) {
		var newChannels = "";
		jQuery.each(channels, function(index, channel){
			if (newChannels == "") {
				newChannels = channel;
			}
			else {
				newChannels += "," + channel;
			}
		});
		request(
			"addchannels", 
			[{name: "id", value: id}, {name: "channels", value: newChannels}], 
			function(data){
				jQuery.each(channels, function(index, channel){
					channels.push(channels);
				});
			},
			null);
	};
	
	_removeChannels = function(channels) {
		var newChannels = "";
		jQuery.each(channels, function(index, channel){
			if (newChannels == "") {
				newChannels = channel;
			}
			else {
				newChannels += "," + channel;
			}
		});
		request(
			"removechannels",
			[{name: "id", value: id}, {name: "channels", value: newChannels}],
			function(data) {
				jQuery.each(channels, function(index, channel){
					// TODO
				});
			},
			null);
	};
	
	subscribe = function() {
		var channelsParam = "";
		jQuery.each(options.channels, function(index, channel){
			if (channelsParam == "") {
				channelsParam = channel;
			}
			else {
				channelsParam += "," + channel;
			}
		});
		request(
			"subscribe",
			[{name: "channels", value: channelsParam}],
			function(data) {
				id = data.subscriberId;
				if (options.onSubscribed) {
					options.onSubscribed(id);
				}
				channelVersion = {};
				setTimeout(getData, 1);
			},
			function(){
				subscribe();
			});
	};
	getData = function() {
		if (options.debug) console.log("getData");
		request(
			"data",
			[{name: "id", value: id}],
			function(result) {
				if (options.debug) console.log("getData - received");
				// got data
				if (result.status == "1") {
					jQuery.each(result.commands, function(index, data){
						if (options.onDataListener) {
							options.onDataListener(data.command, data.channel, data.version, data.data);
						}
					});
					setTimeout(getData, 1);
				}
				// no data yet
				else if (result.status == "0") {
					setTimeout(getData, 1);
				}
				// not subscribed
				else if (result.status == "-1") {
					setTimeout(subscribe, 1);
				}
				else {
					setTimeout(getData, 1);
				}
			},
			function(){
				if (options.debug) console.log("error");
				setTimeout(getData, 1000);
		});
	}
	subscribe();
	return {
		addChannels: function(channels) {
			_addChannels(channels);
		},
		removeChannels: function(channels) {
			_removeChannels(channels);
		},
		connectionType: function() {
			return "LongPoll";
		}
	}
}