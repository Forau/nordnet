# examples

This is a small example of how one could use the api with rpc, and thus make client that doesnt require nordnet's credentials.

## Running examples

Start daemon in one window:

  $ go run daemon.go -user <user> -pass <pass>

This is for demo only. Dont use command arguments for credentials in a real system, since anyone will see them with 'ps -fe'

Then you can run all the commands with a client.  Just run without arguments for help.

  $ go run client.go

Examples:

  $ go run client.go lists   # Gets the list_id's
  $ go run client.go list 16314763  # Gets all instruments on Large Cap

Public feed:
  $ go run client.go PublicStream
  ...
  sub price 101 11

The daemon will reconnect if the session has timed out, but if the feed dies, the client must reconnect in this version.

