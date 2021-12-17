[![Build Status](https://travis-ci.org/akhenakh/gozim.svg?branch=master)](https://travis-ci.org/akhenakh/gozim)

gozim
=====

A Go native implementation for ZIM files. See http://akhenakh.github.io/gozim

ZIM files are used mainly as offline wikipedia copies.

See http://openzim.org/wiki/ZIM_file_format and http://openzim.org/wiki/ZIM_File_Example

Wikipedia/Wikinews/... ZIMs can be downloaded from there http://download.kiwix.org/zim/

![ScreenShot](/shots/browse.jpg)
![ScreenShot](/shots/search.jpg)

build and installation
======================

On Ubuntu/Debian you need those packages to compile gozim
```bash
apt install git liblzma-dev mercurial build-essential
```

Then you can build gozimindex and gozimhttpd with 
```bash
make build
```

cross-compilation
=================

For easy cross-compilation a `!cgo` build version uses a pure go library for lzma parsing.
The pure go library is around ~2.5x slower in benchmarks so compile on your target OS if
performance is important.

running
=======

Optionally, build an index file: `gozimindex -path=yourzimfile.zim -index=yourzimfile.idx`

Start the gozim server: `gozimhttpd -path=yourzimfile.zim [-index=yourzimfile.idx]`

TODO
====
Mmap 1st 2GB on 32 bits
Selective Gzip encode response based on content type
func rather than if for getBytes

