# rebble-image-resizer
Very simple image resizer.

## URL Format

`/(exact/)?(<W>x<H>)?/key`

## Installation

```
go get github.com/pebble-dev/rebble-image-resizer
```

## Usage

```
Usage of rebble-image-resizer:
  --base-url string
        The base URL to which keys are appended
  --listen string
        The address to listen for connections (default "0.0.0.0:8080")
  --max-size string
        The max size of an image (default "1000x1000")
```
