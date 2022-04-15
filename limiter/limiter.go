package limiter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

var logger Logger

type Limiter struct {
	S3Client  *s3.S3
	S3Bucket  string
	ProjectID string
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
	Usage  map[string]int `json:"usage"`
}

// New initializes the Limiter
func New(projectID string) *Limiter {
	return &Limiter{
		ProjectID: projectID,
	}
}

func (l *Limiter) WithAwsCredentials(bucket string, region string, accessKeyID string, secretAccessKey string) *Limiter {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess, &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})
	l.S3Bucket = bucket
	l.S3Client = svc
	return l
}

func (l *Limiter) putJsonObject(key string, payload interface{}) error {
	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	params := &s3.PutObjectInput{
		Bucket:      aws.String(l.S3Bucket),
		Key:         aws.String(key),
		Body:        aws.ReadSeekCloser(bytes.NewReader(p)),
		ContentType: aws.String("application/json"),
	}
	_, err = l.S3Client.PutObjectWithContext(context.Background(), params)
	if err != nil {
		return err
	}
	return nil
}

func (l *Limiter) getFeatureMatrix() (*FeatureMatrix, error) {
	result, err := l.S3Client.GetObjectWithContext(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(l.S3Bucket),
		Key:    aws.String(fmt.Sprintf("%s/feature_matrix.json", l.ProjectID)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchBucket:
				logger.Error("Bucket does not exist")
			case s3.ErrCodeNoSuchKey:
				logger.Error("Feature matrix does not exist")
			}
		}
		return nil, err
	}
	defer result.Body.Close()

	b, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	var featureMatrix FeatureMatrix
	if err = json.Unmarshal(b, &featureMatrix); err != nil {
		return nil, err
	}

	return &featureMatrix, nil
}

func (l *Limiter) getFeatureUsage(userID string) (*FeatureUsage, error) {
	result, err := l.S3Client.GetObjectWithContext(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(l.S3Bucket),
		Key:    aws.String(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				logger.Infof("Creating feature usage json for user %s", userID)
				featureUsage := &FeatureUsage{
					UserID: userID,
					Usage:  map[string]int{},
				}

				err = l.putJsonObject(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID), featureUsage)
				if err != nil {
					logger.Error("Failed to create feature usage: %s", err.Error())

					return nil, err
				}
				return featureUsage, nil
			}
		}
		return nil, err
	}
	defer result.Body.Close()

	b, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	var featureUsage FeatureUsage
	if err = json.Unmarshal(b, &featureUsage); err != nil {
		return nil, err
	}

	return &featureUsage, nil
}

func (l *Limiter) shouldSendWebhook(featureMatrix FeatureMatrix, featureUsage FeatureUsage, featureID string, userID string) {
	curr := featureUsage.Usage[featureID]
	for _, plan := range featureMatrix.Plans {
		for _, feature := range plan.Features {
			if feature.FeatureID == featureID {
				hook := feature.Webhook
				if hook.Enabled && float32(curr)/float32(feature.Value) > hook.Threshold {
					SendWebhook(hook.Url, hook.Token, hook.Payload, userID, featureID, curr, feature.Value)
				}
			}
		}
	}
}

// Feature checks if a feature can be accessed
func (l *Limiter) Feature(planID string, featureID string, userID string) bool {
	featureMatrix, err := l.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return false
	}
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return false
	}

	for _, plan := range featureMatrix.Plans {
		if plan.PlanID == planID {
			for _, feature := range plan.Features {
				if feature.FeatureID == featureID {
					if !feature.Enabled {
						logger.Infof("Feature %s disabled, allow.", featureID)
						return true
					}
					if feature.Type == "Boolean" && feature.Value == 1 {
						return true
					}
					if feature.Type == "Boolean" && feature.Value == 0 {
						return false
					}
					if feature.Soft {
						logger.Infof("Feature %s is soft, allow.", featureID)
						return true
					}
					if usage, ok := featureUsage.Usage[featureID]; ok {
						return usage < feature.Value
					}
					// Feature in plan but undefined on user
					return true
				}
			}
		}
	}

	logger.Infof("Feature %s not found in any plan, deny.", featureID)
	return false
}

// Increment increments feature usage by one
func (l *Limiter) Increment(featureID string, userID string) error {
	featureMatrix, err := l.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return err
	}
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	logger.Infof("Feature %s, User %s: Incrementing usage", featureID, userID)
	featureUsage.Usage[featureID] += 1

	err = l.putJsonObject(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID), featureUsage)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	l.shouldSendWebhook(*featureMatrix, *featureUsage, featureID, userID)

	return nil
}

// Decrement decrements feature usage by one
func (l *Limiter) Decrement(featureID string, userID string) error {
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	if featureUsage.Usage[featureID] > 0 {
		logger.Infof("Feature %s, User %s: Decrementing usage", featureID, userID)
		featureUsage.Usage[featureID] -= 1
	}

	err = l.putJsonObject(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID), featureUsage)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	return nil
}

// Set sets feature usage to some value
func (l *Limiter) Set(featureID string, userID string, value int) error {
	featureMatrix, err := l.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return err
	}
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	logger.Infof("Feature %s, User %s: Setting usage to %d", featureID, userID, value)
	featureUsage.Usage[featureID] = value

	err = l.putJsonObject(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID), featureUsage)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	l.shouldSendWebhook(*featureMatrix, *featureUsage, featureID, userID)

	return nil
}

// FeatureMatrix returns the feature matrix for the project
func (l *Limiter) FeatureMatrix() (*FeatureMatrix, error) {
	featureMatrix, err := l.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return nil, err
	}
	return featureMatrix, nil
}

// Usage returns the user's usage data
func (l *Limiter) Usage(userID string) (*FeatureUsage, error) {
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return nil, err
	}
	return featureUsage, nil
}
