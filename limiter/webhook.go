package limiter

import (
	"bytes"
	"fmt"
	"net/http"
)

func SendWebhook(url string, token string, userID string, featureID string, value int, limit int) error {
	logger.Infof("User %s, triggering webhook: %s", userID, url)

	var jsonStr = []byte(fmt.Sprintf(`{"user_id":%s,"feature_id":%s,"value":%d,"limit":%d}`, userID, featureID, value, limit))
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Errorf("User %s, webhook failed: %s", userID, err.Error())
		return err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("User %s, webhook failed: %s", userID, err.Error())
		return err
	}
	defer resp.Body.Close()

	logger.Infof("User %s, webhook triggered: %s", userID, url)

	return nil
}
