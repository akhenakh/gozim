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

cross-compilation
=================

For easy cross-compilation a `!cgo` build version uses a pure go library for lzma parsing.
The pure go library is around ~2.5x slower in benchmarks so compile on your target OS if
performance is important.

running
=======

Optionally, build an index file: `gozimindex -path=yourzimfile.zim -indexPath=yourzimfile.idx`

Start the gozim server: `gozimhttpd -path=yourzimfile.zim [-index=yourzimfile.idx]`

TODO
====
Mmap 1st 2GB on 32 bits
Selective Gzip encode response based on content type
func rather than if for getBytes

Docker
======

To use the Dockerfile, first create a docker image: `docker build -t gozim .`

In order to run the Docker image, you need to set an environment variable called ZIM_PATH for the path to the .zim file, and optionally another environment variable called INDEX_PATH for the path to the index files.
One way to do this is to mount a directory on the local machine that contains these files as a volume of your docker container at run time:

`docker run -it --rm -p 8080:8080 -v /path/to/directory/containing/zim/file:/go/data -e "ZIM_PATH=/go/data/wikipedia.zim" -e "INDEX_PATH=/go/data/wikipedia.zim.idx" gozim`

The above command will run gozimhttpd on port 8080 of the host machine.  It will serve the zim file /path/to/directory/containing/zim/file/wikipedia.zim using the index /path/to/directory/containing/zim/file/wikipedia.zim.idx.

In order to create an index file using the docker container, you still need to mount the directory on the local machine that contains the zim file, and you need to change the entrypoint for the docker image and pass the path to the zim file and the output index files:

`docker run -it --rm -v /path/to/directory/containing/zim/file:/go/data --entrypoint=/bin/bash gozim -c "/go/bin/gozimindex -path=/go/data/wikipedia.zim -index=/go/data/wikipedia.zim.idx"`
