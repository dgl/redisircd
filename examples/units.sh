#!/usr/bin/env bash
# A units(1) based calculator/converter.
# Run ./units.sh units
# And /mode #channel +RP units

# <dg> = 1 AUD as GBP
# <units> 0.55028042
# <dg> = 1 microfortnight as microlunarmonth
# <units> 0.4740847

set -euo pipefail
shopt -s extglob

pubsub=${1:-units}

units_wrap() {
  # Ignore failures, output useful logging
  (set -x; units "$@") || :
}

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

  # Look for "= 1+2"
  if [[ ${text:0:1} = "=" ]]; then
    param="${text:1}"
    # "= 500 MiB as GiB"? i.e. conversion style
    first="${param/ @(as|to) */}"
    second=""
    if [[ $first != $param ]]; then
      second="${param/* @(as|to) /}"
    fi
    if [[ -z $second ]]; then
      # Calculation
      units_wrap --compact -1 "$first" | xargs -0 redis-cli publish $pubsub
    else
      # Conversion
      units_wrap --compact -1 "$first" "$second" | xargs -0 redis-cli publish $pubsub
    fi
  fi
done
