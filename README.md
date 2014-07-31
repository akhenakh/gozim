gozim
=====

A Go native implementation for ZIM files.  

ZIM files are used mainly as offline wikipedia copies.

See http://openzim.org/wiki/ZIM_file_format and http://openzim.org/wiki/ZIM_File_Example

Wikipedia ZIMs can be downloaded from there http://download.kiwix.org/zim/

NOT FINISHED YET, API MAY CHANGE

TODO
====
Agressive cache reponse into browser  
Use a Pool to buffer ?  
Mmap 1st 2GB on 32 bits  
Gzip response  
Search with title stemming  
no index optimization (go to the half of the article count position then compare and so on)  

