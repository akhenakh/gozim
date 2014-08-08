gozim
=====

A Go native implementation for ZIM files. See http://akhenakh.github.io/gozim  

ZIM files are used mainly as offline wikipedia copies.

See http://openzim.org/wiki/ZIM_file_format and http://openzim.org/wiki/ZIM_File_Example

Wikipedia ZIMs can be downloaded from there http://download.kiwix.org/zim/

NOT FINISHED YET, API MAY CHANGE

TODO
====
Use a Pool to buffer ?  
Mmap 1st 2GB on 32 bits  
Selective Gzip encode response based on content type  
Search with title stemming  
func rather than if for getBytes  
