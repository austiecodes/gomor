# README

## Installation

install `go` first, 1.24 or higher is recommended.

```shell
go install github.com/austiecodes/goa/cmd/goa@latest
```

the goa binary will be installed to `~/go/bin` directory.
add it to your `PATH` environment variable, like this:

```shell
export PATH=$PATH:~/go/bin
```

then add following config to your `mcp-server` config file:

```json
{
  "mcpServers": {
    "goa": {
      "command": "goa",
      "args": ["mcp"]
    }
  }
}
```

## Usage
1. set up provider, 
use `goa set` command and select `provider` to set up
currently supporting:
* openai: chat-completion api
* google gemini
* anthropic
use your own apikey and setup your baseurl


2. set up `tool-model` and `embedding-model`
use `goa set` command and select `tool-model` and `embedding-model` to set up

3. config your own memory settings
use `goa set` command and select `memory` to set up

4. edit memory history
use `goa memory` command to edit memory history



now you are ok to goa!