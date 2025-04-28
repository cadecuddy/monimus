package rabbitmq

import (
	"fmt"
	"os"

	"github.com/cadecuddy/spotify-scraper/internal/utils"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	Conn     *amqp.Connection
	Channel  *amqp.Channel
	ScrapeQ  amqp.Queue
	ProcessQ amqp.Queue
}

func InitRabbitMQClient() *RabbitMQClient {
	rabbitmqHost := os.Getenv("RABBITMQ_HOST_NAME")
	rabbitmqUser := os.Getenv("RABBITMQ_DEFAULT_USER")
	rabbitmqPass := os.Getenv("RABBITMQ_DEFAULT_PASS")

	connectionUrl := fmt.Sprintf("amqp://%s:%s@%s", rabbitmqUser, rabbitmqPass, rabbitmqHost)

	conn, connErr := amqp.Dial(connectionUrl)
	utils.FailOnError(connErr, "Failed to connect to RabbitMQ")

	queueChannel, err := conn.Channel()
	utils.FailOnError(err, "Failed to open a channel")

	scrapeQ, err := queueChannel.QueueDeclare(
		"ids-to-scrape",
		true,
		false,
		false,
		false,
		nil,
	)
	utils.FailOnError(err, "Failed to declare scrape queue")

	processQ, err := queueChannel.QueueDeclare(
		"ids-to-process",
		true,
		false,
		false,
		false,
		nil,
	)
	utils.FailOnError(err, "Failed to declare process queue")

	err = queueChannel.Qos(1, 0, false)
	utils.FailOnError(err, "Failed to set QoS")

	return &RabbitMQClient{
		Conn:     conn,
		Channel:  queueChannel,
		ScrapeQ:  scrapeQ,
		ProcessQ: processQ,
	}
}

func PreloadSeedUsername(rmq *RabbitMQClient) {
	seedUsername := os.Getenv("SPOTIFY_SEED_USERNAME")

	err := rmq.Channel.Publish(
		"",
		rmq.ScrapeQ.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(seedUsername),
		},
	)
	if err != nil {
		fmt.Println("Failed to publish seed username", err)
	}

	fmt.Printf("Preloaded seed username into the scrape queue.\n")
}

func (client *RabbitMQClient) PublishToQueue(queueName string, content string) error {
	err := client.Channel.Publish("", queueName, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(content),
	})

	return err
}
