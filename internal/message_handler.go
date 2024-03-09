package internal

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"net/http"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type MessageHandler struct {
	mqttClient     MQTT.Client
	responsesTopic string
	targetURL      string
	splitLines     bool
}

func NewMessageHandler(mqttClient MQTT.Client, responsesTopic, targetURL string, splitLines bool) *MessageHandler {
	mh := MessageHandler{
		mqttClient:     mqttClient,
		responsesTopic: responsesTopic,
		targetURL:      targetURL,
		splitLines:     splitLines,
	}
	return &mh
}

func (handler *MessageHandler) OnMessage(client MQTT.Client, mqttMessage MQTT.Message) {
	log.Printf("incoming message on %s", mqttMessage.Topic())
	req, err := http.NewRequest(http.MethodPost, handler.targetURL, bytes.NewReader(mqttMessage.Payload()))
	if err != nil {
		log.Printf("error creating request to %s: %v", handler.targetURL, err)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("error making request to %s: %v", handler.targetURL, err)
		return
	}
	log.Printf("response status code %d", res.StatusCode)
	defer res.Body.Close()

	if handler.splitLines {
		scanner := bufio.NewScanner(res.Body)
		scanner.Split(bufio.ScanLines)
		for scanner.Scan() {
			token := handler.mqttClient.Publish(handler.responsesTopic, 1, false, scanner.Text())
			<-token.Done()
			if token.Error() != nil {
				log.Printf("error trying to publish to responses topic: %v", token.Error())
			}
		}
		return
	}

	// Return entire response payload as a single message
	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("error reading response body from %s: %v", handler.targetURL, err)
		return
	}
	token := handler.mqttClient.Publish(handler.responsesTopic, 1, false, body)
	<-token.Done()
	if token.Error() != nil {
		log.Printf("error trying to publish to responses topic: %v", token.Error())
	}
}
