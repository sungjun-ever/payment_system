package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SlackClient interface {
	Send(msg string) error
}

type slackClient struct {
	webhookURL string
}

func NewSlackClient(webhookURL string) SlackClient {
	return slackClient{
		webhookURL: webhookURL,
	}
}

func (s slackClient) Send(msg string) error {
	data := map[string]string{"text": msg}
	jsonData, _ := json.Marshal(data)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	bodyReader := bytes.NewReader(jsonData)

	req, err := http.NewRequest("POST", s.webhookURL, bodyReader)

	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Println("status code:", resp.StatusCode)
	fmt.Println("response body:", string(body))

	return nil
}
