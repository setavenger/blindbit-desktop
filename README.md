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
- Configure network (testnet/mainnet)
- Other wallet settings

## Configuration

The application stores configuration in `~/.blindbit-desktop/blindbit.toml`. Default settings include:

- Network: testnet
- Oracle URL: https://oracle.testnet.blindbit.com
- Electrum URL: ssl://electrum.blockstream.info:60002
- HTTP Port: 8080

## Security Features

- **Seed Generation**: Uses BIP39 for secure seed phrase generation
- **BIP352 Integration**: Full support for Silent Payments
- **Local Storage**: Wallet data stored locally with proper permissions
- **Private Keys**: Never transmitted over network

## Technical Details

### Architecture

The application is built using:
- **Fyne v2**: Cross-platform GUI framework
- **BIP352**: Silent Payment implementation
- **BIP39**: Mnemonic seed generation
- **HD Wallet**: Hierarchical deterministic wallet structure

### Integration

This desktop application integrates with:
- **blindbit-scan**: For UTXO scanning and blockchain monitoring
- **blindbit-wallet-cli**: For wallet operations and transaction creation

### Data Storage

Wallet data is stored in JSON format at `~/.blindbit-desktop/wallet.json`:
- Wallet configuration
- UTXO list
- Last scan height
- Labels and addresses

## Development

### Project Structure

```
blindbit-desktop/
├── cmd/
│   └── blindbit-desktop/   # Main application binary
│       └── main.go         # Application entry point
├── go.mod                  # Go module definition
├── internal/
│   ├── gui/               # GUI components
│   │   └── main.go        # Main GUI logic
│   └── wallet/            # Wallet management
│       └── manager.go     # Wallet operations
├── build.sh               # Build script
└── README.md              # This file
```

### Building for Distribution

#### Windows
```bash
go build -ldflags -H=windowsgui -o blindbit-desktop.exe ./cmd/blindbit-desktop
```

#### macOS
```bash
go build -o blindbit-desktop ./cmd/blindbit-desktop
```

#### Linux
```bash
go build -o blindbit-desktop ./cmd/blindbit-desktop
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support and questions:
- Open an issue on GitHub
- Check the [blindbit-scan documentation](https://github.com/setavenger/blindbit-scan)
- Check the [blindbit-wallet-cli documentation](https://github.com/setavenger/blindbit-wallet-cli)

## Acknowledgments

- [Fyne](https://fyne.io/) for the excellent GUI framework
- [BIP352](https://github.com/bitcoin/bips/blob/master/bip-0352.mediawiki) for Silent Payments specification
- The Bitcoin community for ongoing development and support
