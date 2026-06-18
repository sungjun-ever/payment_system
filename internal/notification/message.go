package notification

type Message struct {
	Channel Channel
	To      string
	Title   string
	Body    string
}
