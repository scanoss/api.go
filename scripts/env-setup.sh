#!/bin/bash

##########################################
#
# This script will copy all the required files into the correct locations on the server
# Config goes into: /usr/local/etc/scanoss/api
# Logs go into: /var/log/scanoss/api
# Service definition goes into: /etc/systemd/system
# Binary & startup go into: /usr/local/bin
#
################################################################

echo "Setting up API system folders..."
mkdir -p /usr/local/etc/scanoss/api
mkdir -p /var/log/scanoss/api
echo "Changing ownership to scanoss..."
chown -R scanoss /var/log/scanoss

echo "Copying service startup config..."
cp scanoss-go-api.service /etc/systemd/system
cp scanoss-go-api.sh /usr/local/bin

CONF=app-config-prod.json
if [ -f $CONF ] ; then
  echo "Copying app config to /usr/local/etc/scanoss/api ..."
  cp $CONF /usr/local/etc/scanoss/api
else
  echo "Please put the config file into: /usr/local/etc/scanoss/api/$CONF"
fi
BINARY=scanoss-go-api
if [ -f $BINARY ] ; then
  echo "Copying app binary to /usr/local/bin ..."
  cp $BINARY /usr/local/bin
else
  echo "Please copy the API binary file into: /usr/local/bin/$BINARY"
fi
