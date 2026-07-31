# 🐕 downboy

**downboy** — is a lightweight, fast, and scalable cli tool written in go for parallel website uptime monitoring.
it checks server status in real time and sends alerts to communication channels when issues are detected.

---

## 🚀 features

* ⚡️ **scalable worker pool:** processes hundreds or thousands of url checks concurrently with configurable worker limits (`--concurrency`).
* ⚙️ **traffic & protocol optimization:** uses fast HTTP `HEAD` requests with automatic fallback to `GET` (for `405 Method Not Allowed` servers).
* ⏱️ **resilience & timeouts:** context-aware network checks with configurable timeouts (`--timeout`) and automatic retry backoff (`--retries`).
* 📦 **flexible configuration:** accepts URLs via CLI arguments or JSON configuration files (`--config`).
* 📑 **multi-channel notifications:** modular system (`MultiNotifier`) supporting console output, telegram bots, and discord webhooks.
* 📊 **single-pass CI mode:** run single-pass checks (`--once`) with formatted lipgloss summary reports and exit codes for CI/CD pipelines.
* 🔒 **secure secrets:** automatically loads secrets from environment variables and `.env` files.
* 🧪 **robust testing:** comprehensive test suite covering network checks, retries, and notification channels.

---

## 🛠 automation and building

the project uses a `Makefile` to manage build, test, and formatting workflows:

| command | description |
| :--- | :--- |
| `make build` | builds the executable binary for the current operating system |
| `make run` | compiles and immediately runs the tool |
| `make test` | runs all unit tests in the project |
| `make test-cover` | runs tests and generates an html code coverage report |
| `make lint` | runs `go vet ./...` static analysis |
| `make fmt` | automatically formats source code using `go fmt` |

---

## 💻 usage

### 1. running directly with a list of websites
```bash
./downboy google.com github.com yandex.ru
```

### 2. running in single-pass mode (for CI/CD)
```bash
./downboy --once google.com github.com
```

### 3. custom concurrency, timeout, and retry settings
```bash
./downboy -c 50 -t 3 -r 2 -i 15 google.com github.com
```

### 4. running with a configuration file
```bash
./downboy --config config.json
```

### 5. setting up notifications (Telegram & Discord)
To enable alerts, create a `.env` file in the project root:
```env
TELEGRAM_BOT_TOKEN=your_secret_bot_token
TELEGRAM_CHAT_ID=your_telegram_chat_id
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/your_webhook_id/token
```

