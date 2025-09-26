package main

type DataType = int

const (
	KEY     = DataType(0)
	NUMBER  = DataType(1)
	DECIMAL = DataType(2)
)

type FieldData struct {
	Id          uint   `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	//Identifier  string   `json:"identifier"`
	Purpose   []string `json:"purpose"`
	Type      DataType `json:"type"`
	ItemCount int      `json:"itemCount"`
	LastSeen  int64    `json:"lastSeen"`
	Created   int64    `json:"created"`
}
