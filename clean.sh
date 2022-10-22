#!/bin/sh
# Time-stamp: <2022-10-22 17:11:55 krylon>

cd $GOPATH/src/github.com/blicero/ticker/

rm -vf bak.ticker ticker dbg.build.log && du -sh . && git fsck --full && git reflog expire --expire=now && git gc --aggressive --prune=now && du -sh .

