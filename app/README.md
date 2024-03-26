# Go言語を使ったLINE Bot
## ビルド
以下のコマンドでLINE BotのDockerイメージをビルド

```bash
docker build -t line-bot-hands-on .
```

## 実行方法
以下のコマンドでコンテナを立ち上げる

```bash
cp ../sample.env .env
docker run --rm -d --name line-bot-hands-on -p 8080:8080 --env-file ./.env line-bot-hands-on
```
