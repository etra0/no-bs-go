package internal

import (
	"log"
	"net/url"
	"os"
	"strings"

	tbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api        *CobaltAPI
	dispatcher chan *VideoMessage
}

func NewBot(url string) *Bot {
	api := NewAPI(url)
	return &Bot{api: api, dispatcher: make(chan *VideoMessage)}
}

type VideoMessage struct {
	Name string
	Msg  tbot.VideoConfig
}

func (bot *Bot) HandleMessage(msg *tbot.Message) {
	tiktokLink := bot.containsTiktokLink(msg.Text)
	if tiktokLink == nil {
		return
	}

	result := bot.handleRequest(tiktokLink)
	if result == nil {
		log.Println("Couldn't get the video.")
		return
	}

	vConfig := tbot.NewVideo(msg.Chat.ID, tbot.FilePath(*result))
	vConfig.ReplyToMessageID = msg.MessageID

	output_msg := VideoMessage{Name: *result, Msg: vConfig}
	bot.dispatcher <- &output_msg
}

// If no tiktok link is found, it'll return a nil pointer.
// Otherwise, an URL pointer is returned.
func (bot *Bot) containsTiktokLink(msg string) *url.URL {
	words := strings.Split(msg, " ")

	var tiktok_link *string = nil
	for _, word := range words {
		if strings.Contains(word, "tiktok.com") {
			tiktok_link = &word
			break
		}
	}

	if tiktok_link == nil {
		return nil
	}

	log.Println("Found tiktok link: ", *tiktok_link)

	u, err := url.Parse(*tiktok_link)
	if err != nil {
		log.Println("Couldn't parse the URI ", *tiktok_link, " ", err)
		return nil
	}

	return u
}

// This function will be in charge of sending the messages. This function locks, it is suggested to run it with
// a goroutine.
func (b *Bot) RunDispatcher(bot *tbot.BotAPI) {
	for {
		result := <-b.dispatcher
		bot.Send(result.Msg)
		os.Remove(result.Name)
	}
}

// This function handles if the url is either a stream or a picker, and returns
// a video file accordingly. In case it cannot get the video, it simply returns a nil pointer.
func (bot *Bot) handleRequest(url *url.URL) *string {
	responseJson := bot.api.RequestTiktokInfo(url)

	switch responseJson["status"] {
	case "stream":
		response_video := bot.api.DownloadVideo(responseJson["url"].(string))
		return &response_video
	case "picker":
		slideshow, err := NewSlideshowFromRequest(responseJson)
		if err != nil {
			log.Println("Error while creating slideshow: ", err)
			return nil
		}

		response_video := slideshow.GenerateVideo()
		return response_video
	case "error":
		log.Println("Error: ", responseJson["text"])
	default:
		log.Println("Unknown status: ", responseJson["status"])
	}

	return nil
}
