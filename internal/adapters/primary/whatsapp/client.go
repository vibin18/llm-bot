package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	"github.com/skip2/go-qrcode"
	"github.com/vibin/whatsapp-llm-bot/internal/core/domain"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	waProto "go.mau.fi/whatsmeow/binary/proto"
)

// Client implements WhatsAppClient interface
type Client struct {
	client          *whatsmeow.Client
	sessionPath     string
	allowedGroups   map[string]bool
	messageHandlers []func(*domain.Message)
	mu              sync.RWMutex
	qrChan          chan string
	logger          waLog.Logger
}

// NewClient creates a new WhatsApp client
func NewClient(sessionPath string, allowedGroups []string, logger waLog.Logger) (*Client, error) {
	allowed := make(map[string]bool)
	for _, group := range allowedGroups {
		allowed[group] = true
	}

	return &Client{
		sessionPath:   sessionPath,
		allowedGroups: allowed,
		qrChan:        make(chan string, 1),
		logger:        logger,
	}, nil
}

// Start initializes and starts the WhatsApp client
func (c *Client) Start(ctx context.Context) error {
	// Ensure session directory exists
	if err := os.MkdirAll(c.sessionPath, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	// Setup database for session storage
	dbPath := fmt.Sprintf("%s/whatsapp.db", c.sessionPath)
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), c.logger)
	if err != nil {
		return fmt.Errorf("failed to create store: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	c.client = whatsmeow.NewClient(deviceStore, c.logger)
	c.client.AddEventHandler(c.eventHandler)

	// Connect
	if c.client.Store.ID == nil {
		// No existing session, need to pair
		qrChan, err := c.client.GetQRChannel(ctx)
		if err != nil {
			return fmt.Errorf("failed to get QR channel: %w", err)
		}

		err = c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		// Handle QR code in background
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					c.logger.Infof("QR code received, scan with WhatsApp to authenticate")

					// Print QR code to terminal
					qr, err := qrcode.New(evt.Code, qrcode.Medium)
					if err == nil {
						fmt.Println("\n" + qr.ToSmallString(false) + "\n")
					}

					c.qrChan <- evt.Code
				}
			}
		}()
	} else {
		// Existing session, just connect
		err = c.client.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
	}

	return nil
}

// Stop disconnects the WhatsApp client
func (c *Client) Stop(ctx context.Context) error {
	if c.client != nil {
		c.client.Disconnect()
	}
	return nil
}

// SendMessage sends a message to a WhatsApp group
func (c *Client) SendMessage(ctx context.Context, groupJID, message string) error {
	if c.client == nil {
		return fmt.Errorf("client not initialized")
	}

	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return fmt.Errorf("invalid JID: %w", err)
	}

	msg := &waProto.Message{
		Conversation: proto.String(message),
	}

	_, err = c.client.SendMessage(ctx, jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// GetGroups returns all groups the bot is part of
func (c *Client) GetGroups(ctx context.Context) ([]*domain.Group, error) {
	if c.client == nil {
		return nil, fmt.Errorf("client not initialized")
	}

	groups, err := c.client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}

	result := make([]*domain.Group, 0, len(groups))
	for _, group := range groups {
		c.mu.RLock()
		isAllowed := c.allowedGroups[group.JID.String()]
		c.mu.RUnlock()

		// Get full group info to fetch the name
		groupName := group.Name
		if groupName == "" {
			// Try to get group info for the name
			groupInfo, err := c.client.GetGroupInfo(group.JID)
			if err == nil && groupInfo != nil {
				groupName = groupInfo.Name
			}
		}

		// If still empty, use a fallback based on JID
		if groupName == "" {
			groupName = "Group " + group.JID.User
		}

		result = append(result, &domain.Group{
			JID:          group.JID.String(),
			Name:         groupName,
			IsAllowed:    isAllowed,
			Participants: len(group.Participants),
		})
	}

	return result, nil
}

// GetAuthStatus returns the current authentication status
func (c *Client) GetAuthStatus(ctx context.Context) (*domain.AuthStatus, error) {
	status := &domain.AuthStatus{
		IsAuthenticated: false,
	}

	if c.client != nil && c.client.IsConnected() && c.client.Store.ID != nil {
		status.IsAuthenticated = true
	} else {
		// Try to get QR code if available (non-blocking)
		select {
		case qr := <-c.qrChan:
			status.QRCode = c.generateQRDataURL(qr)
			// Put it back for next request
			go func() { c.qrChan <- qr }()
		default:
			// No QR code available yet
		}
	}

	return status, nil
}

// OnMessage registers a message handler
func (c *Client) OnMessage(handler func(*domain.Message)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageHandlers = append(c.messageHandlers, handler)
}

// eventHandler handles WhatsApp events
func (c *Client) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		// Only process group messages
		if !v.Info.IsGroup {
			return
		}

		groupJID := v.Info.Chat.String()

		// Check if group is allowed
		c.mu.RLock()
		isAllowed := c.allowedGroups[groupJID]
		c.mu.RUnlock()

		if !isAllowed {
			return
		}

		// Extract message content
		var content string
		var isReplyToBot bool

		if v.Message.GetConversation() != "" {
			content = v.Message.GetConversation()
		} else if v.Message.GetExtendedTextMessage() != nil {
			extMsg := v.Message.GetExtendedTextMessage()
			content = extMsg.GetText()

			// Check if this is a reply to bot's message
			if extMsg.ContextInfo != nil && extMsg.ContextInfo.StanzaID != nil {
				// Check if the quoted message is from the bot
				quotedMsg := extMsg.ContextInfo.GetParticipant()
				if quotedMsg != "" && quotedMsg == c.client.Store.ID.String() {
					isReplyToBot = true
				}
			}
		}

		if content == "" {
			return
		}

		// Create domain message
		msg := &domain.Message{
			ID:           v.Info.ID,
			GroupJID:     groupJID,
			Sender:       v.Info.Sender.String(),
			Content:      content,
			Timestamp:    v.Info.Timestamp,
			IsFromBot:    false,
			IsReplyToBot: isReplyToBot,
		}

		// Call all registered handlers
		c.mu.RLock()
		handlers := c.messageHandlers
		c.mu.RUnlock()

		for _, handler := range handlers {
			go handler(msg)
		}

	case *events.Connected:
		c.logger.Infof("Connected to WhatsApp")

	case *events.Disconnected:
		c.logger.Infof("Disconnected from WhatsApp")
	}
}

// UpdateAllowedGroups updates the list of allowed groups
func (c *Client) UpdateAllowedGroups(groups []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.allowedGroups = make(map[string]bool)
	for _, group := range groups {
		c.allowedGroups[group] = true
	}
}

// generateQRDataURL converts QR code to data URL
func (c *Client) generateQRDataURL(code string) string {
	// Generate QR code as PNG image
	png, err := qrcode.Encode(code, qrcode.Medium, 256)
	if err != nil {
		c.logger.Errorf("Failed to generate QR code: %v", err)
		// Fallback to text
		encoded := base64.StdEncoding.EncodeToString([]byte(code))
		return fmt.Sprintf("data:text/plain;base64,%s", encoded)
	}

	// Convert to base64 data URL
	encoded := base64.StdEncoding.EncodeToString(png)
	return fmt.Sprintf("data:image/png;base64,%s", encoded)
}
