#!/bin/bash
###
# SPDX-License-Identifier: GPL-2.0-or-later
#
# Copyright (C) 2018-2023 SCANOSS.COM
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 2 of the License, or
# (at your option) any later version.
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
# You should have received a copy of the GNU General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.
###
#
# Package up the local scripts directory into a tar archive for deployment on a server
#
if [ "$1" = "-h" ] || [ "$1" = "-help" ] ; then
  echo "$0 [-help] <platform> <version>"
  echo "   Create a compressed archived of the scripts folder"
  echo "   <platform> platform the package is destined for (linux_amd64, linux_arm64)"
  echo "   <version> version number of the package"
  exit 1
fi
if [ -z "$1" ] ; then
  echo "ERROR: Please provide a package platform: linux_amd or linux_arm"
  exit 1
fi
if [ -z "$2" ] ; then
  echo "ERROR: Please provide a package version"
  exit 1
fi
if [ ! -d "scripts" ] ; then
  echo "ERROR: script folder does not exist."
  exit 1
fi
export COPYFILE_DISABLE=true  # Required if packaging on OSX
platform=$1
version=$2
build=1
tar_name="scanoss-go_${platform}_${version}-${build}.tgz"
# Get a unique archive name
while [ -f "$tar_name" ] ; do
  ((build++))
  tar_name="scanoss-go_${platform}_${version}-${build}.tgz"
done
if ! cp config/app-config-prod.json scripts ; then
  echo "ERROR copying sample prod config to scripts folder"
  exit 1
fi
echo "Packing scripts..."
if ! tar --format=ustar -cvzf "$tar_name" scripts ; then
  echo "ERROR packaging the scripts folder"
  exit 1
fi
echo
echo "Package archive name: $tar_name"
exit 0
