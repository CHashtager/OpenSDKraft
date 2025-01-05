# OpenSDKraft

OpenSDKraft is a tool for generating Go SDKs from OpenAPI v3 specifications. It takes an OpenAPI specification and generates a complete, ready-to-use Go SDK.

## Installation

```bash
# Clone the repository
git clone https://github.com/chashtager/opensdkraft.git
cd opensdkraft

# Install dependencies and build
make dev-deps
make build

# Optional: Install globally
make install
```

## Usage

```bash
# Basic usage
sdkraft [flags] <openapi-file>

# Example with config file
sdkraft -c config.yaml swagger.yaml

# Available flags:
#   -c, --config string     Config file path (default "./config.yaml")
#   -o, --output string     Output directory (default "./generated")
#   -p, --package string    Package name for generated code
#   -v, --verbose          Enable verbose logging
#       --with-tests       Generate tests (default true)
```

## Generated SDK Structure

When you run OpenSDKraft, it generates an SDK with the following structure:

```
generated-sdk/
├── client.go          # Main SDK client
├── models/           # Generated model types
│   ├── model1.go
│   ├── model2.go
│   └── ...
├── operations/       # Generated API operations
│   ├── operation1.go
│   ├── operation2.go
│   └── ...
└── tests/           # Generated tests (if enabled)
    ├── models/
    └── operations/
```

## Tool Structure

The OpenSDKraft tool itself is structured as follows:

```
.
├── cmd/
│   └── main.go           # CLI entry point
├── internal/
│   ├── config/          # Configuration handling
│   ├── generator/       # Core SDK generation
│   │   ├── models.go    # Model generation
│   │   ├── operations.go # Operation generation
│   │   └── templates.go # Template handling
│   ├── parser/         # OpenAPI spec parsing
│   └── utils/          # Common utilities
├── templates/          # Go templates for generation
├── config.yaml         # Default configuration
└── Makefile           # Build and development tasks
```

## Configuration

Create a `config.yaml` file to customize the SDK generation:

```yaml
sdkName: YourSDK
outputDir: ./generated
packageName: yoursdk

codeStyle:
  usePointers: true
  indentStyle: space
  maxLineLength: 120
  generateComments: true

generator:
  includeExamples: true
  includeValidation: true
  generateInterfaces: true
  clientOptions:
    timeout: 30
    retryEnabled: true
    maxRetries: 3

# See config.yaml example for full configuration options
```

## Examples

1. Generate a Petstore SDK:
```bash
# Create a new directory for your SDK
mkdir petstore-sdk
cd petstore-sdk

# Generate the SDK
sdkraft -o . -p petstore ../petstore.yaml

# Use the generated SDK in your Go code
import "github.com/yourusername/petstore-sdk"

client := petstore.NewClient("https://api.petstore.com")
pets, err := client.ListPets(context.Background(), &petstore.ListPetsParams{
    Limit: 10,
})
```

2. Generate SDK with custom configuration:
```bash
sdkraft -c custom-config.yaml -o ./my-sdk openapi.yaml
```

## Development

1. Install development dependencies:
```bash
make dev-deps
```

2. Run tests:
```bash
make test
```

3. Run linter:
```bash
make lint
```

4. Build the binary:
```bash
make build
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.