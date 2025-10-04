# WhatsApp LLM Bot - Implementation Plan

## Project Overview
A Go application using hexagonal architecture that listens to WhatsApp group chats and responds using Local LLM models running on a remote Ollama server. Supports trigger words, reply detection, and webhook integration for external service calls.

## Technology Stack
- **Language**: Go
- **WhatsApp**: go.mau.fi/whatsmeow
- **LLM Integration**: langchaingo (Ollama)
- **Logging**: slog
- **Config**: YAML + Environment Variables
- **Containerization**: Docker (multi-stage build)
- **Build Tool**: Makefile
- **Admin UI**: Modern web interface

## Architecture (Hexagonal)

### Directory Structure
```
.
├── cmd/
│   └── bot/
│       └── main.go
├── internal/
│   ├── core/
│   │   ├── domain/
│   │   │   ├── models.go
│   │   │   └── ports.go
│   │   └── services/
│   │       ├── chat_service.go
│   │       └── llm_service.go
│   ├── adapters/
│   │   ├── primary/
│   │   │   ├── whatsapp/
│   │   │   │   └── listener.go
│   │   │   └── http/
│   │   │       ├── handlers.go
│   │   │       └── server.go
│   │   └── secondary/
│   │       ├── llm/
│   │       │   └── ollama.go
│   │       └── storage/
│   │           ├── memory.go
│   │           └── config.go
│   └── config/
│       └── config.go
├── web/
│   ├── static/
│   │   ├── css/
│   │   └── js/
│   └── templates/
│       └── admin.html
├── config.yaml
├── Dockerfile
├── Makefile
└── README.md
```

## Implementation Phases

### Phase 1: Project Setup & Core Domain
- [ ] Initialize Go module
- [ ] Setup directory structure
- [ ] Define domain models (Message, Group, Config)
- [ ] Define ports (interfaces) for:
  - MessageRepository
  - LLMProvider
  - WhatsAppClient
  - ConfigStore
  - GroupManager

### Phase 2: Configuration Management
- [ ] Implement YAML configuration structure:
  - Ollama server URL
  - Model name
  - Allowed WhatsApp groups
  - Admin UI port
  - Logging level
- [ ] Environment variable overrides
- [ ] Config validation
- [ ] Hot-reload capability for group changes

### Phase 3: WhatsApp Integration (go.mau.fi/whatsmeow)
- [ ] Implement WhatsApp client adapter
- [ ] QR code authentication flow
- [ ] Session persistence (LevelDB with volume mount)
- [ ] Group message listener
- [ ] Message sender
- [ ] Group filter based on config
- [ ] Event handlers (connect, disconnect, message)

### Phase 4: LLM Integration (langchaingo + Ollama)
- [ ] Implement Ollama adapter
- [ ] Connection to remote Ollama server
- [ ] Model selection from config
- [ ] Prompt engineering for chat context
- [ ] Streaming response handling
- [ ] Error handling and retries
- [ ] Context management (conversation history)

### Phase 5: Core Services
- [ ] ChatService:
  - Message routing
  - Group validation
  - LLM invocation
  - Response formatting
- [ ] LLMService:
  - Context building
  - Prompt construction
  - Response processing
- [ ] GroupService:
  - Group management
  - Allow/deny list
  - Config synchronization

### Phase 6: Admin UI
- [ ] REST API endpoints:
  - GET /api/groups (list all groups bot is in)
  - GET /api/config/allowed-groups
  - POST /api/config/allowed-groups (update)
  - GET /api/status (bot status, connection state)
  - POST /api/auth/qr (trigger QR code generation)
- [ ] Modern web interface:
  - Group selection checkboxes
  - Save to config.yaml
  - Real-time status display
  - QR code display for WhatsApp auth
  - Connection status indicator
- [ ] Static file serving
- [ ] WebSocket for real-time updates (optional)

### Phase 7: Logging & Monitoring
- [ ] Structured logging with slog
- [ ] Log levels (debug, info, warn, error)
- [ ] Request/response logging
- [ ] Error tracking
- [ ] Performance metrics

### Phase 8: Docker Containerization
- [ ] Multi-stage Dockerfile:
  - Stage 1: Build (golang:alpine)
  - Stage 2: Runtime (alpine:latest)
- [ ] Volume mounts:
  - `/data` for WhatsApp LevelDB
  - `/config` for config.yaml
- [ ] Environment variables
- [ ] Health check endpoint
- [ ] Expose ports (admin UI, health)

### Phase 9: Build Automation (Makefile)
- [ ] Targets:
  - `make build` - Build binary
  - `make test` - Run tests
  - `make lint` - Code linting
  - `make docker-build` - Build Docker image
  - `make docker-run` - Run container
  - `make clean` - Cleanup
  - `make dev` - Development mode
- [ ] Version tagging
- [ ] Cross-compilation support

### Phase 10: Testing & Documentation
- [ ] Unit tests for services
- [ ] Integration tests for adapters
- [ ] Mock implementations for testing
- [ ] README with:
  - Setup instructions
  - Configuration guide
  - Docker usage
  - QR code authentication steps
  - Troubleshooting
- [ ] API documentation

### Phase 11: Advanced Features
- [ ] Trigger words support (multiple trigger words)
- [ ] Reply detection (respond to replies to bot messages)
- [ ] Webhook integration:
  - Sub-trigger words for webhook routing
  - HTTP POST to external webhooks
  - Response forwarding back to WhatsApp
  - Admin UI for webhook management
  - Synchronous config updates

## Configuration Schema (config.yaml)

```yaml
app:
  name: "whatsapp-llm-bot"
  port: 8080
  log_level: "info"

whatsapp:
  session_path: "./whatsapp_session"
  allowed_groups:
    - "group1_jid@g.us"
    - "group2_jid@g.us"
  trigger_words:
    - "@bot"
    - "@sasi"

ollama:
  url: "http://ollama-server:11434"
  model: "llama2"
  temperature: 0.7
  timeout: 30s

storage:
  type: "memory"  # memory or future: redis, postgres

webhooks:
  - sub_trigger: "@family"
    url: "http://example.com/webhook/family"
  - sub_trigger: "@web"
    url: "http://example.com/webhook/web"
```

## Docker Usage

### Build:
```bash
make docker-build
```

### Run:
```bash
docker run -d \
  -p 8080:8080 \
  -v $(pwd)/data:/data \
  -v $(pwd)/config.yaml:/config/config.yaml \
  -e OLLAMA_URL=http://ollama-server:11434 \
  whatsapp-llm-bot:latest
```

## Key Implementation Details

### WhatsApp Authentication Flow:
1. First run: Display QR code via Admin UI or logs
2. Scan with WhatsApp mobile app
3. Session saved to LevelDB
4. Subsequent runs: Auto-authenticate from saved session

### Message Processing Flow:
1. Receive message from WhatsApp group
2. Check if group is in allowed list
3. Extract message content and context
4. Send to LLM service
5. Get LLM response
6. Format and send back to WhatsApp group

### Admin UI Group Management:
1. Bot joins/discovers groups
2. Admin UI displays all groups
3. Admin selects allowed groups
4. Update config.yaml
5. Reload group filter without restart


## Dependencies

```go
require (
    go.mau.fi/whatsmeow v0.0.0-latest
    github.com/tmc/langchaingo v0.0.0-latest
    gopkg.in/yaml.v3 v3.0.1
    github.com/gorilla/mux v1.8.0
    google.golang.org/protobuf v1.31.0
)
```

## Next Steps
1. Start with Phase 1 (Project Setup)
2. Implement core domain models and ports
3. Build adapters incrementally
4. Test each component before integration
5. Containerize once core features are stable
