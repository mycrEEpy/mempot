# mempot

[![Go Reference](https://pkg.go.dev/badge/github.com/mycreepy/mempot.svg)](https://pkg.go.dev/github.com/mycreepy/mempot)
[![Go Report Card](https://goreportcard.com/badge/github.com/mycreepy/mempot?style=flat-square)](https://goreportcard.com/report/github.com/mycreepy/mempot)
[![Go Build & Test](https://github.com/mycrEEpy/mempot/actions/workflows/build.yml/badge.svg)](https://github.com/mycrEEpy/mempot/actions/workflows/build.yml)
[![Go Coverage](https://github.com/mycreepy/mempot/wiki/coverage.svg)](https://raw.githack.com/wiki/mycreepy/mempot/coverage.html)

`mempot` is a small and easy generic memory cache for Go.

## Usage

```go
package main

import (
	"context"
	"fmt"

	"github.com/mycreepy/mempot"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := mempot.NewCache[string, string](ctx, mempot.DefaultConfig)

	cache.Set("foo", "bar")

	item, ok := cache.Get("foo")
	if !ok {
		panic("item not found or expired")
	}

	fmt.Println(item.Data)
}
```
