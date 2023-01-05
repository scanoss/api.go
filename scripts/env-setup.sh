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

export RUNTIME_USER=scanoss
if ! getent passwd $RUNTIME_USER > /dev/null ; then
  echo "Runtime user does not exist: $RUNTIME_USER"
  echo "Please create using: useradd --system $RUNTIME_USER"
  exit 1
fi
if [ "$EUID" -ne 0 ] ; then
  echo "Please run as root"
  exit 1
fi
read -p "Install SCANOSS Go API (y/n) [n]? " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]] ; then
  echo "Starting installation..."
else
  echo "Stopping."
  exit 1
fi

echo "Setting up API system folders..."
if ! mkdir -p /usr/local/etc/scanoss/api ; then
  echo "mkdir failed"
  exti 1
fi
if ! mkdir -p /var/log/scanoss/api ; then
  echo "mkdir failed"
  exit 1
fi
if [ "$RUNTIME_USER" != "root" ] ; then
  export LOG_DIR=/var/log/scanoss
  echo "Changing ownership of $LOG_DIR to $RUNTIME_USER ..."
  if ! chown -R $RUNTIME_USER $LOG_DIR ; then
    echo "chown of $LOG_DIR to $RUNTIME_USER failed"
    exit 1
  fi
  export LDB=/var/lib/ldb
  cur_dir=$(pwd)
  if ! cd $LDB ; then
    echo "cannot access $LDB"
    exit 1
  fi
  echo "Changing ownership of $LDB to $RUNTIME_USER ..."
  if ! chown -R $RUNTIME_USER . ; then
    echo "chown of $LDB to $RUNTIME_USER failed"
    exit 1
  fi
  cd $cur_dir
fi
export service_stopped=""
if [ -f /etc/systemd/system/scanoss-go-api.service ] ; then
  echo "Stopping scanoss-go-api service first..."
  if ! systemctl stop scanoss-go-api ; then
    echo "service stop failed"
    exit 1
  fi
  export service_stopped="true"
fi
echo "Copying service startup config..."
if ! cp scanoss-go-api.service /etc/systemd/system ; then
  echo "service copy failed"
  exti 1
fi
if ! cp scanoss-go-api.sh /usr/local/bin ; then
  echo "api startup script copy failed"
  exit 1
fi

CONF=app-config-prod.json
if [ -f $CONF ] ; then
  echo "Copying app config to /usr/local/etc/scanoss/api ..."
  if ! cp $CONF /usr/local/etc/scanoss/api ; then
    echo "copy $CONF failed"
    exit 1
  fi
else
  echo "Please put the config file into: /usr/local/etc/scanoss/api/$CONF"
fi
BINARY=scanoss-go-api
if [ -f $BINARY ] ; then
  echo "Copying app binary to /usr/local/bin ..."
  if ! cp $BINARY /usr/local/bin ; then
    echo "copy $BINARY failed"
    echo "Please make sure the service is stopped: systemctl stop scanoss-go-api"
    exit 1
  fi
else
  echo "Please copy the API binary file into: /usr/local/bin/$BINARY"
fi

SC_ENGINE=scanoss
if [ -f $SC_ENGINE ] ; then
  echo "Copying $SC_ENGINE binary to /usr/bin ..."
  if ! cp $SC_ENGINE /usr/bin ; then
    echo "copy $SC_ENGINE failed"
    exti 1
  fi
else
  if [ ! -f /usr/bin/$SC_ENGINE ] ; then
    echo "Please copy the $SC_ENGINE binary file into: /usr/bin/$SC_ENGINE"
  fi
fi
echo "Installation complete."
if [ "$service_stopped" == "true" ] ; then
  echo "Restarting service after install..."
  if ! systemctl start scanoss-go-api ; then
    echo "failed to restart service"
    exit 1
  fi
  systemctl status scanoss-go-api
fi
echo
echo "Review service config in: /usr/local/etc/scanoss/api/$CONF"
echo "Start the service using: systemctl start scanoss-go-api"
echo "Stop the service using: systemctl stop scanoss-go-api"
echo "Get service status using: systemctl status scanoss-go-api"
echo "Count the number of running scans using: ps -ef | grep \$(pgrep scanoss-go-api) | grep -v grep | wc -l"
echo
