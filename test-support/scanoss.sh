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

# Simulate getting file contents
if [ "$1" == "-h" ] || [ "$2" == "-h" ] || [ "$1" == "-help" ] || [ "$2" == "-help" ] ; then
  echo "SCANOSS engine simulator help"
  echo " command options..."
  exit 0
fi

# Simulate getting file contents
if [ "$1" == "-k" ] || [ "$2" == "-k" ] || [ "$3" == "-k" ] ; then
  for i in "$@"; do :; done
  md5=$i
  echo "file contents: $md5"
  echo "line 2"
  echo "line 3"
  exit 0
fi

# Simulate getting SBOM attribution
if [ "$1" == "-a" ] || [ "$2" == "-a" ] || [ "$3" == "-a" ] ; then
  for i in "$@"; do :; done
  sbom=$i
  echo "attribution: $sbom"
  echo "line 2"
  echo "line 3"
  exit 0
fi

# Simulate getting license details
if [ "$1" == "-l" ] || [ "$2" == "-l" ] || [ "$3" == "-l" ] ; then
  for i in "$@"; do :; done
  license=$i
  echo "{\"$license\": {\"patent_hints\": \"yes\", \"copyleft\": \"no\", \"checklist_url\": \"https://www.osadl.org/fileadmin/checklists/unreflicenses/Apache-2.0.txt\",\"osadl_updated\": \"2022-12-12T13:47:00+00:00\"}}"
  exit 0
fi

# Simulate invalid kb name
if [ "$1" == "-n" ] || [ "$2" == "-n" ] || [ "$3" == "-n" ] || [ "$4" == "-n" ] || [ "$5" == "-n" ] || [ "$6" == "-n" ] || [ "$7" == "-n" ] || [ "$8" == "-n" ]; then
  for i in "$@"; do :; done
  scf=$i
  echo "{Error: file and url tables must be present in $scf KB in order to proceed with the scan"
  exit 1
fi

# Simulate return a scan result
if [ "$1" == "-w" ] || [ "$2" == "-w" ] || [ "$3" == "-w" ] || [ "$4" == "-w" ] || [ "$5" == "-w" ] || [ "$6" == "-w" ] || [ "$7" == "-w" ] || [ "$8" == "-w" ]; then
  for i in "$@"; do :; done
  scf=$i
  echo " {\"$scf\":[{\"id\": \"none\", \"server\": { \"kb_version\": {\"daily\": \"23.08.09\", \"monthly\": \"23.07\"}, \"version\": \"5.2.7\"}}]}  "
  exit 0
fi

# Unknown command option, respond with error
echo "Unknown command option: $*"
exit 1
