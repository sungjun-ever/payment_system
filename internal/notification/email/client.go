package email

import "fmt"

type EmailClient interface {
	Send(to, subject, body string) (err error)
}

type emailClient struct {
	email string
	name  string
}

func NewEmailClient(email, name string) EmailClient {
	return emailClient{email, name}
}

func (e emailClient) Send(to, subject, body string) (err error) {
	fmt.Println("email send to: ", to, " subject: ", subject, " body: ", body)
	return nil
}
