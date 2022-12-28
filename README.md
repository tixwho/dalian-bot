# dalian-bot

Discord bot w/ various housekeeping functions, written in Golang.

In active development. Powered by [disrord.go](https://github.com/bwmarrin/discordgo) framework.

**🚧Warning🚧**: Reconstruction in progress!!! The new entry point is cmd/next.go. The documentation
below is *outdated* and will be rewritten in the future.

## Functions 

#### Utilities
* website-archiving (/save-site): Archive given website with tags and notes. 
  * Scroll archived sites in an interactive way (/list-site)
  * A snapshot is generated and stored into onedrive (in dev)
  * Automatically store *every* website in the given channel (in dev)

#### For fun
* **WHAT** (what): repeat the last message with bold font


## Directory struct

### Code directory

1. pkg
   1. clients: lower-level clients for intereaction
   2. commands: everything about discord commands.
      1. model.go: where you'll find all interfaces & struct definitions.
      2. setup.go: in charge of registering handlers to DiscordGo framework.
   3. data: wrapper for mongoDB interactions
   4. discord: wrapper for discord interactions.
   5. lifecycle.go: very self-explanatory.
2. main.go: entry function.

### Non-code directories
parallel to the executable file.
1. config: storing configurations for the bot.
   1. credentials.yaml: see credentials_format.yaml
2. static: storing permanent static files other than config, usually for testing & testing command
purposes.
