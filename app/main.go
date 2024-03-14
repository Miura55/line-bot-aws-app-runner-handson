package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func healthHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Health check")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func eventHandler(req *webhook.CallbackRequest, r *http.Request, bot *messaging_api.MessagingApiAPI, err error) {
	log.Println("Received events")
	for _, event := range req.Events {
		log.Printf("Event: %v", event)
		switch e := event.(type) {
		case webhook.FollowEvent:
			if _, err = bot.ReplyMessage(
				&messaging_api.ReplyMessageRequest{
					ReplyToken: e.ReplyToken,
					Messages: []messaging_api.MessageInterface{
						&messaging_api.TextMessage{
							Text: "友達追加ありがとう！",
						},
					},
				},
			); err != nil {
				log.Println(err)
				return
			}
		case webhook.MessageEvent:
			switch message := e.Message.(type) {
			case webhook.TextMessageContent:
				if _, err = bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
						Messages: []messaging_api.MessageInterface{
							&messaging_api.TextMessage{
								Text: message.Text,
							},
						},
					},
				); err != nil {
					log.Println(err)
					return
				}
			}
		}
	}
}

func main() {
	handler, err := webhook.NewWebhookHandler(os.Getenv("CHANNEL_SECRET"))
	if err != nil {
		log.Fatal(err)
		return
	}

	bot, err := messaging_api.NewMessagingApiAPI(os.Getenv("CHANNEL_TOKEN"))
	if err != nil {
		log.Fatal(err)
		return
	}

	handler.HandleEvents(func(req *webhook.CallbackRequest, r *http.Request) {
		eventHandler(req, r, bot, err)
	})

	http.HandleFunc("/health", healthHandler)
	http.Handle("/callback", handler)
	fmt.Println("Server is running...:8080")
	http.ListenAndServe(":8080", nil)
}
