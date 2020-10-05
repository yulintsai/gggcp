# gggcp
gcp status monitor with telegram bot api

# docker-compose
```yaml
version: "3"

services:
  gggcp:
    image: rain123473/gggcp:latest
    restart: always
    command: ["./gggcp", "watch", "${telegram_bot_token}", "${telegram_chat_id}"]

```