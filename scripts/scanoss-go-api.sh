#!/bin/bash

##########################################
#
# This script is designed to run by Systemd SCANOSS GO API service.
# It rotates scanoss log file and starts SCANOSS API.
# Install it in /usr/local/bin
#
################################################################
DEFAULT_ENV="prod"
ENVIRONMENT="${1:-$DEFAULT_ENV}"
LOGFILE=/var/log/scanoss/api/scanoss-api-${ENVIRONMENT}.log
CONF_FILE=/usr/local/etc/scanoss/api/app-config-${ENVIRONMENT}.json
ENV_FILE=/usr/local/etc/scanoss/api/app-config-${ENVIRONMENT}.env
CMD_ARGS=(--json-config "$CONF_FILE")
# Rotate log
if [ -f "$LOGFILE" ] ; then
  echo "rotating logfile..."
  TIMESTAMP=$(date '+%Y%m%d-%H%M%S')
  BACKUP_FILE=$LOGFILE.$TIMESTAMP
  cp "$LOGFILE" "$BACKUP_FILE"
  gzip -f "$BACKUP_FILE"
fi
echo > "$LOGFILE"
# Add env file if it exists
if [ -e "$ENV_FILE" ] && [ ! -r "$ENV_FILE" ] ; then
  echo "env file is not readable: $ENV_FILE" >&2
  exit 1
elif [ -f "$ENV_FILE" ] ; then
  echo "adding env file"
  CMD_ARGS+=(--env-config "$ENV_FILE")
fi
# echo "removing old fingerprint & sbom temporary files..."
# rm -f /tmp/finger*.wfp /tmp/sbom*.json /tmp/failed-finger*.wfp

echo "starting SCANOSS GO API"
#start API
exec /usr/local/bin/scanoss-go-api "${CMD_ARGS[@]}" > "$LOGFILE" 2>&1
