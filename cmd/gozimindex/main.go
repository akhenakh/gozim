package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/akhenakh/gozim"
	"github.com/szferi/gomdb"
)

var (
	path       = flag.String("path", "", "path for the zim file")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	z          *zim.ZimReader
)

func main() {
	flag.Parse()
	if *path == "" {
		panic(errors.New("provide a zim file path"))
	}

	z, err := zim.NewReader(*path, false)
	if err != nil {
		panic(err)
	}

	// create a directory for the db
	pathidx := *path + "idx"
	fmt.Println("creating directory", pathidx)
	err = os.Mkdir(pathidx, 0774)
	if err != nil {
		panic(err)
	}

	// open the db
	env, err := mdb.NewEnv()
	if err != nil {
		panic(err)
	}
	env.SetMapSize(1 << 30) // max file size

	err = env.Open(pathidx, 0, 0664)
	if err != nil {
		panic(err)
	}
	txn, _ := env.BeginTxn(nil, 0)
	dbi, _ := txn.DBIOpen(nil, 0)
	defer env.DBIClose(dbi)
	txn.Commit()

	i := 0

	txn, _ = env.BeginTxn(nil, 0)
	for a := range z.ListArticles() {
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.LittleEndian, a.URLPtr)
		if err != nil {
			fmt.Println("binary.Write failed:", err)
		}

		txn.Put(dbi, []byte(a.FullURL()), buf.Bytes(), 0)

		i++
		if i == 1000 {
			fmt.Print("*")
			i = 0
			txn.Commit()
			txn, _ = env.BeginTxn(nil, 0)
		}
	}
	txn.Commit()
	stat, _ := env.Stat()
	fmt.Println(stat.Entries)

}
