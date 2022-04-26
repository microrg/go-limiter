package backend

type Backend interface {
	Bind(planID string, userId string) error
	Feature(featureID string, userID string) bool
	Increment(featureID string, userID string) error
	Decrement(featureID string, userID string) error
	Set(featureID string, userID string, value int) error
	FeatureMatrix() (*FeatureMatrix, error)
	Usage(userID string) (*FeatureUsage, error)
}

type FeatureMatrix struct {
	Plans []Plan `json:"plans"`
}

type Plan struct {
	PlanID   string    `json:"plan_id"`
	Features []Feature `json:"features"`
}

type Feature struct {
	FeatureID string  `json:"feature_id"`
	Type      string  `json:"type"`
	Value     int     `json:"value,omitempty"`
	Enabled   bool    `json:"enabled"`
	Soft      bool    `json:"soft"`
	Webhook   Webhook `json:"webhook"`
}

type Webhook struct {
	Enabled   bool    `json:"enabled"`
	Url       string  `json:"url,omitempty"`
	Token     string  `json:"token,omitempty"`
	Threshold float32 `json:"threshold,omitempty"`
	Payload   string  `json:"payload,omitempty"`
}

type FeatureUsage struct {
	UserID string         `json:"user_id"`
	PlanID string         `json:"plan_id"`
	Usage  map[string]int `json:"usage"`
}
