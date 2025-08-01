# Flappy Bird Clone in Go

Flappy Gopher game that runs on Windows, Mac, Linux and in the browser.

It uses the [prototype/draw library](https://github.com/gonutz/prototype).


## Build and Run

For Windows, Linux and Mac, build and run with:

    go build .
    go run .

The browser port uses WASM. Install the `drawsm` tool like this:

    go install github.com/gonutz/prototype/cmd/drawsm@latest

Once installed, build for and run in the browser with:

    drawsm build
    drawsm run


## Modifying the game

The code is in the top level `.go` files.

Resources are in folders `raw` and `rsc`.

The `raw` folder contains raw image files. These are drawn at 3 times the
desired final scale. They are drawn without anti-aliasing for easier
modification (blurry edges can be hard to select exactly, so we use hard pixel
edges in the raw images).

The `rsc` folder contains the final images used in the game. They are scaled
down to 33% the original size from the `raw` folder images.

There are some `.pdn` files in `raw` which can only be opened with
[Paint.NET](https://www.getpaint.net/). This is not necessary, though, since
there are equivalent PNG files for them in `raw` as well.

For the background music there is `raw/music.ceol` which can be edited with
[Bosca Ceoil](https://yurisizov.itch.io/boscaceoil-blue).

Except for `rsc/music.wav`, sound files in `rsc` have no `raw` equivalent, since
they were created directly as uncompressed WAV files.

All assets in `rsc` will be embedded into the executable via Go's [embed
package](https://pkg.go.dev/embed).

### Windows Icon

To have the executable (`.exe`) file on Windows display an icon, the Go compiler
needs to be given a `.syso` file that contains the icon (`.ico`). `rsc/icon.png`
is the icon image. The Windows batch script `build_icon.bat` converts
`rsc/icon.png` into icon `rsc/icon.ico`. It then creates `rsrc_windows_386.syso`
and `rsrc_windows_amd64.syso` from that icon.
These files are automatically used by the Go compiler when building for Windows.

## Shrink PNGs

There is a Windows batch script `shrink_rsc.bat` that uses the
[ZopfliPNG](https://github.com/google/zopfli) tool - which is expected to be on
the path - to optimize (i.e. shrink) all PNG files in the `rsc` folder.
This will make the binaries - which embed the `rsc` folder - smaller.
