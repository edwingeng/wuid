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

docker run --name mysql-server -p 3306:3306 -e MYSQL_ROOT_PASSWORD=hello -d mysql
[[ $? -ne 0 ]] && exit 1

printImportantMessage "It may take quite a few seconds to get ready."
for ((i=0;i<1000;i++)); do
    docker run -it --rm mysql mysqladmin ping -h host.docker.internal --silent
    [[ $? -eq 0 ]] && echo "Ready." && break
    echo "Waiting $((i+1))..."
    sleep 1
done

sleep 1
docker run -v `pwd`/db.sql:/tmp/db.sql -it --rm mysql cat /tmp/db.sql | mysql -h host.docker.internal -u root -phello
[[ $? -ne 0 ]] && exit 1

echo "Job done."
