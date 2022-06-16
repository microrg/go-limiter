package limiter

import (
	"github.com/microrg/go-limiter/pkg/backend"
	. "github.com/microrg/go-limiter/pkg/interface"
)

type Limiter struct {
	Backend   Backend
	ProjectID string
}

// New initializes the Limiter
func New(projectID string) *Limiter {
	return &Limiter{
		ProjectID: projectID,
	}
}

func (l *Limiter) WithDefaultBackend(apiToken string) *Limiter {
	l.Backend = backend.NewDefaultBackend(l.ProjectID, apiToken)
	return l
}

func (l *Limiter) WithS3Backend(bucket string, region string, accessKeyID string, secretAccessKey string) *Limiter {
	l.Backend = backend.NewS3Backend(l.ProjectID, bucket, region, accessKeyID, secretAccessKey)
	return l
}

// Bind a user to a plan
func (l *Limiter) Bind(planID string, userID string) error {
	return l.Backend.Bind(planID, userID)
}

// Feature checks if a feature can be accessed
func (l *Limiter) Feature(featureID string, userID string) bool {
	return l.Backend.Feature(featureID, userID)
}

// Increment increments feature usage by one
func (l *Limiter) Increment(featureID string, userID string, value int) error {
	return l.Backend.Increment(featureID, userID, value)
}

// Decrement decrements feature usage by one
func (l *Limiter) Decrement(featureID string, userID string, value int) error {
	return l.Backend.Decrement(featureID, userID, value)
}

// Set sets feature usage to some value
func (l *Limiter) Set(featureID string, userID string, value int) error {
	return l.Backend.Set(featureID, userID, value)
}

// FeatureMatrix returns the feature matrix for the project
func (l *Limiter) FeatureMatrix() (*FeatureMatrix, error) {
	return l.Backend.FeatureMatrix()
}

// Usage returns the user's usage data
func (l *Limiter) Usage(userID string) (*FeatureUsage, error) {
	return l.Backend.Usage(userID)
}
