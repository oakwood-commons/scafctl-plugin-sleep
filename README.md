# scafctl-plugin-sleep

Sleep/delay provider plugin for scafctl. Pauses workflow execution for a
configurable duration. Useful for rate limiting, waiting for external systems,
or pacing workflow execution.

## Installation

```bash
# Build from source
task build

# Or download from releases
gh release download --repo github.com/oakwood-commons/scafctl-plugin-sleep
```

## Usage

Register this plugin in your scafctl configuration, then reference
the **sleep** provider in your solutions:

```yaml
resolvers:
  wait-for-deploy:
    resolve:
      with:
        - provider: sleep
          inputs:
            duration: "5s"
```

## Development

```bash
# Run tests
task test

# Run linter
task lint

# Build
task build

# Full CI pipeline
task ci
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache-2.0 -- see [LICENSE](LICENSE) for details.
