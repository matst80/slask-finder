package main

import (
	"encoding/csv"
	"log"
	"os"

	"github.com/matst80/slask-finder/pkg/index"
)

func readCsvFile(filePath string) [][]string {
	f, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal("Unable to parse file as CSV for "+filePath, err)
	}

	return records
}

func populateContentFromCsv(idx *index.ContentIndex) {
	records := readCsvFile("content.csv")
	for _, record := range records {

		itm, err := index.ContentItemFromLine(record)
		if err != nil {
			idx.AddItem(itm)
		}
	}
}
