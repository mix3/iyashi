package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mix3/ape-slack"
)

type FlickrSearchResponse struct {
	Photos struct {
		Page    int    `json:"page"`
		Pages   int    `json:"pages"`
		PerPage int    `json:"perpage"`
		Total   string `json:"total"`
		Photo   []struct {
			Id       string `json:"id"`
			Owner    string `json:"owner"`
			Secret   string `json:"secret"`
			Server   string `json:"server"`
			Farm     int    `json:"farm"`
			Title    string `json:"title"`
			Ispublic int    `json:"ispublic"`
			Isfriend int    `json:"isfriend"`
			Isfamily int    `json:"isfamily"`
		} `json:"photo"`
	} `json:"photos"`
}

type TumblrSearchResponse struct {
	Response struct {
		Posts []struct {
			Photos []struct {
				OriginalSize struct {
					Url string `json:"url"`
				} `json:"original_size"`
			} `json:"photos"`
		} `json:"posts"`
		TotalPosts int `json:"total_posts"`
	} `json:"response"`
}

func get(baseUrl string, param map[string]string) (*http.Response, error) {
	queries := []string{}
	for k, v := range param {
		queries = append(queries, fmt.Sprintf("%s=%s", k, url.QueryEscape(v)))
	}
	reqUrl := fmt.Sprintf("%s?%s", baseUrl, strings.Join(queries, "&"))
	return http.Get(reqUrl)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	var (
		slackBotToken  = flag.String("slack-bot-token", "", "slack bot token")
		flickrApiToken = flag.String("flickr-api-token", "", "flickr api token")
		tumblrApiToken = flag.String("tumblr-api-token", "", "tumblr api token")
	)
	flag.Parse()

	conn := ape.New(*slackBotToken)

	conn.AddAction("癒やし", "`@{{ .EventCtx.UserName }} 癒やし <文言>` でflickrから画像を返すよ! 文言はスペース区切りで複数指定可", func(e *ape.Event) error {
		args := e.Command().Args()
		if len(args) == 0 {
			args = append(args, "猫")
		}
		args = append(args, "-hentai", "-porn", "-sexy", "-fuck")
		keyword := strings.Join(args, " ")

		resp, err := get(
			"https://api.flickr.com/services/rest/",
			map[string]string{
				"api_key":        *flickrApiToken,
				"format":         "json",
				"nojsoncallback": "1",
				"method":         "flickr.photos.search",
				"text":           keyword,
				"safe_mode":      "1",
				"media":          "photo",
			},
		)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var res FlickrSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			return err
		}

		if len(res.Photos.Photo) == 0 {
			return fmt.Errorf("見つからないよ(´・ω・｀)")
		}

		photo := res.Photos.Photo[rand.Intn(len(res.Photos.Photo)+1)]

		imgUrl := fmt.Sprintf(
			"https://farm%d.staticflickr.com/%s/%s_%s.jpg",
			photo.Farm,
			photo.Server,
			photo.Id,
			photo.Secret,
		)

		e.ReplyWithoutPermalink(imgUrl)

		return nil
	})

	zoiTotalPosts := 0
	conn.AddAction("ぞい", "http://ganbaruzoi.tumblr.com/ から画像をランダムで返すよ", func(e *ape.Event) error {
		offset := 0
		if 0 < zoiTotalPosts {
			offset = rand.Intn(zoiTotalPosts/20+1) * 20
		}
		urls := []string{}
		url := fmt.Sprintf(
			"http://api.tumblr.com/v2/blog/ganbaruzoi.tumblr.com/posts/photo?api_key=%s&offset=%d",
			*tumblrApiToken,
			offset,
		)
		resp, err := http.Get(url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var res TumblrSearchResponse
		err = json.NewDecoder(resp.Body).Decode(&res)
		if err != nil {
			return fmt.Errorf("failed unmarshal json: %v", err)
		}

		zoiTotalPosts = res.Response.TotalPosts

		for _, post := range res.Response.Posts {
			for _, photo := range post.Photos {
				urls = append(urls, photo.OriginalSize.Url)
			}
		}

		if len(urls) == 0 {
			return fmt.Errorf("見つからないよ(´・ω・｀)")
		}

		e.ReplyWithoutPermalink(urls[rand.Intn(len(urls))])

		return nil
	})

	log.Println("launch bot")

	conn.Loop()
}
