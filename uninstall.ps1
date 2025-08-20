# matlas-cli Uninstallation Script for Windows

param(
    [Parameter(HelpMessage="Keep configuration directory")]
    [switch]$KeepConfig,
    
    [Parameter(HelpMessage="Remove all installations without confirmation")]
    [switch]$Force,
    
    [Parameter(HelpMessage="Show help information")]
    [switch]$Help
)

# Configuration
$BINARY_NAME = "matlas"

# Helper functions
function Write-Success {
    param([string]$Message)
    Write-Host "✓ $Message" -ForegroundColor Green
}

function Write-Warning {
    param([string]$Message)
    Write-Host "⚠ $Message" -ForegroundColor Yellow
}

function Write-Error {
    param([string]$Message)
    Write-Host "✗ $Message" -ForegroundColor Red
}

function Write-Info {
    param([string]$Message)
    Write-Host "ℹ $Message" -ForegroundColor Blue
}

function Show-Usage {
    Write-Host @"
matlas-cli Uninstallation Script for Windows

USAGE:
    .\uninstall.ps1 [OPTIONS]

OPTIONS:
    -KeepConfig            Keep configuration directory
    -Force                 Remove all installations without confirmation
    -Help                  Show this help

EXAMPLES:
    .\uninstall.ps1                    # Interactive uninstallation
    .\uninstall.ps1 -Force            # Remove all without confirmation
    .\uninstall.ps1 -KeepConfig       # Remove binary but keep config

"@
}

function Find-Installations {
    $installations = @()
    
    # Common installation directories
    $searchDirs = @(
        "C:\Program Files\matlas",
        "C:\Program Files (x86)\matlas",
        "$env:USERPROFILE\matlas",
        "$env:LOCALAPPDATA\matlas"
    )
    
    # Add custom directory if specified
    if ($env:MATLAS_INSTALL_DIR) {
        $searchDirs += $env:MATLAS_INSTALL_DIR
    }
    
    # Search PATH directories
    $pathDirs = $env:PATH -split ';'
    foreach ($dir in $pathDirs) {
        if ($dir -and (Test-Path $dir)) {
            $searchDirs += $dir
        }
    }
    
    # Remove duplicates and search for binary
    $uniqueDirs = $searchDirs | Sort-Object -Unique
    
    foreach ($dir in $uniqueDirs) {
        $binaryPath = Join-Path $dir "$BINARY_NAME.exe"
        if (Test-Path $binaryPath) {
            $installations += $binaryPath
        }
    }
    
    return $installations
}

function Remove-Binary {
    param([string]$BinaryPath)
    
    $dir = Split-Path $BinaryPath -Parent
    
    try {
        if (Test-Path $BinaryPath) {
            Write-Info "Removing $BinaryPath..."
            Remove-Item $BinaryPath -Force
            Write-Success "Removed $BinaryPath"
            
            # Try to remove directory if it's empty and looks like our install dir
            if ((Get-ChildItem $dir | Measure-Object).Count -eq 0 -and 
                ($dir -like "*matlas*" -or $dir -like "*Program Files\matlas*")) {
                try {
                    Remove-Item $dir -Force
                    Write-Success "Removed empty directory $dir"
                }
                catch {
                    Write-Info "Could not remove directory $dir"
                }
            }
            
            return $true
        }
        else {
            Write-Warning "$BinaryPath not found"
            return $false
        }
    }
    catch {
        Write-Error "Failed to remove $BinaryPath : $($_.Exception.Message)"
        return $false
    }
}

function Remove-FromPath {
    param([string]$InstallDir)
    
    try {
        # Check user PATH
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath -and $userPath -like "*$InstallDir*") {
            Write-Info "Removing $InstallDir from user PATH..."
            $newPath = ($userPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
            [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
            Write-Success "Removed $InstallDir from user PATH"
        }
        
        # Check system PATH (requires admin)
        try {
            $systemPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
            if ($systemPath -and $systemPath -like "*$InstallDir*") {
                Write-Info "Removing $InstallDir from system PATH..."
                $newPath = ($systemPath -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
                [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
                Write-Success "Removed $InstallDir from system PATH"
            }
        }
        catch {
            Write-Info "Could not modify system PATH (admin privileges required)"
        }
        
        # Update current session
        $env:PATH = ($env:PATH -split ';' | Where-Object { $_ -ne $InstallDir }) -join ';'
        
    }
    catch {
        Write-Error "Failed to update PATH: $($_.Exception.Message)"
    }
}

function Remove-Config {
    $configDir = "$env:USERPROFILE\.matlas"
    
    if (Test-Path $configDir) {
        Write-Info "Configuration directory found: $configDir"
        
        if ($Force) {
            $response = "Y"
        }
        else {
            $response = Read-Host "Remove configuration directory? [y/N]"
        }
        
        if ($response -match "^[Yy]") {
            try {
                Write-Info "Removing configuration directory..."
                Remove-Item $configDir -Recurse -Force
                Write-Success "Removed configuration directory"
            }
            catch {
                Write-Error "Failed to remove configuration directory: $($_.Exception.Message)"
            }
        }
        else {
            Write-Info "Keeping configuration directory"
        }
    }
    else {
        Write-Info "No configuration directory found"
    }
}

# Main uninstallation function
function Main {
    if ($Help) {
        Show-Usage
        exit 0
    }
    
    Write-Info "matlas-cli Uninstallation Script for Windows"
    Write-Info "============================================="
    
    # Find all installations
    $installations = Find-Installations
    
    if ($installations.Count -eq 0) {
        Write-Info "No $BINARY_NAME installations found"
        
        # Still offer to clean up config
        if (-not $KeepConfig) {
            Remove-Config
        }
        
        return
    }
    
    Write-Info "Found $($installations.Count) installation(s):"
    foreach ($installation in $installations) {
        Write-Host "  - $installation"
    }
    Write-Host ""
    
    # Confirmation
    if (-not $Force) {
        $response = Read-Host "Remove all installations? [y/N]"
        
        if ($response -notmatch "^[Yy]") {
            Write-Info "Uninstallation cancelled"
            exit 0
        }
    }
    
    # Remove all installations
    $removed = 0
    $failed = 0
    
    foreach ($installation in $installations) {
        if (Remove-Binary $installation) {
            $removed++
            
            # Remove from PATH if it was the last binary in that directory
            $installDir = Split-Path $installation -Parent
            $remainingBinary = Join-Path $installDir "$BINARY_NAME.exe"
            if (-not (Test-Path $remainingBinary)) {
                Remove-FromPath $installDir
            }
        }
        else {
            $failed++
        }
    }
    
    # Summary
    if ($removed -gt 0) {
        Write-Success "Removed $removed installation(s)"
    }
    
    if ($failed -gt 0) {
        Write-Warning "$failed installation(s) could not be removed"
    }
    
    # Remove configuration
    if (-not $KeepConfig) {
        Remove-Config
    }
    
    # Final message
    if ($removed -gt 0) {
        Write-Success "matlas-cli uninstallation completed!"
        
        # Verify removal
        try {
            $null = Get-Command $BINARY_NAME -ErrorAction Stop
            Write-Warning "$BINARY_NAME command is still available (may be in a different location)"
            Write-Info "Run: Get-Command $BINARY_NAME"
        }
        catch {
            Write-Success "$BINARY_NAME command is no longer available"
        }
        
        Write-Warning "Please restart PowerShell or open a new terminal for PATH changes to take effect"
    }
}

# Run main function
Main
