# WhatsApp LLM Bot

A Go application built with hexagonal architecture that listens to WhatsApp group chats and responds using Local LLM models running on a remote Ollama server.

## Features

- 🏗️ **Hexagonal Architecture** - Clean separation of concerns with well-defined ports and adapters
- 📱 **WhatsApp Integration** - Connect to WhatsApp groups using whatsmeow library
- 🤖 **LLM Integration** - Uses langchaingo to connect with Ollama models
- 🎨 **Modern Admin UI** - Web interface for group management and configuration
- 🔐 **QR Code Authentication** - Easy WhatsApp login via QR code
- 📝 **Structured Logging** - Built-in logging with slog
- 🐳 **Docker Support** - Multi-stage Docker build with volume mounts
- 🛠️ **Build Automation** - Comprehensive Makefile for development and deployment

## Architecture

```
.
├── cmd/bot/                    # Application entry point
├── internal/
│   ├── core/
│   │   ├── domain/            # Domain models and interfaces (ports)
│   │   └── services/          # Business logic
│   ├── adapters/
│   │   ├── primary/           # Inbound adapters
│   │   │   ├── whatsapp/     # WhatsApp client
│   │   │   └── http/         # REST API & Admin UI
│   │   └── secondary/         # Outbound adapters
│   │       ├── llm/          # Ollama LLM provider
│   │       └── storage/      # Message repository
│   └── config/               # Configuration management
├── web/                       # Admin UI assets
│   ├── static/               # CSS, JS
│   └── templates/            # HTML templates
├── config.yaml               # Configuration file
├── Dockerfile                # Multi-stage Docker build
└── Makefile                  # Build automation
```

## Quick Start

### Prerequisites

- Go 1.23 or higher (dependencies require Go 1.24+ toolchain, which will be auto-downloaded)
- Ollama running on a server (local or remote)
- Docker (optional, for containerized deployment)

### Local Development

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd whatsapp-llm-bot
   ```

2. **Install dependencies**
   ```bash
   make deps
   ```

3. **Configure the application**

   Edit `config.yaml`:
   ```yaml
   app:
     name: "whatsapp-llm-bot"
     port: 8080
     log_level: "info"

   whatsapp:
     session_path: "./whatsapp_session"
     allowed_groups: []

   ollama:
     url: "http://localhost:11434"
     model: "llama2"
     temperature: 0.7
     timeout: "30s"

   storage:
     type: "memory"
   ```

4. **Build and run**
   ```bash
   make build
   make run
   ```

5. **Access the Admin UI**

   Open http://localhost:8080 in your browser

6. **Authenticate WhatsApp**

   - The first time you run the bot, a QR code will be displayed in the Admin UI
   - Scan it with your WhatsApp mobile app (WhatsApp > Settings > Linked Devices)
   - The session will be saved for future use

7. **Configure allowed groups**

   - Navigate to the Admin UI
   - Select the groups you want the bot to listen to
   - Click "Save Changes"

### Docker Deployment

1. **Build the Docker image**
   ```bash
   make docker-build
   ```

2. **Run the container**
   ```bash
   docker run -d \
     --name whatsapp-llm-bot \
     -p 8080:8080 \
     -v $(pwd)/data:/data \
     -v $(pwd)/config.yaml:/config/config.yaml \
     -e OLLAMA_URL=http://your-ollama-server:11434 \
     whatsapp-llm-bot:latest
   ```

   Or use the Makefile:
   ```bash
   make docker-run
   ```

3. **View logs**
   ```bash
   make docker-logs
   ```

4. **Stop the container**
   ```bash
   make docker-stop
   ```

## Configuration

### Environment Variables

The following environment variables can override config file settings:

- `CONFIG_PATH` - Path to config file (default: `config.yaml`)
- `APP_PORT` - HTTP server port
- `APP_LOG_LEVEL` - Log level (debug, info, warn, error)
- `WHATSAPP_SESSION_PATH` - WhatsApp session directory
- `OLLAMA_URL` - Ollama server URL
- `OLLAMA_MODEL` - Ollama model name
- `OLLAMA_TEMPERATURE` - LLM temperature (0.0-2.0)

### Config File Structure

See `config.yaml` for the complete configuration structure.

## Development

### Available Make Targets

```bash
make help              # Show all available targets
make build             # Build the application
make test              # Run tests
make test-coverage     # Run tests with coverage report
make lint              # Run linter
make fmt               # Format code
make docker-build      # Build Docker image
make docker-run        # Run Docker container
make clean             # Clean build artifacts
make dev               # Run with auto-reload (requires air)
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run vet
make vet

# Run linter (requires golangci-lint)
make lint
```

## API Documentation

### REST Endpoints

- `GET /api/groups` - List all WhatsApp groups
- `GET /api/config/allowed-groups` - Get allowed groups
- `POST /api/config/allowed-groups` - Update allowed groups
- `GET /api/status` - Get bot status and authentication state
- `GET /api/auth/qr` - Get QR code for authentication
- `GET /api/health` - Health check endpoint

### Request/Response Examples

**Get all groups:**
```bash
curl http://localhost:8080/api/groups
```

**Update allowed groups:**
```bash
curl -X POST http://localhost:8080/api/config/allowed-groups \
  -H "Content-Type: application/json" \
  -d '{"groups": ["group1@g.us", "group2@g.us"]}'
```

## WhatsApp Authentication

### First Time Setup

1. Start the application
2. Open the Admin UI at http://localhost:8080
3. A QR code will be displayed
4. Open WhatsApp on your phone
5. Go to Settings > Linked Devices > Link a Device
6. Scan the QR code
7. The session will be saved in the configured session path

### Session Persistence

- WhatsApp session is stored in SQLite database
- Default location: `./whatsapp_session/` (or `/data/whatsapp_session` in Docker)
- Session persists across restarts
- Use volume mount in Docker to preserve session data

## Troubleshooting

### WhatsApp Connection Issues

- Ensure QR code is scanned within the timeout period
- Check that session directory has proper permissions
- Verify WhatsApp on phone is connected to internet

### LLM Not Responding

- Verify Ollama server is running and accessible
- Check `OLLAMA_URL` configuration
- Ensure the specified model is available in Ollama
- Check logs for connection errors

### Docker Issues

- Ensure volumes are properly mounted
- Check that ports are not already in use
- Verify config file path in volume mount
- Check container logs: `docker logs whatsapp-llm-bot`

### Permission Issues

- Ensure data directories have correct permissions
- In Docker, files are owned by user ID 1000
- Check that config file is readable

## Project Structure

- **Domain Models** (`internal/core/domain/models.go`) - Core business entities
- **Ports** (`internal/core/domain/ports.go`) - Interface definitions
- **Services** (`internal/core/services/`) - Business logic implementation
- **Adapters** (`internal/adapters/`) - External integrations
- **Configuration** (`internal/config/`) - Config management
- **Main** (`cmd/bot/main.go`) - Application bootstrap

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests and linting
5. Submit a pull request

## License

[Your License Here]

## Acknowledgments

- [whatsmeow](https://github.com/tulir/whatsmeow) - WhatsApp Web API implementation
- [langchaingo](https://github.com/tmc/langchaingo) - LangChain Go implementation
- [Ollama](https://ollama.ai/) - Local LLM runtime
