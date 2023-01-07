#!/bin/bash
###
# SPDX-License-Identifier: GPL-2.0-or-later
#
# Copyright (C) 2018-2022 SCANOSS.COM
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
if [ "$1" == "-h" ] || [ "$2" == "-h" ]; then
  echo "SCANOSS engine help"
  echo " command options..."
  exit 0
fi
# Simulate getting file contents
if [ "$1" == "-k" ] || [ "$2" == "-k" ]; then
  if [ "x$3" == "x" ] ; then
    md5=$2
  else
    md5=$3
  fi
  echo "file contents: $md5"
  echo "line 2"
  echo "line 3"
  exit 0
fi

# Simulate getting SBOM attribution
if [ "$1" == "-a" ] || [ "$2" == "-a" ]; then
  if [ "x$3" == "x" ] ; then
    sbom=$2
  else
    sbom=$3
  fi
  echo "attribution: $sbom"
  echo "line 2"
  echo "line 3"
  exit 0
fi

# Simulate getting license details
if [ "$1" == "-l" ] || [ "$2" == "-l" ]; then
  if [ "x$3" == "x" ] ; then
    license=$2
  else
    license=$3
  fi
  echo "{\"$license\": {\"patent_hints\": "yes", \"copyleft\": \"no\", \"checklist_url\": \"https://www.osadl.org/fileadmin/checklists/unreflicenses/Apache-2.0.txt\",\"osadl_updated\": \"2022-12-12T13:47:00+00:00\"}}"
  exit 0
fi

# Simulate return a scan result
if [ "$1" == "-w" ] || [ "$2" == "-w" ] || [ "$3" == "-w" ] || [ "$4" == "-w" ] || [ "$5" == "-w" ] || [ "$6" == "-w" ] || [ "$7" == "-w" ]; then
  if [ "x$3" == "x" ] ; then
    scf=$2
  else
    scf=$3
  fi
  echo " {\"$scf\":[{\"id\": \"none\"}]}  "
  exit 0
fi

# Unknown command option, respond with error
echo "Unknown command option: $@"
exit 1
