@echo off
pushd %~dp0

go install github.com/gonutz/ico/cmd/ico@v1.1.0
ico rsc\icon.png

go install github.com/gonutz/rsrc/cmd/rsrc@v1.0.0
rsrc -ico rsc\icon.ico -arch 386 -o rsrc_windows_386.syso
rsrc -ico rsc\icon.ico -arch amd64 -o rsrc_windows_amd64.syso

popd
