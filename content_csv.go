package main

import (
	"encoding/csv"
	"log"
	"os"
	"sync"

	"github.com/matst80/slask-finder/pkg/index"
)

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	csvReader.Comma = ';'
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}

func populateContentFromCsv(idx *index.ContentIndex, file string, group *sync.WaitGroup) {
	defer group.Done()
	// records := readCsvFile(file)
	// for i, record := range records {
	// 	if i == 0 {
	// 		log.Println("Importing content records")
	// 		continue
	// 	}
	// 	itm, err := index.ContentItemFromLine(record)
	// 	if err == nil {
	// 		idx.AddItem(itm)
	// 	} else {
	// 		log.Println(err)
	// 	}
	// }
	idx.Load()
}
