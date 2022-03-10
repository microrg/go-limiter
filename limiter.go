package limiter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Limiter struct {
	S3Client *s3.S3
	S3Bucket string
}

type FeatureMatrix struct {
	Plans []Plan `json:"plans"`
}

type Plan struct {
	PlanID   string    `json:"plan_id"`
	Features []Feature `json:"plans"`
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
func New(bucket string) (*Limiter, error) {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess)

	return &Limiter{
		S3Client: svc,
		S3Bucket: bucket,
	}, nil
}

func (l *Limiter) getFeatureMatrix() (*FeatureMatrix, error) {
	result, err := l.S3Client.GetObjectWithContext(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(l.S3Bucket),
		Key:    aws.String("feature_matrix.json"),
	})
	if err != nil {
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
		Key:    aws.String(fmt.Sprintf("users/%s.json", userID)),
	})
	if err != nil {
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
func (l *Limiter) Feature(featureID string, userID string) (bool, error) {
	featureMatrix, err := l.getFeatureMatrix()
	if err != nil {
		// Deny by default
		return false, fmt.Errorf("unable to retrieve feature matrix")
	}
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		// Create if doesnt exist
		featureUsage = &FeatureUsage{
			UserID: userID,
			Usage:  map[string]int{},
		}
		p, err := json.Marshal(featureUsage)
		if err != nil {
			// Deny by default
			return false, err
		}
		params := &s3.PutObjectInput{
			Bucket: aws.String(l.S3Bucket),
			Key:    aws.String(fmt.Sprintf("users/%s.json", userID)),
			Body:   aws.ReadSeekCloser(bytes.NewReader(p)),
		}
		_, err = l.S3Client.PutObjectWithContext(context.Background(), params)
		if err != nil {
			// Deny by default
			return false, err
		}
	}
	for _, plan := range featureMatrix.Plans {
		for _, feature := range plan.Features {
			if feature.FeatureID == featureID {
				if !feature.Enabled {
					return true, nil
				}
				if feature.Type == "boolean" && feature.Value == 1 {
					return true, nil
				}
				if usage, ok := featureUsage.Usage[featureID]; ok {
					if usage > feature.Value {
						return false, nil
					}
					return true, nil
				}
			}
		}
	}

	// Feature not in matrix, allow.
	return true, nil
}

// Track emits a feature usage event
func (l *Limiter) Track(featureID string, userID string) error {
	featureUsage, err := l.getFeatureUsage(userID)
	if err != nil {
		// Create if doesnt exist
		featureUsage = &FeatureUsage{
			UserID: userID,
			Usage:  map[string]int{},
		}
		p, err := json.Marshal(featureUsage)
		if err != nil {
			return err
		}
		params := &s3.PutObjectInput{
			Bucket: aws.String(l.S3Bucket),
			Key:    aws.String(fmt.Sprintf("users/%s.json", userID)),
			Body:   aws.ReadSeekCloser(bytes.NewReader(p)),
		}
		_, err = l.S3Client.PutObjectWithContext(context.Background(), params)
		if err != nil {
			return err
		}
	}

	// Increment usage
	if _, ok := featureUsage.Usage[featureID]; ok {
		featureUsage.Usage[featureID] += 1
	}

	// Reupload object
	p, err := json.Marshal(featureUsage)
	if err != nil {
		return err
	}
	params := &s3.PutObjectInput{
		Bucket: aws.String(l.S3Bucket),
		Key:    aws.String(fmt.Sprintf("users/%s.json", userID)),
		Body:   aws.ReadSeekCloser(bytes.NewReader(p)),
	}
	_, err = l.S3Client.PutObjectWithContext(context.Background(), params)
	if err != nil {
		return err
	}

	return nil
}
