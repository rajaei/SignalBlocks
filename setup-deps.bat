@echo off
cd /d "d:\OWN\SignalBlocks"
echo Downloading Go modules...
go mod download
echo Tidying go.mod...
go mod tidy
echo Dependencies installed successfully!
pause
