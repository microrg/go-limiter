package limiter

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

func SendWebhook(url string, token string, payload string, userID string, featureID string, usage int, limit int) error {
	logger.Infof("User %s, triggering webhook: %s", userID, url)
	payload = strings.ReplaceAll(payload, "{{user_id}}", userID)
	payload = strings.ReplaceAll(payload, "{{feature_id}}", featureID)
	payload = strings.ReplaceAll(payload, `"{{usage}}"`, strconv.Itoa(usage))
	payload = strings.ReplaceAll(payload, `"{{limit}}"`, strconv.Itoa(limit))
	var jsonStr = []byte(payload)
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
