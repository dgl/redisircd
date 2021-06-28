#!/usr/bin/env bash
# Example of a very simple "bot" that watches hacker news and then publishes it
# to a channel ("hn" redis pubsub).

# ./hn.sh => top stories
# ./hn.sh new => new stories

set -euo pipefail

type=${1:-top}
pub=${2:-hn}

# https://github.com/HackerNews/API
hn_firebase() {
  curl -s "https://hacker-news.firebaseio.com/v0/${1}.json"
}

stories() {
  hn_firebase "${1}stories" | jq -r '.[]'
}

seen() {
  key="hn${type}seen"
  [[ $(redis-cli hget $key "$1") = 1 ]] && return 0
  redis-cli hset $key "$1" 1 >/dev/null
  return 1
}

echo "Watching for $type stories and publishing to $pub"

while :; do
  for id in $(stories $type); do
    if ! seen $id; then
      line="$(hn_firebase "item/$id" | \
        jq -r '.title + " \u000312" +
          (if .url then .url else "https://news.ycombinator.com/item?id=" + (.id | tostring) end) +
          "\u0003 (" + (.score | tostring) + ")"')"
      (
        set -x
        redis-cli publish "$pub" "$line"
      ) >/dev/null
    fi
  done
  sleep 60
done
