package index

import (
	"encoding/json"
	"testing"

	"github.com/matst80/slask-finder/pkg/types"
)

const mockItem = `{
    "url": "/product/gaming/datorkomponenter/processor-cpu/amd-ryzen-7-9800x3d-processor/861201",
    "releaseDate": "2024-11-07T00:00:00Z",
    "saleStatus": "ACT",
    "onlineSaleStatus": "ACT",
    "presaleDate": "2024-11-07T00:00:00Z",
    "restock": "2025-10-01T22:00:00Z",
    "img": "/image/dv_web_D18000127428576/861201/amd-ryzen-7-9800x3d-processor.jpg",
    "bp": "8-kärnig/16-trådar\nAM5 sockel, 120W TDP\nUpp till 5,2 GHz max boost",
    "lastUpdate": 1758882905507,
    "created": 1730851200000,
    "buyable": true,
    "description": "Höj ditt spelande och kreativa prestanda med AMD Ryzen 7 9800X3D-processorn. Med Zen 5-arkitektur, 8 kärnor, 16 trådar och en innovativ 3D V-Cache-teknik för förbättrad termisk prestanda och högre klockhastigheter.",
    "buyableInStore": true,
    "boxSize": "0002",
    "mp": 0,
    "stock": {
        "2001": "2",
        "2006": "1",
        "2008": "2",
        "2019": "2",
        "2031": "2",
        "2036": "1",
        "2037": "2",
        "2038": "1",
        "2237": "1"
    },
    "sku": "861201",
    "title": "AMD Ryzen 7 9800X3D processor",
    "id": 861201,
    "values": {
        "1": "ZHAW",
        "10": "Gaming",
        "11": "Datorkomponenter",
        "12": "Processor (CPU)",
        "16": "BESTSELLER",
        "2": "AMD",
        "23": "PT272",
        "24": "565",
        "3": "10+",
        "30": "PT103",
        "30203": 104,
        "30353": "9800X3D",
        "30879": "AMD Ryzen 7 9800X3D",
        "31": "PT262",
        "31009": 8,
        "31158": "Processor (CPU)",
        "31586": "AMD Ryzen 7",
        "32": "PT272",
        "32073": 16,
        "32075": 4700,
        "32076": 5200,
        "32103": "Socket AM5",
        "32152": "AMD",
        "32153": "AMD Radeon™ Graphics",
        "32198": "Ja",
        "34658": 24,
        "35": "59030",
        "35914": "5.0",
        "35922": "Dual-channel",
        "35978": 192,
        "35979": "DDR5",
        "35980": "5600",
        "35989": 120,
        "35990": 162,
        "36": "NORM",
        "36201": "AMD 9000-series",
        "36202": [
            "A620",
            "B650",
            "B650E",
            "X670",
            "X670E",
            "X870",
            "X870E"
        ],
        "36203": "Ja",
        "36204": "Nej",
        "36206": "Nej",
        "36333": "Ja",
        "37": "590",
        "4": 515900,
        "6": 48,
        "7": 43,
        "9": "Elgiganten"
    }
}`

func BenchmarkDecodeMockItem(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var item DataItem
		if err := json.Unmarshal([]byte(mockItem), &item); err != nil {
			b.Fatal(err)
		}
	}
}

func TestDataItem_GetFacetValue(t *testing.T) {
	var item DataItem
	if err := json.Unmarshal([]byte(mockItem), &item); err != nil {
		t.Fatal(err)
	}

	if _, ok := item.GetFacetValue(1); !ok {
		t.Error("expected to find facet value with id 1")
	}
}

func TestDataItem_GetFacets(t *testing.T) {
	var item DataItem
	if err := json.Unmarshal([]byte(mockItem), &item); err != nil {
		t.Fatal(err)
	}

	facets := item.GetFacets()
	if len(facets) == 0 {
		t.Error("expected to find facets")
	}
}

func TestDataItem_SetValue(t *testing.T) {
	item := DataItem{
		Fields: make(types.ItemFields),
	}
	item.SetValue(1, "test")

	if val, ok := item.GetFacetValue(1); !ok || val != "test" {
		t.Error("expected to find facet value with id 1 and value 'test'")
	}
}
