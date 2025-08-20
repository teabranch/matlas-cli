# Feature: Cross-Platform Installation System

## Summary

Implemented a comprehensive installation system for matlas-cli that supports macOS, Linux, and Windows with automatic platform detection, PATH configuration, and shell integration. The system provides multiple installation methods ranging from one-liner quick installs to customizable installation scripts, along with proper uninstallation and upgrade mechanisms.

## Components Added

### Installation Scripts
- **`install.sh`**: Universal installation script for macOS and Linux with automatic platform detection, PATH setup, and shell integration (bash, zsh, fish)
- **`install.ps1`**: PowerShell installation script for Windows with user/system installation options and automatic PATH management
- **`quick-install.sh`**: One-liner installation script that downloads and runs the full installer

### Maintenance Scripts
- **`uninstall.sh`**: Unix-like uninstaller that finds all installations, removes binaries, cleans PATH configurations, and optionally removes config directory
- **`uninstall.ps1`**: Windows PowerShell uninstaller with similar functionality to the Unix version
- **`upgrade.sh`**: Upgrade script that detects current installation, fetches latest version, and performs in-place upgrades

### Build Integration
- **Enhanced Makefile**: Added `install`, `install-user`, and `uninstall` targets for development workflow integration

## Features

### Cross-Platform Support
- **macOS**: Intel (amd64) and Apple Silicon (arm64) support
- **Linux**: x86_64 (amd64) and ARM64 support  
- **Windows**: AMD64 and ARM64 support

### Installation Methods
1. **Quick Install**: One-liner commands for each platform
2. **Script-based**: Downloaded scripts with advanced options
3. **Manual**: Traditional download-extract-install process
4. **Build from Source**: Developer-friendly build process

### Automatic Configuration
- **PATH Detection**: Automatically detects and configures PATH for different shells
- **Shell Integration**: Supports bash, zsh, fish, and PowerShell
- **Permission Handling**: Graceful handling of admin/sudo requirements
- **User vs System**: Support for both user-level and system-level installations

### Maintenance
- **Version Detection**: Checks current and available versions
- **Upgrade Support**: In-place upgrades with version verification
- **Clean Uninstall**: Removes binaries, PATH entries, and configuration files
- **Multiple Installation Detection**: Finds and manages multiple installations

## Installation Options

| Method | Platform | Admin Required | Customizable |
|--------|----------|----------------|--------------|
| Quick Install | All | System dirs only | No |
| Installation Script | All | System dirs only | Yes |
| Manual Install | All | System dirs only | Yes |
| Make targets | macOS/Linux | Optional | Yes |
| Build from Source | All | No | Yes |

## Documentation

### README Updates
- **Comprehensive Installation Section**: Detailed instructions for all platforms and methods
- **Verification Steps**: How to confirm successful installation
- **Troubleshooting**: Common issues and solutions
- **Upgrade Instructions**: How to update existing installations
- **Uninstall Instructions**: Clean removal procedures

### Usage Examples
- Quick one-liner installations
- Custom directory installations  
- Version-specific installations
- User vs system installations
- Development workflow integration

## Testing Considerations

The installation system handles various scenarios:
- Different user permission levels
- Missing dependencies (curl/wget)
- Network connectivity issues
- Multiple existing installations
- Different shell environments
- Custom installation directories
- Version compatibility checks

## Future Enhancements

Potential improvements for future iterations:
- Package manager integration (homebrew, chocolatey, apt, yum)
- Automatic update notifications
- Installation verification checksums
- Rollback mechanism for failed upgrades
- Silent installation options for CI/CD

## Related Files

### Scripts
- `install.sh` - Unix-like installer
- `install.ps1` - Windows PowerShell installer  
- `uninstall.sh` - Unix-like uninstaller
- `uninstall.ps1` - Windows PowerShell uninstaller
- `upgrade.sh` - Upgrade utility
- `quick-install.sh` - One-liner installer

### Build System
- `Makefile` - Enhanced with installation targets
- `scripts/build/build.sh` - Cross-compilation support

### Documentation  
- `README.md` - Comprehensive installation documentation
