#!/usr/bin/env bash

[[ "$TRACE" ]] && set -x
pushd `dirname "$0"` > /dev/null
trap __EXIT EXIT

colorful=false
tput setaf 7 > /dev/null 2>&1
if [[ $? -eq 0 ]]; then
    colorful=true
fi

function __EXIT() {
    popd > /dev/null
}

function printError() {
    $colorful && tput setaf 1
    >&2 echo "Error: $@"
    $colorful && tput setaf 7
}

function printImportantMessage() {
    $colorful && tput setaf 3
    >&2 echo "$@"
    $colorful && tput setaf 7
}

function printUsage() {
    $colorful && tput setaf 3
    >&2 echo "$@"
    $colorful && tput setaf 7
}

root='github.com/edwingeng/wuid'
dirs='callback mongo mysql redis'

for d in $dirs; do
    go vet "$root/$d/wuid"
done
go vet "$root/internal"
go vet "$root/bench"

errcheck "$root"/... | ag -v '[ \t]*defer'

golint $dirs | ag -v 'receiver name should be a reflection of its identity'
