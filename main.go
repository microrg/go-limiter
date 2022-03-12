package main

import (
	"fmt"

	"github.com/microrg/go-limiter/limiter"
)

func main() {
	client, err := limiter.New("limiter-test", "my-project")
	if err != nil {
		panic(err)
	}

	if client.Feature("p1f2", "5a8a1ca3-aee8-4a96-9bb4-673442728f2e") {
		fmt.Println("Pass")
	} else {
		fmt.Println("Fail")
	}

	err = client.Increment("p1f2", "5a8a1ca3-aee8-4a96-9bb4-673442728f2e")
	if err != nil {
		panic(err)
	}

	err = client.Set("p1f2", "5a8a1ca3-aee8-4a96-9bb4-673442728f2e", 5)
	if err != nil {
		panic(err)
	}
}
