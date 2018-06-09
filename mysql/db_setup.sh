#!/usr/bin/env bash

docker run -d --name wuid-mysql -p 3306:3306 -e MYSQL_DATABASE=test -e MYSQL_ROOT_PASSWORD=password yobasystems/alpine-mariadb