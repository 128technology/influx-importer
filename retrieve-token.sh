#!/bin/bash

read -p '128T HTTP Url: ' URL
read -p 'Username: ' USER
read -s -p 'Password (hidden): ' PASS

echo
echo Retrieving token...
echo

curl -X POST -H 'Content-Type: application/json' -d "{\"username\": \"$USER\", \"password\": \"$PASS\"}" -k $URL/api/v1/login
echo