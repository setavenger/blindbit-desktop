# BlindBit Desktop

A modern desktop application that combines the functionality of blindbit-scan and blindbit-wallet-cli into a single, user-friendly GUI application for managing Bitcoin Silent Payment (BIP 352) wallets.

## Features

- **Wallet Management**: Create new wallets with generated seed phrases or import existing ones
- **Address Generation**: Generate Silent Payment addresses for receiving Bitcoin
- **UTXO Scanning**: Scan for your wallet's UTXOs
- **Transaction Sending**: Send Bitcoin to regular and Silent Payment addresses
- **Balance Tracking**: View your wallet balance and transaction history
- **Cross-Platform**: Works on macOS and Linux (Windows?)
- **Lightweight**: Built with Fyne GUI framework, no browser required

*Windows should technically work but I cannot test it. If it doesn't, you could help me debug issues :)

## Prerequisites

- Go 1.24.1 or later
- C compiler (for Fyne dependencies)
- System development tools

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/setavenger/blindbit-desktop.git
cd blindbit-desktop
```

2. Install dependencies:
```bash
go mod tidy
```

3. Build the application:
```bash
go build -o blindbit-desktop ./cmd/blindbit-desktop
```

4. Run the application:
```bash
./blindbit-desktop
```

To use a custom data directory:

```bash
./blindbit-desktop --datadir /path/to/datadir-2
```

### Using Go Install

```bash
go install github.com/setavenger/blindbit-desktop/cmd/blindbit-desktop@latest
```

## Packaging

To package the application for distribution, use the Fyne package tool.

**Native compilation** (building on the same platform):
- No `-os` flag needed

**Cross-compilation** (building on a different platform):
- Use the `-os` flag to specify the target platform

### Examples

#### Native Packaging

```bash
# macOS (on macOS)
fyne package -src ./cmd/blindbit-desktop

# Windows (on Windows)
fyne package -src ./cmd/blindbit-desktop

# Linux (on Linux)
fyne package -src ./cmd/blindbit-desktop
```

#### Cross-Compilation

```bash
# macOS (from Linux/Windows)
fyne package -src ./cmd/blindbit-desktop -os darwin

# Windows (from macOS/Linux)
fyne package -src ./cmd/blindbit-desktop -os windows

# Linux (from macOS/Windows)
fyne package -src ./cmd/blindbit-desktop -os linux
```

For a complete packaging script example, see `package-macos.sh`.

**Note**: Packaging requires the Fyne CLI tool. Install it with:

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
```

## Configuration

The application stores wallet data in `~/.blindbit-desktop/wallet.dat` by default, or in `<datadir>/wallet.dat` when `--datadir` is provided.

Default settings:
- **Network**: signet
- **Oracle Address** (gRPC):
  - Mainnet: `oracle.setor.dev`
  - Signet: `signet.oracle.setor.dev`
- **TLS**: Enabled by default

## Support

For support and questions:
- Open an issue on GitHub
