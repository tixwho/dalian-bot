# dalian-bot

Discord bot w/ various housekeeping functions, written in Golang.

## Code Directory struct

1. pkg
   1. clients: lower-level clients for intereaction
   2. commands: everything about discord commands.
      1. model.go: where you'll find all interfaces & struct definitions.
      2. setup.go: in charge of registering handlers to DiscordGo framework.
   3. data: wrapper for mongoDB interactions
   4. discord: wrapper for discord interactions.
   5. lifecycle.go: very self-explanatory.
2. main.go: entry function.

## Other directories
parallel to the executable file.
1. config: storing configurations for the bot.
   1. credentials.yaml: see credentials_format.yaml
2. static: storing permanent static files, usually for testing & testing command purposes.
