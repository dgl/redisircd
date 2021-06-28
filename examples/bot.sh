#!/usr/bin/env bash
# A bot wrapper. This makes anything that matches [a-z0-9]+ and is executable
# in the current directory into a "!" command.

# Run something like:
#   echo "$(which uptime)" > uptime
#   chmod +x uptime
#   ./bot.sh
# Then on IRC:
#   /mode #channel +RP bot
#   <dg> !uptime
#   <bot> 08:31:35  up 5 days  6:10,  2 users,  load average: 0.26, 0.57, 1.02
#
# Note this carefully wraps uptime, as "uptime(1)" itself takes a file argument
# and could be used to do unexpected things. Better to use custom wrapper
# scripts that further sanity check the arguments provided. Also, you know, not
# deal with user input in shell scripts, but that's no fun.

set -eu
shopt -s extglob

pubsub=${1:-bot}

# stdbuf from https://stackoverflow.com/a/66103101, because that's obvious,
# thanks redis-cli.
stdbuf -oL redis-cli subscribe "${pubsub}:out" | while read type; do
  read channel # 2nd line
  read message # 3rd line
  if [[ $type != message ]]; then
    continue
  fi
  nick="${message/ */}"
  text="${message/+([^ ]) /}"

  # Look for "!"
  if [[ ${text:0:1} = "!" ]]; then
    param="${text:1}"
    command="${param/ */}"
    command="${command,,?}"
    command="${command//[^a-z0-9]/}"
    params=""
    if [[ ${param/* /} != $param ]]; then
      params="${param/+([^ ]) /}"
    fi
    if [[ -n $command ]] && [[ -x $command ]]; then
      echo "$params" | nick="$nick" ./$command | xargs -0 -r -n1 -s500 redis-cli publish ${pubsub}
    fi
  fi
done
