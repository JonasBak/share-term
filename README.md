# [WIP] share-term

Get a "read-only" shareable link to your terminal to help with remote demonstration/debugging.

![screenshot](https://user-images.githubusercontent.com/16608915/85126041-f19b1880-b22c-11ea-9a8b-1fc3a5d36c1c.png)

## Deployment

You can easily run the server with docker
```
docker run --rm -p 8080:8080 jonasbak/share-term
docker run --rm -p 8080:8080 jonasbak/share-term <arg> ...
```

`share-term` can take the following arguments:
* `-server`
  * start `share-term` as a server, requires some template html files, which is why it's easier to run the server with docker
* `-addr <host>` where `<host>` is a hostname/ip without a protocol first (e.g. no `http://...`)
  * When running with `-server` this defines what ip/port you want to listen to
  * When running without `-server` this defines what server you want to connect to
  * This value can also be set via the `SHARE_TERM_ADDR` env variable, so the client doesn't have to keep typing it in.
* `-insecure` if you run the server without https, should only be used for testing
  * Both the server and client must use the flag to be able to connect
  * `share-term` doesn't handle any encryption itself, so it should be put behind a reverse proxy

The default arguments for the docker image are `-server -addr 0.0.0.0:8080`

## Development

Start a server with
```
go run . -addr localhost:8080 -insecure -server
```
and connect to it with
```
go run . -addr localhost:8080 -insecure
```

The "client" starts a shell (taken from the `SHELL` env variable) in a new PTY. It copies everything from your stdin to the PTY stdin, and copies everything from the PTY stdout to both your stdout and sends it to the server. There is currently no "persistence" for a session, meaning that if a user joins a session from the website, the user will only see what happends from that point and onward. This is why joining a session after the client already has opened something like vim or htop looks like a mess.
