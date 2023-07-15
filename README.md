# No Bullshit TT Bot

A bot that download tiktoks (both videos and image sequences).

I got tired of the amount of spam I received from the available sources.

Using the neat [Cobalt API](https://github.com/wukko/cobalt/blob/current/docs/API.md)

Rewritten in golang for that extra perf™

**Requires `ffmpeg` and `ffprobe`  in your PATH**

## Usage

build the main file

```
go build cmd/no-bs-go/main.go
```

Then run it with the bot token

```
BOT_TOKEN=mytoken ./main
```
