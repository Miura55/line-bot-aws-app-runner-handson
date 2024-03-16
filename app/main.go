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

type TodoKey struct {
	UserId    string `dynamodbav:"userId"`
	Timestamp string `dynamodbav:"timestamp"`
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

func todoController(userId string, timestamp string, text ...string) ([]messaging_api.MessageInterface, error) {
	region := os.Getenv("AWS_REGION")
	tableName := os.Getenv("DYNAMODB_TABLE_NAME")
	replyMessages := []messaging_api.MessageInterface{}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
		return replyMessages, err
	}

	client := dynamodb.NewFromConfig(cfg)

	// 削除処理を実行
	if text == nil {
		key := TodoKey{
			UserId:    userId,
			Timestamp: timestamp,
		}
		av, err := attributevalue.MarshalMap(key)
		if err != nil {
			log.Fatal(err)
			return replyMessages, err
		}
		_, err = client.DeleteItem(context.TODO(), &dynamodb.DeleteItemInput{
			TableName: aws.String(tableName),
			Key:       av,
		})
		if err != nil {
			log.Fatal(err)
			return replyMessages, err
		}
		replyMessages = append(replyMessages, &messaging_api.TextMessage{
			Text: "削除しました",
		})
	} else {
		// メッセージの種類によって処理を分岐
		textMessage := text[0]
		switch textMessage {
		case "list":
			query := TodoQuery{
				UserId: userId,
			}
			av, err := attributevalue.MarshalMap(query)
			if err != nil {
				log.Fatal(err)
				return replyMessages, err
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
				return replyMessages, err
			}

			actions := []messaging_api.ActionInterface{}
			if len(result.Items) == 0 {
				replyMessages = append(replyMessages, &messaging_api.TextMessage{
					Text: "タスクはありません",
				})
				return replyMessages, nil
			}

			for _, item := range result.Items {
				var todoItem TodoItem
				err = attributevalue.UnmarshalMap(item, &todoItem)
				if err != nil {
					log.Fatal(err)
					return replyMessages, err
				}
				log.Println(todoItem)
				actions = append(actions, &messaging_api.PostbackAction{
					Label: todoItem.Text,
					Data:  todoItem.Timestamp,
				})
			}
			replyMessages = append(replyMessages, &messaging_api.TemplateMessage{
				AltText: "タスク一覧",
				Template: &messaging_api.ButtonsTemplate{
					Text:    "タスク一覧です。完了したタスクをタップすると削除されます。",
					Actions: actions,
				},
			})
		default:
			item := TodoItem{
				UserId:    userId,
				Timestamp: timestamp,
				Text:      textMessage,
			}
			av, err := attributevalue.MarshalMap(item)
			if err != nil {
				log.Fatal(err)
				return replyMessages, err
			}

			_, err = client.PutItem(context.TODO(), &dynamodb.PutItemInput{
				TableName: aws.String(tableName),
				Item:      av,
			})
			if err != nil {
				log.Fatal(err)
				return replyMessages, err
			}
			replyMessages = append(replyMessages, &messaging_api.TextMessage{
				Text: "登録しました",
			})
		}
	}

	return replyMessages, nil
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
			sourceId := getSourceId(e.Source)
			log.Printf("SourceId: %v", sourceId)

			// メッセージの種類によって処理を分岐
			switch message := e.Message.(type) {
			case webhook.TextMessageContent:
				replyMessages := []messaging_api.MessageInterface{
					&messaging_api.TextMessage{
						Text: message.Text,
					},
				}

				// TODO: ハンズオン前にコメントアウトする
				timestamp := e.Timestamp
				replyMessages, err = todoController(sourceId, fmt.Sprint(timestamp), message.Text)
				if err != nil {
					log.Println(err)
					return
				}

				if _, err = bot.ReplyMessage(
					&messaging_api.ReplyMessageRequest{
						ReplyToken: e.ReplyToken,
						Messages:   replyMessages,
					},
				); err != nil {
					log.Println(err)
					return
				}
			}
		case webhook.PostbackEvent:
			// 送信元のIDを取得
			sourceId := getSourceId(e.Source)
			log.Printf("SourceId: %v", sourceId)

			// 受け取ったデータを表示
			replyMessages, err := todoController(sourceId, e.Postback.Data)
			if err != nil {
				log.Println(err)
				return
			}

			if _, err = bot.ReplyMessage(
				&messaging_api.ReplyMessageRequest{
					ReplyToken: e.ReplyToken,
					Messages:   replyMessages,
				},
			); err != nil {
				log.Println(err)
				return
			}
		}
	}
}

func getSourceId(source webhook.SourceInterface) string {
	switch s := source.(type) {
	case webhook.UserSource:
		return s.UserId
	case webhook.GroupSource:
		return s.GroupId
	case webhook.RoomSource:
		return s.RoomId
	default:
		return ""
	}
}
