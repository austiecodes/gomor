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

4. call memory operations directly from an agent or shell

```shell
# Save a memory
gomor memory --save "The user prefers concise answers" --tags "preference,style"

# Query memories in a LLM-friendly JSON format
gomor memory --query "How should I answer this user?" --json

# Delete an incorrect memory by id
gomor memory --delete "memory-id" --json
```

For shell or LLM usage, prefer `--json` so the caller can reliably parse ids and scores.
Memory retrieval is a weak signal for recency, not a correctness confirmation. Delete memories that are clearly wrong or obsolete.

now you are ok to gomor!
