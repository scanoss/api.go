#!/bin/bash

##########################################
#
# This script is designed to run by Systemd SCANOSS GO API service.
# It rotates scanoss log file and starts SCANOSS API.
# Install it in /usr/local/bin
#
################################################################
TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
ENVIRONMENT="prod"
LOGFILE=/var/log/scanoss/api/scanoss-api-${ENVIRONMENT}.log
BACKUP_FILE=$LOGFILE.$TIMESTAMP
CONF_FILE=/usr/local/etc/scanoss/api/app-config-${ENVIRONMENT}.json
# Rotate log
if [ -f $LOGFILE ] ; then
  echo "rotating logfile..."
  cp $LOGFILE $BACKUP_FILE
  gzip -f $BACKUP_FILE
fi
echo > $LOGFILE

# echo "removing old fingerprint & sbom temporary files..."
# rm -f /tmp/finger*.wfp /tmp/sbom*.json /tmp/failed-finger*.wfp

#start API
echo "starting SCANOSS GO API"

exec /usr/local/bin/scanoss-go-api --json-config $CONF_FILE > $LOGFILE 2>&1
