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
client, _ := limiter.New("my-s3-bucket", "my-project-id")

// Check if a feature is within limit
if client.Feature("my-feature", "user-id") {
    // Pass
}

// Increment usage by 1.
client.Increment("my-feature", "user-id")

// Set usage to some value.
client.Set("my-feature", "user-id", 5)
```