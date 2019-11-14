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

if [[ `docker ps -q -f name=wuid-mongo` == '' ]]; then
	if [[ `docker ps -a -q -f name=wuid-mongo` == '' ]]; then
	    docker run -d --name wuid-mongo -p 27017:27017 mongo:3.6
		[[ $? -ne 0 ]] && exit 1
	else
	    docker start wuid-mongo
		[[ $? -ne 0 ]] && exit 1
	fi
fi

echo "Done."
