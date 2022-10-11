#!/bin/sh
# Time-stamp: <2022-10-10 19:39:19 krylon>

/usr/bin/env perl -pi'.bak' -e 's{"ticker/(\w+)"}{"github.com/blicero/ticker/$1"}' $@


