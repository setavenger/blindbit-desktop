# BlindBit Desktop GUI Components

This directory contains the GUI components for the BlindBit Desktop application, organized into logical, manageable files.

## File Structure

### Core Components

- **`main_gui.go`** - Core GUI structure and management
  - `MainGUI` struct definition
  - `UTXODisplay` struct for UTXO data
  - `NewMainGUI()` constructor
  - Main content creation and management
  - Periodic updates and cleanup
  - Wallet info updates and UTXO refresh logic

### UI Screens

- **`welcome.go`** - Welcome screen for new users
  - `createWelcomeScreen()` - Initial onboarding interface
  - Create wallet and import wallet buttons

### Tab Components

- **`overview_tab.go`** - Main dashboard tab
  - `createOverviewTab()` - Overview with UTXO table
  - `startScanning()` / `stopScanning()` - Scanning controls
  - Network info display

- **`utxo_tab.go`** - Detailed UTXO management tab
  - `createUTXOOverviewTab()` - UTXO statistics and detailed view
  - UTXO statistics (total, unspent, spent, balance)
  - Clear UTXOs functionality
  - Enhanced UTXO table with headers

- **`send_tab.go`** - Transaction sending tab
  - `createSendTab()` - Send transaction form
  - `sendTransaction()` - Transaction execution logic

- **`receive_tab.go`** - Address display tab
  - `createReceiveTab()` - Receive address display
  - Copy address functionality

- **`settings_tab.go`** - Configuration tab
  - `createSettingsTab()` - Settings form
  - Network selection (testnet, mainnet, signet, regtest)
  - Oracle and Electrum server configuration
  - Wallet settings (dust limit, label count, birth height)
  - Tor configuration

### Dialog Components

- **`dialogs.go`** - Wallet creation and import dialogs
  - `showCreateWalletDialog()` - Wallet creation options
  - `showGenerateSeedDialog()` - New seed generation
  - `showEnterSeedDialog()` - Seed import form
  - `showImportWalletDialog()` - Import wallet flow

### Legacy

- **`window.go`** - Backward compatibility placeholder
  - Contains only package declaration and documentation
  - All functionality moved to separate component files

## Benefits of This Structure

1. **Maintainability** - Each file has a single responsibility
2. **Readability** - Smaller files are easier to understand
3. **Modularity** - Components can be modified independently
4. **Testability** - Individual components can be tested in isolation
5. **Collaboration** - Multiple developers can work on different components simultaneously

## Usage

The main entry point remains the same - import the `gui` package and use `NewMainGUI()` to create the main GUI instance. All the internal organization is transparent to the calling code.

```go
import "github.com/setavenger/blindbit-desktop/internal/gui"

// Create the main GUI
mainGUI := gui.NewMainGUI(window, walletManager)
content := mainGUI.GetContent()
```

## Adding New Components

When adding new UI components:

1. Create a new file with a descriptive name (e.g., `new_feature_tab.go`)
2. Follow the existing naming conventions
3. Add the component to the main tab creation in `main_gui.go`
4. Update this README with the new component description 