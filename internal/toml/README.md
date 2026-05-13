# TOML Parser for Go

A minimalistic TOML parser that converts TOML to JSON or native Go data structures without external dependencies.

## Features

- **Zero Dependencies**: Pure Go implementation
- **Fast & Lightweight**: Direct parsing without intermediate representations
- **Two Output Formats**: JSON bytes or native Go maps
- **Clean API**: Simple functions for common use cases
- **Error Reporting**: Line-specific parsing errors
- **Supported Types**: Strings, numbers, booleans, dates, arrays, nested tables

## Installation

```bash
go get github.com/bulga138/toml
```

Or simply copy the toml.go file to your project.

## Usage

Convert TOML to JSON

```go
package main

import (
    "fmt"
    "log"
    "toml"
)

func main() {
    tomlData := `
title = "TOML Example"
number = 42
enabled = true

[database]
server = "192.168.1.1"
ports = [8001, 8002, 8003]
`

    // Convert to JSON
    jsonData, err := toml.Parse(tomlData)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("JSON Output: %s\n", jsonData)
    // Output: {"database":{"ports":[8001,8002,8003],"server":"192.168.1.1"},"enabled":true,"number":42,"title":"TOML Example"}
}
```

Parse TOML to Native Go Structures

```go
package main

import (
"fmt"
"log"
"toml"
)

func main() {
tomlData := `
name = "Example"
count = 123
items = ["apple", "banana", "cherry"]

[server]
host = "localhost"
port = 8080
`

    // Parse to native Go map
    data, err := toml.ParseNative(tomlData)
    if err != nil {
        log.Fatal(err)
    }

    // Access values
    fmt.Printf("Name: %s\n", data["name"])
    fmt.Printf("Count: %v\n", data["count"])

    server := data["server"].(map[string]interface{})
    fmt.Printf("Host: %s\n", server["host"])
    fmt.Printf("Port: %v\n", server["port"])

    items := data["items"].([]interface{})
    fmt.Printf("Items: %v\n", items)

}
```

## Supported TOML Features

Basic Types

# Strings

```toml
name = "Hello World"
```

# Integers

```toml
count = 42
```

# Floats

```toml
price = 3.14
```

# Booleans

```toml
enabled = true
debug = false
```

# Dates (RFC3339)

```toml
created = 2023-05-29T10:00:00Z
```

Arrays

```toml
numbers = [1, 2, 3]
strings = ["red", "yellow", "green"]
mixed = [1, "two", 3.0, true]
```

Tables

# Simple table

```
[database]
host = "localhost"
port = 5432
```

# Nested tables

```
[servers]
[servers.alpha]
ip = "10.0.0.1"

[servers.beta]
ip = "10.0.0.2"
```

Comments

# This is a comment

```
key = "value" # This is also a comment
```

API Reference

```
func Parse(tomlData string) ([]byte, error)
```

Converts TOML string to JSON bytes.

Returns:

```
    []byte: JSON representation of the TOML data
    error: Parsing error with line number if applicable
```

```
func ParseNative(tomlData string) (map[string]interface{}, error)
```

Converts TOML string to native Go data structures.

Returns:

```
    map[string]interface{}: Nested map representation of TOML data
    error: Parsing error with line number if applicable
```

Error Handling

All parsing functions return errors that implement the error interface:

```go
result, err := toml.Parse(tomlData)
if err != nil {
if parseErr, ok := err.(\*toml.ParseError); ok {
fmt.Printf("Parse error at line %d: %s\n", parseErr.Line, parseErr.Msg)
} else {
fmt.Printf("Error: %s\n", err)
}
}
```

Performance Tips

    Reuse Parser: For multiple parsing operations, consider creating a parser instance
    Memory: The parser creates new maps for each table - for large datasets, consider streaming approaches
    Validation: Parse early to catch syntax errors before processing

Limitations

This minimal parser does not support:

    Multi-line strings
    Inline tables ({ key = value })
    Hexadecimal numbers
    Advanced string escaping
    Array of tables ([[array]])

For full TOML v1.0 compliance, consider using a more comprehensive library.
License

MIT License - see LICENSE file for details.
Contributing

Contributions welcome! Please ensure:

    Code follows Go idioms
    Tests are included
    No external dependencies added
    Keep it minimal and performant
