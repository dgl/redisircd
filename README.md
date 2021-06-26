# Redis IRCd

A redis backed IRC server.

## What?

This is a very simple IRC server, that is backed by Redis. This is mostly a toy
to provide something similar to [irccat](https://github.com/irccloud/irccat),
but running as a server and using Redis, because why not?

A particular use is sending the kind of thing you'd send to Slack via webhooks,
except there are no rate limits, so ideal for lots of data, or just where Slack
doesn't fit well.

This does not scale like a real IRC server would, in particular even if you use
a clustered Redis behind this, you'll find out that doesn't
[scale](https://github.com/redis/redis/issues/2672).

## Usage

Run redisircd:

redisircd --listen :6667 --redis localhost:6379

pflag

Todo options:

 --auth

 --pubsub-prefix
   If you make dedicated pubsub channels or you just don't want people watching every prefix
 --monitor
   Enable &monitor channel (see MONITOR output)
 --control
   Enable &control channel (send raw redis commands)

 --irc-state-key
   Key to use to store IRC state (optional), set to "irc:channels" and a hash
   will be created storing the state of each channel, meaning redisircd keeps
   state between restarts.

Connect an IRC client to it.

Then /join #test

Then run: redis-cli publish test foo

You should see:

<test> foo

/mode #test +TNL json .foo .text

JSON:

redis-cli publish test '{"foo":"hi","text":"body"}'

<hi> body

Modes:

+T type: JSON | PLAIN [default]
+N nick: If JSON: JSON path expression
+L line: If JSON: JSON path expression
+m kind of moderated in IRC sense, no users can send messages
+M users can send messages, to chat about whatever, but they aren't PUBLISHed

+nt are unused, because some clients try to set these on channel creation, but
we don't want that.
+o and so on isn't implemented, everyone gets ops for now.

/join &monitor

MONITOR command output

/join &control

Raw redis commands:

SET ...
