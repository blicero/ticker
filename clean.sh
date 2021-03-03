#!/bin/sh

cd $GOPATH/src/ticker/

rm -vf bak.ticker ticker dbg.build.log && du -sh . && git fsck --full && git reflog expire --expire=now && git gc --aggressive --prune=now && du -sh .

