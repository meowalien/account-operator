package price

import (
	"account-operator/rabbitmq"
	"context"
	"github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type PriceReceiver interface {
	Start(ctx context.Context) (Delivers, error)
	Close()
}

func NewPriceReceiver() PriceReceiver {
	exchangeName := viper.GetString("receiver.exchangeName")
	symbols := viper.GetStringSlice("receiver.symbols")
	return &priceReceiver{
		exchangeName: exchangeName,
		symbols:      symbols,
	}
}

type priceReceiver struct {
	ch           *amqp091.Channel
	exchangeName string
	symbols      []string
}

type Delivers map[string]<-chan amqp091.Delivery

func (p *priceReceiver) makeDeliveryChan(symbol string) (symbolCh <-chan amqp091.Delivery, err error) {
	q, err := p.ch.QueueDeclare(
		"",    // name
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return nil, err
	}
	err = p.ch.QueueBind(
		q.Name,         // queue name
		symbol,         // routing key
		p.exchangeName, // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}
	symbolCh, err = p.ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		true,   // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, err
	}
	return symbolCh, nil
}

func (p *priceReceiver) Start(ctx context.Context) (Delivers, error) {
	var err error
	p.ch, err = rabbitmq.NewChannel(ctx)
	if err != nil {
		return nil, err
	}
	err = p.ch.Qos(100, 0, false)
	if err != nil {
		return nil, err
	}
	var ds = make(Delivers)
	for _, symbol := range p.symbols {
		symbolCh, makeDeliveryChanErr := p.makeDeliveryChan(symbol)
		if makeDeliveryChanErr != nil {
			return nil, makeDeliveryChanErr
		}
		ds[symbol] = symbolCh
	}
	return ds, nil
}

func (p *priceReceiver) Close() {
	err := p.ch.Close()
	if err != nil {
		logrus.Errorf("Failed to close reporter: %s", err)
	}
	return
}
