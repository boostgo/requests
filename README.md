# `github.com/boostgo/requests`

# Get started

```go
package main

import (
	"context"
	"fmt"
	
	"github.com/boostgo/requests"
)

func main() {
	ctx := context.Background()

	// raw way

	response, err := requests.Get(ctx, "https://google.com")
	if err != nil {
		panic(err)
	}

	fmt.Println("status:", response.Status())          // 200 OK
	fmt.Println("status code:", response.StatusCode()) // 200
	response.Raw()                                     // *http.Response
	fmt.Printf("body: %s\n", response.BodyRaw())       // <!doctype html><html....

	// via client

	client := requests.New()

	response, err = client.
		R(ctx).
		GET("https://google.com")
	if err != nil {
		panic(err)
	}

	fmt.Println("status:", response.Status())          // 200 OK
	fmt.Println("status code:", response.StatusCode()) // 200
	response.Raw()                                     // *http.Response
	fmt.Printf("body: %s\n", response.BodyRaw())       // <!doctype html><html....
}

```