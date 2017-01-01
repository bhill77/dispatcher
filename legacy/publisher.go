package dispatcher

import (
	"encoding/json"
	"errors"
	"github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

type Publisher struct {
	log               Log
	ch                *amqp.Channel
	confirmationChan  chan amqp.Confirmation
	active            bool
	defaultExchange   string
	defaultRoutingKey string
}

func (s *Publisher) PublishCustom(task *Task, exchange, routingKey string) error {

	if task.Exchange == "" {
		if exchange != "" {
			task.Exchange = exchange
		} else {
			if s.defaultExchange != "" {
				task.Exchange = s.defaultExchange
			} else {
				return errors.New("No exchange passed")
			}
		}
	}

	if task.RoutingKey == "" {
		if routingKey != "" {
			task.RoutingKey = routingKey
		} else {
			if s.defaultRoutingKey != "" {
				task.RoutingKey = s.defaultRoutingKey
			} else {
				return errors.New("No routing key passed")
			}
		}
	}

	return s.publishTask(task)

}

func (s *Publisher) Publish(task *Task) error {

	if task.Exchange == "" {
		if s.defaultExchange != "" {
			task.Exchange = s.defaultExchange
		} else {
			return errors.New("No exchange passed")
		}
	}

	if task.RoutingKey == "" {
		if s.defaultRoutingKey != "" {
			task.RoutingKey = s.defaultRoutingKey
		} else {
			return errors.New("No routing key passed")
		}
	}

	return s.publishTask(task)

}

func (s *Publisher) publishTask(task *Task) error {

	if task.UUID == "" {
		task.UUID = uuid.NewV4().String()
	}

	if task.Name == "" {
		return errors.New("Task name was not passed")
	}

	msg, err := json.Marshal(task)
	if err != nil {
		return err
	}

	if !s.active {
		return errors.New("Service is disconnected")
	}

	err = s.ch.Publish(task.Exchange, task.RoutingKey, false, false, amqp.Publishing{
		Headers:      amqp.Table(task.Headers),
		ContentType:  "application/json",
		Body:         msg,
		DeliveryMode: amqp.Persistent,
	})
	if err != nil {
		return err
	}

	confirmed := <-s.confirmationChan

	if confirmed.Ack {
		return nil
	}

	return errors.New("Failed to deliver message")
}