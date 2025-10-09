package main

import "math"

type StoreShipFromStore struct {
	Post     bool `json:"post"`
	Home     bool `json:"home"`
	LeadTime int  `json:"leadTime"`
}

type StoreCollectAtStore struct {
	PrePaid  bool `json:"prePaid"`
	LeadTime int  `json:"leadTime"`
}

type Location struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

func (loc Location) IsZero() bool {
	return loc.Latitude == 0 && loc.Longitude == 0
}

func (loc Location) DistanceTo(other Location) float64 {
	const R = 6371e3 // Earth radius in meters
	lat1 := loc.Latitude * (3.141592653589793 / 180)
	lat2 := other.Latitude * (3.141592653589793 / 180)
	dLat := (other.Latitude - loc.Latitude) * (3.141592653589793 / 180)
	dLon := (other.Longitude - loc.Longitude) * (3.141592653589793 / 180)

	a := (math.Sin(dLat/2) * math.Sin(dLat/2)) + math.Cos(lat1)*math.Cos(lat2)*(math.Sin(dLon/2)*math.Sin(dLon/2))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c // in meters
	return distance / 1000.0
}

type StoreDistance struct {
	*Store
	Distance float64 `json:"distance"`
}

type Store struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
	URL         string `json:"url"`
	Address     struct {
		Street   string    `json:"street"`
		Nr       string    `json:"nr"`
		Zip      string    `json:"zip"`
		City     string    `json:"city"`
		Location *Location `json:"location"`
	} `json:"address"`
	OpenHours struct {
		Days  []*StoreDay `json:"days"` // pointer slice to allow null entries in JSON
		Other []struct {
			Closed bool        `json:"closed"`
			Date   StoreDate   `json:"date"`
			Time   []StoreTime `json:"time,omitempty"`
			Text   string      `json:"text"`
		} `json:"other"`
	} `json:"openHours"`
	ShipToStore    bool                 `json:"shipToStore"`
	CollectAtStore *StoreCollectAtStore `json:"collectAtStore,omitempty"`
	OnlineID       string               `json:"onlineId"`
	ShipFromStore  *StoreShipFromStore  `json:"shipFromStore,omitempty"`
}
