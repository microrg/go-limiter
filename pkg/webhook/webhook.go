package webhook

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	. "github.com/microrg/go-limiter/pkg/interface"
	"github.com/microrg/go-limiter/pkg/logging"
)

var logger logging.Logger

func sendWebhook(url string, token string, payload string, userID string, featureID string, usage int, limit int) error {
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

func ShouldSendWebhook(featureMatrix FeatureMatrix, featureUsage FeatureUsage, featureID string, userID string) {
	curr := featureUsage.Usage[featureID]
	for _, plan := range featureMatrix.Plans {
		for _, feature := range plan.Features {
			if feature.FeatureID == featureID {
				hook := feature.Webhook
				if hook.Enabled && float32(curr)/float32(feature.Value) > hook.Threshold {
					sendWebhook(hook.Url, hook.Token, hook.Payload, userID, featureID, curr, feature.Value)
				}
			}
		}
	}
}
