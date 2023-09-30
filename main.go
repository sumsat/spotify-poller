package main

import (
	"context"
	"log"
	"os"
	"net/http"
	"bytes"
	"database/sql"

	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
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
	log.Printf("postgres client start")
	db, err := sql.Open("latest_id_user", os.Getenv("POSTGRES_URL"))
	if err != nil {
		log.Fatal("Error postgres")
	}
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

	log.Printf("get show start")
	res, err := client.GetShow(ctx, spotify.ID(os.Getenv("SPOTIFY_SHOW_ID")), spotify.Market("JP"))
	if err != nil {
		log.Fatalf("SPOTIFY request failed")
	}

	episodes := res.Episodes.Episodes
	latestEpisode := episodes[len(episodes) - 1]

	rows, err := db.Query("SELECT latest_id FROM latest_id WHERE id = 0")
	defer rows.Close()
	if err != nil {
		log.Fatalf("get failed")
	}

	var ids []string
	var tmp string
	for rows.Next() {
		rows.Scan(&tmp)
		ids = append(ids, tmp)
	}

	if (ids[0] != latestEpisode.ID.String()) {
		_, err = db.Exec("UPDATE latest_id SET latest_id = $1 where id = 0", latestEpisode.ID.String())
		log.Printf(latestEpisode.ID.String())
		if err != nil {
			log.Fatalf("instert failed")
		} else {
			HttpPost(os.Getenv("DISCORD_WEBHOOK_URL"), latestEpisode.Name, latestEpisode.ID.String())
		}
	}
	log.Printf("end")

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
