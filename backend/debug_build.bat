@echo off
cd /d d:\stacksprint\backend
go vet ./internal/generator > build_out.txt 2>&1
echo Done >> build_out.txt
