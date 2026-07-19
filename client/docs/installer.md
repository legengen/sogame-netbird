# Windows Package

`scripts/package-windows.ps1` creates an unsigned demo Windows x64 staging directory and ZIP. It verifies the official NetBird v0.74.7 MSI before copying it unchanged into the package. The package contains the Wails GUI, the narrowly scoped elevated helper, release metadata, the upstream NetBird license, and explicit install/uninstall scripts. Windows may show a SmartScreen warning for the unsigned Sogame executables.

The GUI uninstall path retains the official NetBird service by default. Passing `-RemoveNetBird` to `installer/uninstall.ps1` is the only removal path.
