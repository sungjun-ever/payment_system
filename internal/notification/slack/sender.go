package slack

import (
	"context"
	"fmt"
	"order_system/internal/notification"
)

type Sender struct {
	client SlackClient
}

func NewSender(client SlackClient) *Sender {
	return &Sender{client}
}

func (s *Sender) Send(ctx context.Context, msg notification.Message) error {
	if msg.Channel != notification.ChannelSlack {
		return fmt.Errorf("invalid channel: %s", msg.Channel)
	}

	return s.client.Send(msg.Body)
}
