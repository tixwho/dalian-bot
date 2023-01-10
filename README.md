# dalian-bot

A highly extensible multi-purpose bot w/ various housekeeping functions, written in Golang.

In active development. Powered by [disrord.go](https://github.com/bwmarrin/discordgo) framework.

Note: As a personal project, dalian-bot presented here may contain multiple plugins highly customized.
However, you are free to grab it as a bot template for your own project, thanks to the service-plugin system.

**🚧Warning🚧**: Reconstruction basically done, refining documentation & testing. The new entry point is cmd/next.go.

## Workflow

#### On startup
* [Entrypoint](cmd/next.go) collects configuration.
* *Services* are initialized and registered to the *bot*
* *Plugins* are then wired with *services* and ready.

#### When Running
* A *service* interacts with external and send *triggers* to the *Bot*.
* The *bot* dispatches *triggers* to registered *plugins*.
* Each *plugin* work independently and execute tasks, typically by calling *services* wrapped inside.

## What Dalian can do:

#### Utilities
* Archive websites (/archive): Archive given website with tags and notes. 
  * Scroll archived sites in an interactive way , modify and delete existing records.
  * A snapshot is generated and stored into onedrive (in dev)
  * Automatically store *every* website in the given channel (in dev)
* Help messages (/help, $help): Display help messages for commands, if supported by plugin.
* DDTV Webhook Notification (/ddtv): Parse webhook messages coming from [DDTV](https://github.com/CHKZL/DDTV),
a bilibili live-stream recorder, and display in a reasonable way.

#### For fun
* **What** : **WHAT**


## Configuration

A config-generator is working in progress, For now, you can manually save your config file (credentials.yaml)
at config/credentials.yaml following the format.


