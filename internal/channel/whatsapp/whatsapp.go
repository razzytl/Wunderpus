package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
	"github.com/wonderpus/wonderpus/internal/agent"
	wunderpusTypes "github.com/wonderpus/wonderpus/internal/types"
)

// Channel implements the WhatsApp communication channel.
type Channel struct {
	sessionPath string
	manager     *agent.Manager
	client      *whatsmeow.Client
	qrChan      <-chan whatsmeow.QRChannelItem
}

// NewChannel creates a new WhatsApp channel.
func NewChannel(sessionPath string, manager *agent.Manager) *Channel {
	return &Channel{
		sessionPath: sessionPath,
		manager:     manager,
	}
}

// Name returns the channel name.
func (c *Channel) Name() string {
	return "whatsapp"
}

// Start launches the WhatsApp client.
func (c *Channel) Start(ctx context.Context) error {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New(ctx, "sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", c.sessionPath), dbLog)
	if err != nil {
		return fmt.Errorf("whatsapp: failed to init sqlstore: %w", err)
	}
	
	deviceRes, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp: failed to get device: %w", err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	c.client = whatsmeow.NewClient(deviceRes, clientLog)
	c.client.AddEventHandler(c.eventHandler)

	if c.client.Store.ID == nil {
		// New login
		qrChan, _ := c.client.GetQRChannel(ctx)
		err = c.client.Connect()
		if err != nil {
			return err
		}
		
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					// In a real CLI we would use a QR code library to print to terminal
					fmt.Printf("WhatsApp QR Code: %s\n", evt.Code)
					fmt.Println("Please scan this QR code in WhatsApp Linked Devices.")
				} else {
					fmt.Printf("WhatsApp Login Event: %s\n", evt.Event)
				}
			}
		}()
	} else {
		// Already logged in
		err = c.client.Connect()
		if err != nil {
			return err
		}
	}

	slog.Info("whatsapp channel starting")
	return nil
}

// Stop gracefully stops the client.
func (c *Channel) Stop() error {
	if c.client != nil {
		c.client.Disconnect()
	}
	return nil
}

func (c *Channel) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		if v.Info.IsFromMe {
			return
		}
		
		text := ""
		if v.Message.GetConversation() != "" {
			text = v.Message.GetConversation()
		} else if v.Message.GetExtendedTextMessage().GetText() != "" {
			text = v.Message.GetExtendedTextMessage().GetText()
		}

		if text == "" {
			return
		}

		sessionID := fmt.Sprintf("whatsapp_%s", v.Info.Sender.String())
		
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			respRes, err := c.manager.ProcessRequest(ctx, wunderpusTypes.UserMessage{
				SessionID: sessionID,
				Content:   text,
				ChannelID: v.Info.Chat.String(),
			})
			
			var reply string
			if err != nil {
				reply = "Error: " + err.Error()
			} else {
				reply = respRes.Content
			}

			c.client.SendMessage(ctx, v.Info.Chat, &waProto.Message{
				Conversation: proto.String(reply),
			})
		}()
	}
}
