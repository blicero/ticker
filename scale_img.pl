#!/usr/bin/perl
# -*- mode: cperl; coding: utf-8; -*-
# /home/krylon/go/src/ticker/scale_img.pl
# created at 08. 06. 2021 by Benjamin Walkenhorst
# (c) 2021 Benjamin Walkenhorst <krylon@gmx.net>
# Time-stamp: <2021-06-08 17:52:16 krylon>
#  Redistribution and use in source and binary forms, with or without
#  modification, are permitted provided that the following conditions
#  are met:
#  1. Redistributions of source code must retain the copyright
#     notice, this list of conditions and the following disclaimer.
#  2. Redistributions in binary form must reproduce the above copyright
#     notice, this list of conditions and the following disclaimer in the
#     documentation and/or other materials provided with the distribution.
#
#  THIS SOFTWARE IS PROVIDED BY BENJAMIN WALKENHORST ``AS IS'' AND
#  ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
#  IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
#  ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
#  FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
#  DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
#  OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
#  HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
#  LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
#  OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
#  SUCH DAMAGE.

use strict;
use warnings;
use diagnostics;
use utf8;
use feature qw(say);

use Carp;
use English '-no_match_vars';

use Readonly;

Readonly my $PATH => "$ENV{HOME}/.ticker.d/cache";
Readonly my $MAX_SIZE => 1048576; # 1 MiB

chdir $PATH
  or croak "Cannot chdir to $PATH: $OS_ERROR";

opendir my $dir, '.'
  or croak "Cannot open directory $PATH: $OS_ERROR";

while (my $img = readdir $dir) {
  if (-s $img > $MAX_SIZE) {
    say "Shrink $img";
    my $tmp_file = "small_$img";
    my $res = system 'magick', $img, '-scale', '25%', $tmp_file;
    if ($res == 0) {
      rename $tmp_file, $img;
    } else {
      say "Failed to shrink $img";
    }
    unlink $tmp_file;
  }
}

# Local Variables: #
# compile-command: "perl -c /home/krylon/go/src/ticker/scale_img.pl" #
# End: #
