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

## Usage

Run redisircd:

./redisircd --listen :6667 --redis localhost:6379

Connect an IRC client to it.

Then:

```
/join #test
/mode #test +R test
```

Then run: redis-cli publish test foo

You should see:

```
<test> foo
```
