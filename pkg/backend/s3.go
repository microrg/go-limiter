package backend

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

	. "github.com/microrg/go-limiter/pkg/interface"
	"github.com/microrg/go-limiter/pkg/webhook"
)

type S3Backend struct {
	S3Client  *s3.S3
	S3Bucket  string
	ProjectID string
}

// NewS3Backend initializes the S3 Backend
func NewS3Backend(projectID string, bucket string, region string, accessKeyID string, secretAccessKey string) Backend {
	sess := session.Must(session.NewSession())
	svc := s3.New(sess, &aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, ""),
	})

	return &S3Backend{
		S3Bucket:  bucket,
		S3Client:  svc,
		ProjectID: projectID,
	}
}

func (b *S3Backend) putJsonObject(key string, payload interface{}) error {
	p, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	params := &s3.PutObjectInput{
		Bucket:      aws.String(b.S3Bucket),
		Key:         aws.String(key),
		Body:        aws.ReadSeekCloser(bytes.NewReader(p)),
		ContentType: aws.String("application/json"),
	}
	_, err = b.S3Client.PutObjectWithContext(context.Background(), params)
	if err != nil {
		return err
	}
	return nil
}

func (b *S3Backend) getFeatureMatrix() (*FeatureMatrix, error) {
	result, err := b.S3Client.GetObjectWithContext(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(b.S3Bucket),
		Key:    aws.String(fmt.Sprintf("%s/feature_matrix.json", b.ProjectID)),
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

	bt, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	var featureMatrix FeatureMatrix
	if err = json.Unmarshal(bt, &featureMatrix); err != nil {
		return nil, err
	}

	return &featureMatrix, nil
}

func (b *S3Backend) getFeatureUsage(userID string) (*FeatureUsage, error) {
	result, err := b.S3Client.GetObjectWithContext(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(b.S3Bucket),
		Key:    aws.String(fmt.Sprintf("%s/users/%s.json", b.ProjectID, userID)),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case s3.ErrCodeNoSuchKey:
				logger.Infof("Creating feature usage json for user %s", userID)
				featureUsage := &FeatureUsage{
					UserID: userID,
					PlanID: "",
					Usage:  map[string]int{},
				}

				err = b.putJsonObject(fmt.Sprintf("%s/users/%s.json", b.ProjectID, userID), featureUsage)
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

	bt, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}
	var featureUsage FeatureUsage
	if err = json.Unmarshal(bt, &featureUsage); err != nil {
		return nil, err
	}

	return &featureUsage, nil
}

func (b *S3Backend) Bind(planID string, userID string) error {
	featureUsage, err := b.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	logger.Infof("Plan %s, User %s: Binding user to plan", planID, userID)
	featureUsage.PlanID = planID

	err = b.putJsonObject(fmt.Sprintf("%s/users/%s.json", b.ProjectID, userID), featureUsage)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	return nil
}

func (b *S3Backend) Feature(featureID string, userID string) bool {
	featureMatrix, err := b.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return false
	}
	featureUsage, err := b.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return false
	}

	for _, plan := range featureMatrix.Plans {
		if plan.PlanID == featureUsage.PlanID {
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

	logger.Infof("Feature %s not found in %s plan, deny.", featureID, featureUsage.PlanID)
	return false
}

func (b *S3Backend) Increment(featureID string, userID string) error {
	featureMatrix, err := b.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return err
	}
	featureUsage, err := b.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	logger.Infof("Feature %s, User %s: Incrementing usage", featureID, userID)
	featureUsage.Usage[featureID] += 1

	err = b.putJsonObject(fmt.Sprintf("%s/users/%s.json", b.ProjectID, userID), featureUsage)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	webhook.ShouldSendWebhook(*featureMatrix, *featureUsage, featureID, userID)

	return nil
}

func (b *S3Backend) Decrement(featureID string, userID string) error {
	featureUsage, err := b.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	if featureUsage.Usage[featureID] > 0 {
		logger.Infof("Feature %s, User %s: Decrementing usage", featureID, userID)
		featureUsage.Usage[featureID] -= 1
	}

	err = b.putJsonObject(fmt.Sprintf("%s/users/%s.json", b.ProjectID, userID), featureUsage)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	return nil
}

func (b *S3Backend) Set(featureID string, userID string, value int) error {
	featureMatrix, err := b.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return err
	}
	featureUsage, err := b.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return err
	}

	logger.Infof("Feature %s, User %s: Setting usage to %d", featureID, userID, value)
	featureUsage.Usage[featureID] = value

	err = b.putJsonObject(fmt.Sprintf("%s/users/%s.json", b.ProjectID, userID), featureUsage)
	if err != nil {
		logger.Error("Failed to update feature usage: %s", err.Error())
		return err
	}

	webhook.ShouldSendWebhook(*featureMatrix, *featureUsage, featureID, userID)

	return nil
}

func (b *S3Backend) FeatureMatrix() (*FeatureMatrix, error) {
	featureMatrix, err := b.getFeatureMatrix()
	if err != nil {
		logger.Errorf("Failed to fetch feature matrix: %s", err.Error())
		return nil, err
	}
	return featureMatrix, nil
}

func (b *S3Backend) Usage(userID string) (*FeatureUsage, error) {
	featureUsage, err := b.getFeatureUsage(userID)
	if err != nil {
		logger.Errorf("Failed to fetch feature usage: %s", err.Error())
		return nil, err
	}
	return featureUsage, nil
}
