# Budget Splitter Assistant

<p align="center">
    <img src="https://github.com/tiroq/bsa/blob/main/logo.png?raw=true" width="50%" alt="Budget Splitter Assistant Bot">
</p>

A Telegram bot written in Go to split a budget based on user-defined categories.
It supports uploading categories in JSON or YAML format and processing numeric inputs to calculate splits.
Feedback sent via `/feedback` is forwarded to the admin.

## Environment Variables

Set variables via `.env` file

- `TELEGRAM_BOT_TOKEN`: Your Telegram bot token.
- `ADMIN_TELEGRAM_ID`: Telegram user ID of the admin who will receive feedback.

## Usage

1. Set the required environment variables via `.env` file.

2. Build the bot:

  ```bash
  make build
  ```

3. Run the bot:

  ```bash
  make run
  ```

## Docker

To build and run using Docker Compose:

```bash
make docker-up
```
