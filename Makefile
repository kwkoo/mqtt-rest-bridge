BASE:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

.PHONY: run sub pub

run:
	cd $(BASE) \
	&& \
	MQTTBROKER="tcp://localhost:1883" \
	INCOMINGTOPIC="llmreq" \
	OUTGOINGTOPIC="llmresp" \
	TARGETURL="http://localhost:11434/api/generate" \
	SPLITLINES="true" \
	go run .

sub:
	docker exec \
	  -it \
	  mqtt \
	  mosquitto_sub -t llmresp

pub:
	docker exec \
	  -it \
	  mqtt \
	  mosquitto_pub \
	    -t llmreq \
		-m '{"model":"llama", "prompt":"Why is the sky blue?"}'
