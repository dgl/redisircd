# Redis IRCd

A redis backed IRC server.

## Whaaaaat?

This is a very simple IRC server, that is backed by Redis. This is mostly a toy
to provide something similar to [irccat](https://github.com/irccloud/irccat),
but running as a server and using Redis, because why not?

A particular use is sending the kind of thing you'd send to Slack via webhooks,
except there are no rate limits, so ideal for lots of data, or just where Slack
doesn't fit well.

This does not scale like a real IRC server would, in particular even if you
were to use a clustered Redis behind this, you'll find out that doesn't
[scale for pubsub](https://github.com/redis/redis/issues/2672).

## Building

For now just build directly from Git, e.g. using Go tooling like so:

```
go install github.com/dgl/redisircd/cmd/redisircd@latest
```

This will give you a `$(go env GOPATH)/bin/redisircd`

(Needs go 1.16, for earlier use `go get` rather than install.)

## Usage

Watch this!

[![asciicast](https://asciinema.org/a/422798.svg)](https://asciinema.org/a/422798)

or, run redisircd:

```
./redisircd --listen localhost:6667 --redis localhost:6379
```

Connect an IRC client to it.

Then:

```
/join #test
/mode #test +R test
```

Then run: `redis-cli publish test foo`

You should see:

```
<test> foo
```

JSON can be turned on:

```
/mode #test +JTN $.text $.nick
redis-cli publish test '{"text":"hi","nick":"yo"}'
```

Then you should see:

```
<yo> hi
```

## Modes

The custom modes this supports start with capital letters.

* `+R channel` Enable redis pubsub, listening on the given channel
* `+J` Redis pubsub payload is formatted as JSON
* `+N` Use JSONPath expression to extract nickname from JSON payload
* `+T` Use JSONPath expression to extract text from JSON payload
* `+P` Enable publishing things said on the channel. Will be sent to the
  channel configured with `+R` followed by `:out` to avoid loops (e.g.
  `channel:out`).

There's not yet any concept of ops or such. There may never be; this isn't
designed to be available on the public internet.

## Examples

These are designed to show how simple it is to write a bot or other tool for
this. More contributions welcome.

* [hn.sh](examples/hn.sh) is a script to watch for [Hacker
  News](https://news.ycombinator.com) updates and publish them.
* [units.sh](examples/units.sh) is a simple script that acts as a frontend to
  GNU Units and lets a user interact with it like a calculator.
* [bot.sh](examples/bot.sh) is a wrapper script that can run other carefully
  controlled commands.
* You may also be interested in
  [redis-irc-bot](https://github.com/dgl/redis-irc-bot) which is mostly
  compatible with the `:out` scheme used by `+P` but runs as a bot rather than
  a server.
