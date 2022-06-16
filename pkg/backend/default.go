package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	. "github.com/microrg/go-limiter/pkg/interface"
)

const (
	apiV1 = "https://www.applimiter.com/api/v1"
)

type DefaultBackend struct {
	ApiToken   string
	BackendURL string
	ProjectID  string
}

type FeatureResp struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason"`
}

// NewDefaultBackend initializes the S3 Backend
func NewDefaultBackend(projectID string, apiToken string) Backend {
	return &DefaultBackend{
		ApiToken:   apiToken,
		BackendURL: apiV1,
		ProjectID:  projectID,
	}
}

func makeHttpRequest(url string, apiToken string, payload map[string]interface{}) (*http.Response, error) {
	jsonStr, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", apiToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (b *DefaultBackend) Bind(planID string, userID string) error {
	payload := map[string]interface{}{"project_id": b.ProjectID, "plan_id": planID, "user_id": userID}
	resp, err := makeHttpRequest(fmt.Sprintf("%s/%s", b.BackendURL, "/bind"), b.ApiToken, payload)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (b *DefaultBackend) Feature(featureID string, userID string) bool {
	payload := map[string]interface{}{"project_id": b.ProjectID, "feature_id": featureID, "user_id": userID}
	resp, err := makeHttpRequest(fmt.Sprintf("%s/%s", b.BackendURL, "/feature"), b.ApiToken, payload)
	if err != nil {
		logger.Error(err.Error())
		return false
	}
	defer resp.Body.Close()

	featureResp := &FeatureResp{}
	err = json.NewDecoder(resp.Body).Decode(featureResp)
	if err != nil {
		logger.Info(err.Error())
		return false
	}
	if featureResp.Reason != "" {
		logger.Info(featureResp.Reason)
	}
	return featureResp.Allow
}

func (b *DefaultBackend) Increment(featureID string, userID string, value int) error {
	if value == 0 {
		value = 1
	}
	payload := map[string]interface{}{"project_id": b.ProjectID, "feature_id": featureID, "user_id": userID, "value": value}
	resp, err := makeHttpRequest(fmt.Sprintf("%s/%s", b.BackendURL, "/increment"), b.ApiToken, payload)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (b *DefaultBackend) Decrement(featureID string, userID string, value int) error {
	if value == 0 {
		value = 1
	}
	payload := map[string]interface{}{"project_id": b.ProjectID, "feature_id": featureID, "user_id": userID, "value": value}
	resp, err := makeHttpRequest(fmt.Sprintf("%s/%s", b.BackendURL, "/decrement"), b.ApiToken, payload)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (b *DefaultBackend) Set(featureID string, userID string, value int) error {
	payload := map[string]interface{}{"project_id": b.ProjectID, "feature_id": featureID, "user_id": userID, "value": strconv.Itoa(value)}
	resp, err := makeHttpRequest(fmt.Sprintf("%s/%s", b.BackendURL, "/set"), b.ApiToken, payload)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (b *DefaultBackend) FeatureMatrix() (*FeatureMatrix, error) {
	payload := map[string]interface{}{"project_id": b.ProjectID}
	resp, err := makeHttpRequest(fmt.Sprintf("%s/%s", b.BackendURL, "/feature-matrix"), b.ApiToken, payload)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	featureMatrixResp := &FeatureMatrix{}
	err = json.NewDecoder(resp.Body).Decode(featureMatrixResp)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return featureMatrixResp, nil
}

func (b *DefaultBackend) Usage(userID string) (*FeatureUsage, error) {
	payload := map[string]interface{}{"project_id": b.ProjectID, "user_id": userID}
	resp, err := makeHttpRequest(fmt.Sprintf("%s/%s", b.BackendURL, "/usage"), b.ApiToken, payload)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	defer resp.Body.Close()

	featureUsageResp := &FeatureUsage{}
	err = json.NewDecoder(resp.Body).Decode(featureUsageResp)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return featureUsageResp, nil
}
