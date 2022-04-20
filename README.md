# Limiter Go SDK

Limiter is a usage limits and quota management platform.

This Go package tracks usage and enforces limits within a Go application.


## Installation

```
go get github.com/microrg/go-limiter
```

## Quick Usage

### Default Backend

Initialize SDK with the managed storage backend

```golang
import (
    "github.com/microrg/go-limiter/limiter"
)

client := limiter.New("project-id").WithDefaultBackend("api-token")
```

### S3 Backend

Initialize SDK with a private S3 bucket storage

```golang
import (
    "github.com/microrg/go-limiter/limiter"
)

client := limiter.New("project-id").WithS3Backend("s3-bucket", "region", "access-key-id", "secret-access-key")
```

### Available Methods

// Check if a feature is within limit
if client.Feature("plan-name", "feature-name", "user-id") {
    // Pass
}

// Increment usage by 1.
client.Increment("feature-name", "user-id")

// Decrement usage by 1.
client.Decrement("feature-name", "user-id")

// Set usage to some value.
client.Set("feature-name", "user-id", 5)

// Get feature matrix for the project
client.FeatureMatrix("user-id")

// Get feature usage for a user
client.Usage("user-id")
```