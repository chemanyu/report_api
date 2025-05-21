@echo off

set program_version=1.0.0
for /f "delims=" %%t in ('go version') do set compiler_version=%%t
set build_time=%DATE% %TIME%
set author=%username%
go build -ldflags "-X 'main.ProgramVersion=%program_version%' -X 'main.CompileVersion=%compiler_version%' -X 'main.BuildTime=%build_time%' -X 'main.Author=%author%'"
