# README

## Installation

install `go` first, 1.24 or higher is recommended.

```shell
go install github.com/austiecodes/gomor/cmd/gomor@latest
```

the gomor binary will be installed to `~/go/bin` directory.
add it to your `PATH` environment variable, like this:

```shell
export PATH=$PATH:~/go/bin
```

then add following config to your `mcp-server` config file:

```json
{
  "mcpServers": {
    "gomor": {
      "command": "gomor",
      "args": ["mcp"]
    }
  }
}
```

## Usage

1. set up provider,
use `gomor set` command and select `provider` to set up
currently supporting:

* openai: chat-completion api
* google gemini
* anthropic
use your own apikey and setup your baseurl

1. set up `tool-model` and `embedding-model`
use `gomor set` command and select `tool-model` and `embedding-model` to set up

2. config your own memory settings
use `gomor set` command and select `memory` to set up

3. edit memory history
use `gomor memory` command to edit memory history

now you are ok to gomor!
