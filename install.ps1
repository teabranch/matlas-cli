# matlas-cli Installation Script for Windows
# Supports Windows with automatic architecture detection

param(
    [Parameter(HelpMessage="Specific version to install (default: latest)")]
    [string]$Version = "",
    
    [Parameter(HelpMessage="Installation directory (default: C:\Program Files\matlas)")]
    [string]$InstallDir = "",
    
    [Parameter(HelpMessage="Skip automatic PATH setup")]
    [switch]$NoPathSetup,
    
    [Parameter(HelpMessage="Show help information")]
    [switch]$Help
)

# Configuration
$REPO_OWNER = "teabranch"
$REPO_NAME = "matlas-cli"
$BINARY_NAME = "matlas"
$DEFAULT_INSTALL_DIR = "C:\Program Files\matlas"

# Colors for output (if supported)
$esc = [char]27
$Colors = @{
    Red = "$esc[31m"
    Green = "$esc[32m"
    Yellow = "$esc[33m"
    Blue = "$esc[34m"
    Reset = "$esc[0m"
}

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
matlas-cli Installation Script for Windows

USAGE:
    .\install.ps1 [OPTIONS]

OPTIONS:
    -Version <string>        Install specific version (default: latest)
    -InstallDir <string>     Install directory (default: C:\Program Files\matlas)
    -NoPathSetup            Skip automatic PATH setup
    -Help                   Show this help

ENVIRONMENT VARIABLES:
    MATLAS_INSTALL_DIR      Custom installation directory

EXAMPLES:
    .\install.ps1                           # Install latest version
    .\install.ps1 -Version v1.2.3          # Install specific version
    .\install.ps1 -InstallDir C:\Tools     # Install to custom directory
    
    # Install to user directory (no admin required)
    .\install.ps1 -InstallDir "$env:USERPROFILE\matlas"

NOTE: 
    - Installation to system directories requires Administrator privileges
    - User installations don't require Administrator privileges

"@
}

function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Get-Architecture {
    $arch = [System.Environment]::GetEnvironmentVariable("PROCESSOR_ARCHITECTURE")
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { 
            Write-Error "Unsupported architecture: $arch"
            exit 1
        }
    }
}

function Get-LatestVersion {
    Write-Info "Fetching latest release information..."
    
    try {
        $apiUrl = "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
        $release = Invoke-RestMethod -Uri $apiUrl -Method Get
        return $release.tag_name
    }
    catch {
        Write-Error "Failed to fetch latest version: $($_.Exception.Message)"
        return $null
    }
}

function Test-InstallPermissions {
    param([string]$Path)
    
    try {
        # Try to create the directory if it doesn't exist
        if (-not (Test-Path $Path)) {
            New-Item -Path $Path -ItemType Directory -Force | Out-Null
        }
        
        # Test write permissions by trying to create a temporary file
        $testFile = Join-Path $Path "test_write_permissions.tmp"
        "test" | Out-File -FilePath $testFile -Force
        Remove-Item $testFile -Force
        return $true
    }
    catch {
        return $false
    }
}

function Download-Binary {
    param(
        [string]$Version,
        [string]$InstallPath
    )
    
    # Remove 'v' prefix if present
    $cleanVersion = $Version -replace '^v', ''
    
    $platform = "windows-$(Get-Architecture)"
    $archiveName = "$BINARY_NAME-$platform.zip"
    $downloadUrl = "https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/v$cleanVersion/$archiveName"
    
    Write-Info "Downloading $archiveName..."
    
    # Create temporary directory
    $tempDir = Join-Path $env:TEMP "matlas-install-$(Get-Random)"
    New-Item -Path $tempDir -ItemType Directory -Force | Out-Null
    
    try {
        $tempArchive = Join-Path $tempDir $archiveName
        
        # Download the archive
        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $tempArchive -UseBasicParsing
        }
        catch {
            Write-Error "Failed to download from $downloadUrl"
            Write-Error $_.Exception.Message
            return $false
        }
        
        Write-Info "Extracting binary..."
        
        # Extract the archive
        Expand-Archive -Path $tempArchive -DestinationPath $tempDir -Force
        
        # Find the binary
        $binaryPath = $null
        $possiblePaths = @(
            (Join-Path $tempDir "$BINARY_NAME.exe"),
            (Join-Path $tempDir "$BINARY_NAME-$platform.exe")
        )
        
        foreach ($path in $possiblePaths) {
            if (Test-Path $path) {
                $binaryPath = $path
                break
            }
        }
        
        if (-not $binaryPath) {
            # Try to find any .exe file
            $binaryPath = Get-ChildItem -Path $tempDir -Filter "*.exe" | Select-Object -First 1 -ExpandProperty FullName
        }
        
        if (-not $binaryPath) {
            Write-Error "Could not find binary in downloaded archive"
            return $false
        }
        
        # Create install directory
        if (-not (Test-Path $InstallPath)) {
            New-Item -Path $InstallPath -ItemType Directory -Force | Out-Null
        }
        
        # Copy binary to install location
        $finalBinary = Join-Path $InstallPath "$BINARY_NAME.exe"
        Write-Info "Installing to $finalBinary..."
        Copy-Item -Path $binaryPath -Destination $finalBinary -Force
        
        return $true
    }
    catch {
        Write-Error "Installation failed: $($_.Exception.Message)"
        return $false
    }
    finally {
        # Clean up temporary files
        if (Test-Path $tempDir) {
            Remove-Item $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

function Test-InPath {
    param([string]$Directory)
    
    $pathDirs = $env:PATH -split ';'
    return $pathDirs -contains $Directory
}

function Add-ToPath {
    param([string]$Directory)
    
    if (Test-InPath $Directory) {
        Write-Info "PATH already contains $Directory"
        return
    }
    
    Write-Info "Adding $Directory to PATH..."
    
    try {
        # Add to user PATH (doesn't require admin)
        $userPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($userPath) {
            $newPath = "$userPath;$Directory"
        } else {
            $newPath = $Directory
        }
        
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        
        # Update current session PATH
        $env:PATH += ";$Directory"
        
        Write-Success "Added $Directory to user PATH"
        Write-Warning "Please restart your PowerShell session or open a new terminal"
    }
    catch {
        Write-Error "Failed to add directory to PATH: $($_.Exception.Message)"
        Write-Info "Please manually add $Directory to your PATH"
    }
}

function Test-Installation {
    param([string]$InstallPath)
    
    $binaryPath = Join-Path $InstallPath "$BINARY_NAME.exe"
    
    if (Test-Path $binaryPath) {
        Write-Success "Installation successful!"
        
        # Try to get version
        try {
            $versionOutput = & $binaryPath version 2>$null
            if ($versionOutput) {
                Write-Info "Installed version: $($versionOutput[0])"
            }
        }
        catch {
            # Version command might not work, that's ok
        }
        
        Write-Info "Binary location: $binaryPath"
        
        if (Test-InPath $InstallPath) {
            Write-Success "Installation directory is in PATH"
            Write-Info "You can now run: $BINARY_NAME --help"
        } else {
            Write-Warning "Installation directory is not in PATH"
            Write-Info "Add it manually or restart PowerShell after PATH update"
        }
        
        return $true
    } else {
        Write-Error "Installation failed - binary not found at $binaryPath"
        return $false
    }
}

# Main installation function
function Main {
    if ($Help) {
        Show-Usage
        exit 0
    }
    
    Write-Info "matlas-cli Installation Script for Windows"
    Write-Info "=========================================="
    
    # Set install directory
    if (-not $InstallDir) {
        if ($env:MATLAS_INSTALL_DIR) {
            $InstallDir = $env:MATLAS_INSTALL_DIR
        } else {
            $InstallDir = $DEFAULT_INSTALL_DIR
        }
    }
    
    Write-Info "Installation directory: $InstallDir"
    
    # Check if we need admin rights
    $needsAdmin = $InstallDir.StartsWith("C:\Program Files") -or $InstallDir.StartsWith("C:\Windows")
    if ($needsAdmin -and -not (Test-Administrator)) {
        Write-Warning "Installation to system directory requires Administrator privileges"
        Write-Info "Options:"
        Write-Info "1. Run PowerShell as Administrator"
        Write-Info "2. Install to user directory: .\install.ps1 -InstallDir `"$env:USERPROFILE\matlas`""
        exit 1
    }
    
    # Test install permissions
    if (-not (Test-InstallPermissions $InstallDir)) {
        Write-Error "Cannot write to installation directory: $InstallDir"
        Write-Info "Try running as Administrator or choose a different directory"
        exit 1
    }
    
    # Get version to install
    if (-not $Version) {
        $Version = Get-LatestVersion
        if (-not $Version) {
            Write-Error "Failed to fetch latest version"
            exit 1
        }
        Write-Info "Latest version: $Version"
    } else {
        Write-Info "Installing version: $Version"
    }
    
    # Download and install
    if (-not (Download-Binary $Version $InstallDir)) {
        Write-Error "Installation failed"
        exit 1
    }
    
    # Setup PATH if requested
    if (-not $NoPathSetup -and -not (Test-InPath $InstallDir)) {
        Add-ToPath $InstallDir
    }
    
    # Verify installation
    if (Test-Installation $InstallDir) {
        Write-Success "matlas-cli installed successfully!"
        
        # Show next steps
        Write-Host ""
        Write-Info "Next steps:"
        Write-Info "1. Restart PowerShell or open a new terminal"
        Write-Info "2. Run: $BINARY_NAME --help"
        Write-Info "3. Configure authentication: $BINARY_NAME config --help"
    } else {
        exit 1
    }
}

# Run main function
Main
