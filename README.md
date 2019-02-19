# godocjson

Produces JSON-formatted Go documentation.

## Installation

Make sure your Go environment is configured correctly, then run:

```go get github.com/rtfd/godocjson```

## Usage

```godocjson [-e <pattern>] <directory>```

The **godocjson** scans <directory> for Go packages and outputs JSON-formatted documentation to stdout

The options are as follows:

    -e   <pattern>   Exclude files that match specified pattern from processing.
                     Example usage:
                        godocjson -e _test.go ./go/sources/folder
