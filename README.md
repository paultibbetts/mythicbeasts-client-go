# Mythic Beasts Client

[![Go Reference](https://pkg.go.dev/badge/github.com/paultibbetts/mythicbeasts-client-go.svg)](https://pkg.go.dev/github.com/paultibbetts/mythicbeasts-client-go)
[![Go Report Card](https://goreportcard.com/badge/github.com/paultibbetts/mythicbeasts-client-go)](https://goreportcard.com/report/github.com/paultibbetts/mythicbeasts-client-go)
[![Test Status](https://github.com/paultibbetts/mythicbeasts-client-go/actions/workflows/tests.yaml/badge.svg?branch=main)](https://github.com/paultibbetts/mythicbeasts-client-go/actions/workflows/tests.yaml)

mythicbeasts-client-go is a typed Go client for the Mythic Beasts [Raspberry Pi](https://www.mythic-beasts.com/support/api/raspberry-pi) and [VPS](https://www.mythic-beasts.com/support/api/vps) provisioning APIs.

## Installation

```bash
go get github.com/paultibbetts/mythicbeasts-client-go@latest
```

## Usage

### Quick start

```go
package main

import (
	"fmt"

	"github.com/paultibbetts/mythicbeasts-client-go"
)

func main() {
	c := mythicbeasts.NewClient("YOUR_API_KEYID", "YOUR_API_SECRET")

	// Example: list VPS disk sizes
	sizes, err := c.GetVPSDiskSizes()
	if err != nil {
		panic(err)
	}

    fmt.Println("Available HDD sizes:", sizes.HDD)
    fmt.Println("Available SSD sizes:", sizes.SSD)
}
```

### Import

```go
import "github.com/paultibbetts/mythicbeasts-client-go"
```

### Authentication

Create a new [API key](https://www.mythic-beasts.com/customer/api-users) and construct a new client using your API Key ID and secret:

```go
c := mythicbeasts.NewClient("YOUR_API_KEYID", "YOUR_API_SECRET")
```

You can manage your API tokens on [the Mythic Beasts site](https://www.mythic-beasts.com/customer/api-users).

### Idempotent deletion

The deletion of VPS or Pi servers counts a 404 as a success.

## Versioning

This project is pre-1.0 and minor releases may include breaking changes.

[Semantic versioning](https://semver.org/) is used for tags and a v1.0.0 will signal a stable API.

## Contributing

Contributions are welcome. This project uses [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

## License

MIT 2025 Paul Tibbetts.
