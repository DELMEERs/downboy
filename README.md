# 🐕 downboy

**downboy** — is a lightweight and fast cli tool written in go for parallel website uptime monitoring.
it checks server status in real time and sends alerts to communication channels when issues are detected.

---

## 🚀 features

* ⚡️ **parallel monitoring:** every website check runs concurrently in a separate goroutine synchronized via `sync.WaitGroup`.
* ⚙️ **traffic optimization:** uses fast http `HEAD` requests instead of loading full web pages.
* 📦 **flexible configuration:** supports passing a list of urls directly via command-line arguments or loading them from a json file.
* 📑 **multi-channel notifications:** a modular system (`MultiNotifier`) that prints status to the console and sends urgent alerts to a telegram bot.
* 🔒 **secure secrets:** automatically loads tokens and keys from environment variables using `.env` files.
* 🧪 **reliability:** network check logic is completely isolated and covered by unit tests using mocks.

---

## 🛠 automation and building

the project uses a `Makefile` to manage build, test, and formatting workflows:

| command | description |
| :--- | :--- |
| `make build` | builds the executable binary for the current operating system |
| `make run` | compiles and immediately runs the tool |
| `make test` | runs all unit tests in the project |
| `make test-cover` | runs tests and generates an html code coverage report |
| `make fmt` | automatically formats source code using `go fmt` |

---

## 💻 usage

### 1. running directly with a list of websites
```bash
./downboy google.com github.com yandex.ru
```

### 2. running with a configuration file
```bash
./downboy --config config.json
```

### 3. setting up telegram notifications
to enable telegram alerts, create a .env file in the project root and specify your keys:
```TELEGRAM_BOT_TOKEN=your_secret_bot_token```
```TELEGRAM_CHAT_ID=your_telegram_chat_id```

## 🧬 project architecture
the codebase follows the standard go project layout:
```
├── cmd/
│   └── downboy/          # application entry point (main.go)
├── internal/
│   ├── checker/          # network request logic and unit tests
│   ├── config/           # configuration file and env variable parsing
│   └── notifier/         # alert dispatch interfaces and implementations
├── Makefile              # build automation and ci scripts
└── go.mod                # project dependency management
```
