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

if [[ `docker ps -q -f name=wuid-mysql` == '' ]]; then
	if [[ `docker ps -a -q -f name=wuid-mysql` == '' ]]; then
		docker run -d --name wuid-mysql -p 3306:3306 -e MYSQL_DATABASE=test -e MYSQL_ROOT_PASSWORD=password mariadb
		[[ $? -ne 0 ]] && exit 1
	else
		docker start wuid-mysql
		[[ $? -ne 0 ]] && exit 1
	fi
fi

for ((i=0;i<1000;i++)); do
	echo "Trying to create the wuid table [$((i+1))]..."
	mysql -h127.0.0.1 -uroot -ppassword test < ../db.sql > /dev/null
	[[ $? -eq 0 ]] && echo "Done." && break
	sleep 1
done
