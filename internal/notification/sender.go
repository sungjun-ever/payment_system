package notification

import "context"

type Sender interface {
	Send(ctx context.Context, msg Message) error
}
