# EchoLox

LoxBerry Plugin — emulates a Philips Hue bridge so Amazon Alexa controls a Loxone Miniserver via Virtual Inputs.

```
Alexa → Hue API (Port 8083) → EchoLox → Loxone Miniserver
```

## Build

```bash
make all          # cross-compile for arm64, armv7, amd64
make local        # build for current platform
```

## Install

Drop the plugin ZIP into LoxBerry Plugin Manager.

## Development

```bash
go run ./cmd/EchoLox --config config/EchoLox.cfg
```
