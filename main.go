package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	"github.com/kwkoo/configparser"
	"github.com/kwkoo/mqtt-rest-bridge/internal"
)

type Config struct {
	MQTTBroker    string `usage:"MQTT broker URL" default:"tcp://localhost:1883" mandatory:"true"`
	IncomingTopic string `usage:"MQTT topic for incoming messages" mandatory:"true"`
	OutgoingTopic string `usage:"MQTT topic for outgoing messages" mandatory:"true"`
	TargetURL     string `usage:"URL of the REST endpoint" mandatory:"true"`
	SplitLines    bool   `usage:"Send each line as a separate message" default:"false"`
}

func main() {
	config := Config{}
	if err := configparser.Parse(&config); err != nil {
		log.Fatal(err)
	}

	shutdownCtx, cancelSignalNotify := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	var wg sync.WaitGroup

	mqttClient := initializeMQTTClient(config)
	wg.Add(1)
	go func() {
		<-shutdownCtx.Done()
		shutdownMQTTClient(mqttClient)
		wg.Done()
	}()

	<-shutdownCtx.Done()
	cancelSignalNotify()
	log.Print("signal received, waiting for all goroutines to shut down...")
	wg.Wait()
	log.Print("all goroutines terminated")
}

func initializeMQTTClient(config Config) MQTT.Client {
	opts := MQTT.NewClientOptions()
	opts.AddBroker(config.MQTTBroker)
	opts.SetAutoReconnect(true)
	opts.OnConnect = func(mqttClient MQTT.Client) {
		handler := internal.NewMessageHandler(mqttClient, config.OutgoingTopic, config.TargetURL, config.SplitLines)
		if token := mqttClient.Subscribe(config.IncomingTopic, 1, handler.OnMessage); token.Wait() && token.Error() != nil {
			log.Fatalf("could not subscribe to %s: %v", config.IncomingTopic, token.Error())
		}
	}

	mqttClient := MQTT.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("error connecting to MQTT broker %s: %v", config.MQTTBroker, token.Error())
	}
	log.Printf("successfully connected to MQTT broker %s", config.MQTTBroker)

	return mqttClient
}

func shutdownMQTTClient(mqttClient MQTT.Client) {
	log.Print("shutting down MQTT client...")
	mqttClient.Disconnect(5000)
	log.Print("MQTT client successfully shutdown")
}
