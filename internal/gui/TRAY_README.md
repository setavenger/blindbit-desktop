# System Tray Functionality

This document describes the system tray functionality implemented for BlindBit Desktop, with a focus on macOS support.

## Overview

The system tray allows the BlindBit Desktop application to continue running in the background when the main window is closed, providing a persistent presence in the system tray (macOS dock).

## Features

### macOS Support
- **Dock Integration**: The app integrates with the macOS dock
- **Window Management**: Clicking the red close button hides the window instead of closing the app
- **Persistent Running**: The app continues running in the background for wallet scanning and updates

### System Tray Menu
- **Show/Hide**: Toggle the main window visibility
- **Quit**: Completely exit the application

## How It Works

### Window Close Behavior
When the user clicks the red close button (X) on macOS:
1. The window is hidden instead of closed
2. The app continues running in the background
3. The wallet scanner continues to operate
4. The app remains accessible through the dock

### Tray Menu Access
- **macOS**: Right-click the dock icon to access the context menu
- The menu provides options to show/hide the window or quit the app

### Background Operation
- Wallet scanning continues in the background
- UTXO updates and blockchain synchronization continue
- The app maintains its state and wallet connections

## Implementation Details

### Files Modified
- `tray.go`: System tray manager implementation using Fyne's native support
- `main_gui.go`: Integration with main GUI
- `main.go`: App reference passing for tray initialization

### Key Components
- `TrayManager`: Handles system tray functionality
- `SetCloseIntercept`: Prevents window closure, replaces with hiding
- `desktop.App`: Fyne's native system tray interface
- `SetSystemTrayMenu`: Sets up the system tray menu

### Native Fyne Support
This implementation uses Fyne's built-in system tray support (available since v2.2.0) instead of third-party packages. This provides:
- Better integration with the Fyne framework
- Consistent behavior across platforms
- Automatic cleanup and lifecycle management
- No external dependencies

## Usage

### For Users
1. **Normal Operation**: Use the app as usual
2. **Minimize to Tray**: Click the red close button to hide the window
3. **Restore Window**: Right-click dock icon → "Show/Hide" or double-click dock icon
4. **Quit App**: Right-click dock icon → "Quit"

### For Developers
The tray functionality is automatically initialized when creating a new MainGUI instance:

```go
mainGUI := gui.NewMainGUI(myApp, mainWindow, walletManager)
```

## Platform Considerations

### macOS
- Uses native dock integration
- Window close button behavior is overridden
- App remains in dock when window is hidden

### Future Platforms
- Windows: System tray icon in notification area
- Linux: System tray icon in panel
- Cross-platform: Consistent menu behavior

## Benefits

1. **Persistent Operation**: App continues running for background tasks
2. **User Control**: Users can choose to hide the window without losing functionality
3. **Resource Management**: App remains accessible without cluttering the desktop
4. **Wallet Security**: Scanning and updates continue even when window is hidden
5. **Native Integration**: Uses Fyne's built-in system tray support for better reliability

## Limitations

- Tray icon clicks are not directly supported (menu-based interaction only)
- Notifications are not yet implemented
- Platform-specific features may vary
- Dynamic icon updates require additional implementation

## Future Enhancements

- System notifications for wallet events
- Tray icon click handling (where supported)
- Custom tray icons using `desk.SetSystemTrayIcon`
- Enhanced menu options
- Background task status indicators
- Dynamic icon updates based on wallet state 