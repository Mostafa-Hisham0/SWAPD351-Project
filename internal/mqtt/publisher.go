package mqtt

import (
	"fmt"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Publisher struct {
	client mqtt.Client
}

func NewPublisher(broker string, clientID string) (*Publisher, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(broker).
		SetClientID(clientID).
		SetCleanSession(true).
		SetAutoReconnect(true)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("error connecting to broker: %v", token.Error())
	}

	return &Publisher{
		client: client,
	}, nil
}

func (p *Publisher) Publish(topic string, message []byte) error {
	token := p.client.Publish(topic, 0, false, message)
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("error publishing message: %v", token.Error())
	}
	return nil
}

func (p *Publisher) Disconnect() {
	if p.client.IsConnected() {
		p.client.Disconnect(250)
	}
}
