Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process
Set-Location "d:\OWN\SignalBlocks"
Write-Host "Downloading Go modules..."
go mod download
Write-Host "Tidying go.mod..."
go mod tidy
Write-Host "Dependencies installed successfully!" -ForegroundColor Green
Read-Host "Press Enter to exit"
