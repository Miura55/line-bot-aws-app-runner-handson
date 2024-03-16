package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

type TodoItem struct {
	UserId    string `dynamodbav:"userId"`
	Timestamp string `dynamodbav:"timestamp"`
	Text      string `dynamodbav:"text"`
}

type TodoQuery struct {
	UserId string `dynamodbav:":userId"`
}

func todoController(userId string, text string, timestamp int64) (messaging_api.TextMessage, error) {
	region := os.Getenv("AWS_REGION")
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
		return messaging_api.TextMessage{}, err
	}

	client := dynamodb.NewFromConfig(cfg)
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")

	replyMessage := messaging_api.TextMessage{}
	switch text {
	case "list":
		query := TodoQuery{
			UserId: userId,
		}
		av, err := attributevalue.MarshalMap(query)
		if err != nil {
			log.Fatal(err)
			return replyMessage, err
		}

		result, err := client.Query(context.TODO(), &dynamodb.QueryInput{
			TableName:              aws.String(tableName),
			KeyConditionExpression: aws.String("#userId = :userId"),
			ExpressionAttributeNames: map[string]string{
				"#userId": *aws.String("userId"),
			},
			ExpressionAttributeValues: av,
		})
		if err != nil {
			log.Fatal(err)
			return replyMessage, err
		}

		for _, item := range result.Items {
			var todoItem TodoItem
			err = attributevalue.UnmarshalMap(item, &todoItem)
			if err != nil {
				log.Fatal(err)
				return replyMessage, err
			}
			log.Println(todoItem)
		}
	default:
		item := TodoItem{
			UserId:    userId,
			Timestamp: fmt.Sprint(timestamp),
			Text:      text,
		}
		av, err := attributevalue.MarshalMap(item)
		if err != nil {
			log.Fatal(err)
			return replyMessage, err
		}

		_, err = client.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      av,
		})
		if err != nil {
			log.Fatal(err)
			return replyMessage, err
		}
		replyMessage.Text = "登録しました"
	}
	return replyMessage, nil
}

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
			// 送信元のIDを取得
			sourceId := ""
			switch s := e.Source.(type) {
			case webhook.UserSource:
				sourceId = s.UserId
			case webhook.GroupSource:
				sourceId = s.GroupId
			case webhook.RoomSource:
				sourceId = s.RoomId
			}
			log.Printf("SourceId: %v", sourceId)

			// メッセージの種類によって処理を分岐
			switch message := e.Message.(type) {
			case webhook.TextMessageContent:
				replyMessage := messaging_api.TextMessage{
					Text: message.Text,
				}
				timestamp := e.Timestamp
				replyMessage, err = todoController(sourceId, message.Text, timestamp)
				if err != nil {
					log.Println(err)
					return
				}

				if _, err = bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
						Messages: []messaging_api.MessageInterface{
							&replyMessage,
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
