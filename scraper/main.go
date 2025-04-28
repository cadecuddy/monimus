package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cadecuddy/spotify-scraper/internal/rabbitmq"
	"github.com/cadecuddy/spotify-scraper/internal/redis"
	"github.com/go-rod/rod"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	ProcessedUsersKey = "processed_users"
	ScrapedUsersKey   = "scraped_users"
)

func main() {
	rmq := rabbitmq.InitRabbitMQClient()
	redisCli := redis.InitRedisClient()
	browser := rod.New().MustConnect()

	defer rmq.Conn.Close()
	defer rmq.Channel.Close()
	defer browser.MustClose()

	rabbitmq.PreloadSeedUsername(rmq)

	msgs, _ := rmq.Channel.Consume(
		rmq.ScrapeQ.Name, "", false, false, false, false, nil,
	)

	for msg := range msgs {
		processMessage(msg, rmq, redisCli, browser)
	}
}

func processMessage(msg amqp.Delivery, rmq *rabbitmq.RabbitMQClient, redisCli *redis.RedisClient, browser *rod.Browser) {
	userId := string(msg.Body)
	exists, _ := redisCli.Rdb.SIsMember(redisCli.Ctx, ProcessedUsersKey, userId).Result()
	// already processed user
	if exists {
		msg.Ack(false)
		return
	}

	fmt.Println("Scraping ID: ", userId)
	followers := scrapeFollowers(userId, browser)

	for followerId := range followers {
		if isScrapedOrProcessed(followerId, redisCli) {
			continue
		}
		rmq.PublishToQueue(rmq.ScrapeQ.Name, followerId)
		redisCli.Rdb.SAdd(redisCli.Ctx, ScrapedUsersKey, followerId)
	}

	redisCli.Rdb.SAdd(redisCli.Ctx, ProcessedUsersKey, userId)
	rmq.PublishToQueue(rmq.ProcessQ.Name, userId)

	msg.Ack(false)
}

func scrapeFollowers(userId string, browser *rod.Browser) map[string]bool {
	followers := make(map[string]bool)

	url := fmt.Sprintf("https://open.spotify.com/user/%s/followers", userId)
	fmt.Println(url)
	page := browser.MustPage(url)

	gridContainer, err := page.Timeout(5 * time.Second).Element(`div[data-testid="grid-container"]`)
	if err != nil {
		fmt.Println("No followers found for:", userId)
		return map[string]bool{}
	}

	followerCards, _ := gridContainer.Elements("a")
	for _, card := range followerCards {
		href, err := card.Property("href")
		if err != nil {
			log.Printf("Failed to get href for card: %v", err)
			continue
		}

		followerUrl := href.String()
		followerSlice := strings.Split(followerUrl, "/")
		followerId := followerSlice[len(followerSlice)-1]
		followers[followerId] = true
	}

	return followers
}

func isScrapedOrProcessed(userId string, redisCli *redis.RedisClient) bool {
	isProcessed, _ := redisCli.Rdb.SIsMember(redisCli.Ctx, "processed_users", userId).Result()
	isScraped, _ := redisCli.Rdb.SIsMember(redisCli.Ctx, "scraped_users", userId).Result()

	return isProcessed || isScraped
}
