package limiter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
	FeatureID string `json:"feature_id"`
	Type      string `json:"type"`
	Value     int    `json:"value,omitempty"`
	Enabled   bool   `json:"enabled"`
}

type FeatureUsage struct {
	UserID string         `json:"user_id"`
	Usage  map[string]int `json:"usage"`
}

// New initializes the Limiter
func New(bucket string, projectID string) (*Limiter, error) {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess, aws.NewConfig().WithRegion(os.Getenv("AWS_DEFAULT_REGION")))

	return &Limiter{
		S3Client:  svc,
		S3Bucket:  bucket,
		ProjectID: projectID,
	}, nil
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
				p, err := json.Marshal(featureUsage)
				if err != nil {
					return nil, err
				}
				params := &s3.PutObjectInput{
					Bucket:      aws.String(l.S3Bucket),
					Key:         aws.String(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID)),
					Body:        aws.ReadSeekCloser(bytes.NewReader(p)),
					ACL:         aws.String(s3.ObjectCannedACLPublicRead),
					ContentType: aws.String("application/json"),
				}
				_, err = l.S3Client.PutObjectWithContext(context.Background(), params)
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
						logger.Infof("Feature %s disabled, skipping check", featureID)
						return true
					}
					if feature.Type == "boolean" && feature.Value == 1 {
						return true
					}
					if usage, ok := featureUsage.Usage[featureID]; ok {
						return usage <= feature.Value
					}
				}
			}
		}
	}

	logger.Infof("Feature %s not found", featureID)
	return true
}

// Increment increments feature usage by one
func (l *Limiter) Increment(featureID string, userID string) error {
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	logger.Infof("Feature %s, User %s: Incrementing usage", featureID, userID)
	featureUsage.Usage[featureID] += 1

	p, err := json.Marshal(featureUsage)
	if err != nil {
		return err
	}
	params := &s3.PutObjectInput{
		Bucket:      aws.String(l.S3Bucket),
		Key:         aws.String(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID)),
		Body:        aws.ReadSeekCloser(bytes.NewReader(p)),
		ACL:         aws.String(s3.ObjectCannedACLPublicRead),
		ContentType: aws.String("application/json"),
	}
	_, err = l.S3Client.PutObjectWithContext(context.Background(), params)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	return nil
}

// Set sets feature usage to some value
func (l *Limiter) Set(featureID string, userID string, value int) error {
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	logger.Infof("Feature %s, User %s: Setting usage to %d", featureID, userID, value)
	featureUsage.Usage[featureID] = value

	p, err := json.Marshal(featureUsage)
	if err != nil {
		return err
	}
	params := &s3.PutObjectInput{
		Bucket:      aws.String(l.S3Bucket),
		Key:         aws.String(fmt.Sprintf("%s/users/%s.json", l.ProjectID, userID)),
		Body:        aws.ReadSeekCloser(bytes.NewReader(p)),
		ACL:         aws.String(s3.ObjectCannedACLPublicRead),
		ContentType: aws.String("application/json"),
	}
	_, err = l.S3Client.PutObjectWithContext(context.Background(), params)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

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
