package mqtt

import (
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type MessageHandler func(topic string, payload []byte)

type Subscriber struct {
	client         mqtt.Client
	messageHandler MessageHandler
}

func NewSubscriber(broker string, clientID string, messageHandler MessageHandler) (*Subscriber, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true).
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting to broker: %v", token.Error())
	}

	return &Subscriber{
		client:         client,
		messageHandler: messageHandler,
	}, nil
}

func (s *Subscriber) Subscribe(topic string) error {
	token := s.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		if s.messageHandler != nil {
			s.messageHandler(msg.Topic(), msg.Payload())
		}
	})

	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error subscribing to topic %s: %v", topic, token.Error())
	}

	log.Printf("Subscribed to topic: %s\n", topic)
	return nil
}

func (s *Subscriber) Disconnect() {
	if s.client.IsConnected() {
		s.client.Disconnect(250)
	}
}
