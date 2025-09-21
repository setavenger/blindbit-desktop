# BlindBit Desktop

A modern desktop application that combines the functionality of blindbit-scan and blindbit-wallet-cli into a single, user-friendly GUI application for managing Bitcoin Silent Payment (BIP 352) wallets.

## Features

- **Wallet Management**: Create new wallets with generated seed phrases or import existing ones
- **Address Generation**: Generate Silent Payment addresses for receiving Bitcoin
- **UTXO Scanning**: Scan the blockchain for your wallet's UTXOs
- **Transaction Sending**: Send Bitcoin to regular and Silent Payment addresses
- **Balance Tracking**: View your wallet balance and transaction history
- **Cross-Platform**: Works on Windows, macOS, and Linux
- **Lightweight**: Built with Fyne GUI framework, no browser required

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

## Usage

### First Time Setup

1. Launch the application
2. Click "Create New Wallet"
3. Choose "Generate New Seed" to create a new wallet
4. **IMPORTANT**: Write down the generated seed phrase and keep it safe
5. Click "Create Wallet" to proceed

### Importing Existing Wallet

1. Click "Create New Wallet"
2. Choose "Enter Existing Seed"
3. Enter your 24-word seed phrase
4. Click "Import" to load your wallet

### Main Interface

The application has four main tabs:

#### Overview Tab
- View your wallet address and balance
- See all UTXOs (Unspent Transaction Outputs)
- Start/stop blockchain scanning
- Refresh UTXO list

#### Send Tab
- Enter recipient address
- Specify amount in satoshis
- Set fee rate
- Send transactions

#### Receive Tab
- Display your wallet address
- Copy address to clipboard

#### Settings Tab
- Configure network (testnet/mainnet/signet)
- Other wallet settings

## Configuration

The application stores configuration in `~/.blindbit-desktop/blindbit.toml` by default, or in `<datadir>/blindbit.toml` when `--datadir` is provided. Default settings include:

- Network: testnet
- Oracle URL: https://silentpayments.dev/blindbit/mainnet
- HTTP Port: 8080

## Support

For support and questions:
- Open an issue on GitHub
