package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/akhenakh/gozim"
	"github.com/rjohnsondev/golibstemmer"
	"github.com/szferi/gomdb"
)

var (
	path       = flag.String("path", "", "path for the zim file")
	lang       = flag.String("lang", "", "lang of the zim file to index")
	list       = flag.Bool("list", false, "List available stemming language")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	z          *zim.ZimReader
	stem       *stemmer.Stemmer
)

func inList(s []string, value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}

func main() {
	flag.Parse()

	if *list {
		list := stemmer.GetSupportedLanguages()
		for _, lang := range list {
			fmt.Println(lang)
		}
		os.Exit(0)
	}

	if *lang == "" {
		fmt.Println("no lang provided no stemming, could result in accurate indexes")
	} else {
		list := stemmer.GetSupportedLanguages()
		if !inList(list, *lang) {
			fmt.Println("Unsupported language")
			os.Exit(1)
		}

		stem, err := stemmer.NewStemmer(*lang)
		defer stem.Close()
		if err != nil {
			fmt.Println("Error creating stemmer: " + err.Error())
			os.Exit(1)
		}
	}

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
	for idx := range z.ListTitlesPtr() {
		offset := z.GetOffsetAtURLIdx(idx)
		a := z.GetArticleAt(offset)
		fmt.Println(a)

		slices := strings.Fields(a.Title)
		_ = len(slices)
		buf := new(bytes.Buffer)
		err := binary.Write(buf, binary.LittleEndian, idx)
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
