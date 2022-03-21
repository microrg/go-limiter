# Limiter Go SDK

Limiter is a subscription limits management platform.

This Go package tracks usage and enforces limits within a Go application.

- Lightweight and fast
- Relies on S3 for scalability and data availability


## Installation

```
go get github.com/microrg/go-limiter
```

## Quick Usage

The following environment variables must be set:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_DEFAULT_REGION`

```golang
import (
    "github.com/microrg/go-limiter/limiter"
)

// Initialize SDK with S3 bucket containing the feature matrix and usage tracking data
client, _ := limiter.New("my-s3-bucket", "project-id")

// Check if a feature is within limit
if client.Feature("plan-name", "feature-name", "user-id") {
    // Pass
}

// Increment usage by 1.
client.Increment("feature-name", "user-id")

// Set usage to some value.
client.Set("feature-name", "user-id", 5)

// Get feature matrix for the project
client.FeatureMatrix("user-id")

// Get feature usage for a user
client.Usage("user-id")
```