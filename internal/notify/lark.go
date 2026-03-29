package notify

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type LarkNotifier struct {
	webhookURL string
}

func NewLarkNotifier(webhookURL string) *LarkNotifier {
	return &LarkNotifier{webhookURL: webhookURL}
}

func (l *LarkNotifier) Send(title, content string) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": title + "\n" + content,
		},
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(l.webhookURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
