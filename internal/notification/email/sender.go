package email

import (
	"context"
	"fmt"
	"order_system/internal/notification"
)

type Sender struct {
	client EmailClient
}

func NewSender(client EmailClient) *Sender {
	return &Sender{client}
}

func (s *Sender) Send(ctx context.Context, msg notification.Message) error {
	if msg.Channel != notification.ChannelEmail {
		return fmt.Errorf("invalid channel: %s", msg.Channel)
	}

	return s.client.Send(msg.To, msg.Title, msg.Body)
}
