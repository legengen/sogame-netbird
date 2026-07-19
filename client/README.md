# Sogame Windows client

Sogame is a Windows 10/11 x64 Wails application that orchestrates the official
NetBird v0.74.7 daemon. It does not implement any networking protocol.

Development commands:

```powershell
cd client
go test ./...
cd frontend
npm install
npm run build
cd ..
.\scripts\build-windows.ps1
```

The release script accepts only Windows x64 and verifies the official NetBird
MSI metadata before running Wails. The MSI itself is downloaded into a local
artifact cache and is never committed.
