@echo off

REM This script assumes you have ZopfliPNG installed
REM (https://github.com/google/zopfli).
REM It uses zopflipng to optimize (shrink) all PNG files in rsc.

pushd %~dp0\rsc
for %%x in (*.png) do zopflipng -y %%x %%x
popd
