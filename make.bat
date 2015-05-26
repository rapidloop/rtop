@echo off
set GOPATH=%CD%
go build -o rtop.exe src\main.go src\consolehelper_windows.go src\format.go src\sshhelper.go src\stats.go
