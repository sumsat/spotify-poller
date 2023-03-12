package main

import (
	"context"
	"log"
	"os"
	"net/http"
	"time"
	"bytes"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"github.com/zmb3/spotify/v2"
	"golang.org/x/oauth2/clientcredentials"
)

func main() {
	log.Printf("env start")
	if os.Getenv("GO_ENV") != "production" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	log.Printf("end")
	log.Printf("redis client start")
	rdb := redis.NewClient(&redis.Options{
		Addr:	 os.Getenv("REDIS_URL"),
		Password: "", // no password set
		DB:	   0,  // use default DB
	})
	log.Printf("end")

	log.Printf("spotify client start")
	ctx := context.Background()
	config := &clientcredentials.Config{
		ClientID:	 os.Getenv("SPOTIFY_CLIENTID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENTSECRET"),
		TokenURL:	 spotifyauth.TokenURL,
	}
	token, err := config.Token(ctx)
	if err != nil {
		log.Fatalf("couldn't get token: %v", err)
	}
	log.Printf("end")

	httpClient := spotifyauth.New().Client(ctx, token)
	client := spotify.New(httpClient)

	for {
		log.Printf("get show start")
		res, err := client.GetShow(ctx, spotify.ID(os.Getenv("SPOTIFY_SHOW_ID")), spotify.Market("JP"))
		if err != nil {
			log.Fatalf(err)
		}

		episodes := res.Episodes.Episodes
		latestEpisode := episodes[len(episodes) - 1]

		val, err := rdb.Get(ctx, "latest:id").Result()
		if err != nil {
			log.Fatalf(err)
		}

		if (val != latestEpisode.ID.String()) {
			rdb.Set(ctx, "latest:id", latestEpisode.ID.String(), 0)
			HttpPost(os.Getenv("DISCORD_WEBHOOK_URL"), latestEpisode.Name, latestEpisode.ID.String())
		}
		log.Printf("end")
		time.Sleep(5 * 60 * time.Second)
	}

}

func HttpPost(url, name, id string) error {
	jsonStr := `{"content":"【ゆる哲学ラジオ】Spotifyが更新されたよ！\r\r` + name + `\r\r https://open.spotify.com/episode/` + id +`"}`

	log.Printf(jsonStr)
	req, err := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer([]byte(jsonStr)),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return err
}
