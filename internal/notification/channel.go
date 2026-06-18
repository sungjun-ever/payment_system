package notification

type Channel string

const (
	ChannelSlack Channel = "slack"
	ChannelEmail Channel = "email"
)
