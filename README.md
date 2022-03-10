# Limiter Go SDK

Limiter is a subscription limits management platform.

This Go package tracks usage and enforces limits within a Go application.

- Lightweight and fast
- Relies on S3 for scalability and data availability


## Installation

```
go get github.com/microorg/go-limiter
```

## Quick Usage

The following environment variables must be set:

- `AWS_ACCESS_KEY_ID`
- `AWS_SECRET_ACCESS_KEY`
- `AWS_DEFAULT_REGION`

```golang
// Initialize SDK with S3 bucket containing the feature matrix and usage tracking data
client := limiter.New("my-s3-bucket")

// Check if a feature is within limit
if client.Feature("my-feature", "user-id") {
    // Pass
}

// Emit feature usage event
client.Track("my-feature", "user-id")
```