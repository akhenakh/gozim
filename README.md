[![Build status](https://github.com/akhenakh/gozim/actions/workflows/build.yml/badge.svg)](https://github.com/akhenakh/gozim/actions/workflows/build.yml)
[![Lint status](https://github.com/akhenakh/gozim/actions/workflows/lint.yml/badge.svg)](https://github.com/akhenakh/gozim/actions/workflows/lint.yml)

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

On Ubuntu/Debian youn need those packages to compile gozim
```
apt-get install git liblzma-dev mercurial build-essential
```

For the indexer bleve to work properly it's recommended that you use leveldb as storage.
```
go get -u -v -tags all github.com/blevesearch/bleve/...
```

Gozim http server is using go.rice to embed html/css in the binary install the rice command
```
go get github.com/GeertJohan/go.rice
go get github.com/GeertJohan/go.rice/rice
go install github.com/GeertJohan/go.rice
go install github.com/GeertJohan/go.rice/rice
```

Get and build the gozim executables
```bash
go get github.com/akhenakh/gozim/...
cd $GOPATH/src/github.com/akhenakh/gozim
go build github.com/akhenakh/gozim/cmd/gozimhttpd
go build github.com/akhenakh/gozim/cmd/gozimindex
```

After build gozimhttpd command run to embed the files:
```
rice append --exec gozimhttpd
```

On OSX:
```
CGO_CFLAGS=`pkg-config --cflags liblzma` go build 
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

