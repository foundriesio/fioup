# Fioup

A command-line tool for performing Over-The-Air (OTA) updates of [Compose Apps](https://www.compose-spec.io/) published via [the FoundriesFactory™ Platform](https://docs.foundries.io/latest).

## Installation
[See the installation guide.](docs/install.md)

## Building

### Native Build

```bash
make
```

The binary is written to `./bin/fioup`.

### Cross-Compilation

`fioup` supports standard Go cross-compilation. For example:

```bash
GOOS=linux GOARCH=arm64 make
```

## Usage

[See the usage guide.](docs/README.md)

## Getting in Contact

* [Report an Issue on GitHub](../../issues)
* [Open a Discussion on GitHub](../../discussions)

## License

*fioup* is licensed under the [BSD 3-Clause Clear License](https://spdx.org/licenses/BSD-3-Clause-Clear.html).
See [LICENSE](LICENSE) for the full license text.
