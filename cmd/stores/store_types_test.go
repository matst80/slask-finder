package main

import (
	"encoding/json"
	"math"
	"testing"
)

const storeJSONExample = `
{
  "displayName": "Elkjøp Lørenskog",
  "id": "1017",
  "url": "/store/elkjop-lorenskog",
  "address": {
    "street": "Solheimveien",
    "nr": "10",
    "zip": "1461",
    "city": "Lørenskog",
    "location": {
      "lat": 59.93517,
      "lng": 10.93726
    }
  },
  "openHours": {
    "days": [
      null,
      [
        [10, 0, 0],
        [21, 0, 0]
      ],
      [
        [10, 0, 0],
        [21, 0, 0]
      ],
      [
        [10, 0, 0],
        [21, 0, 0]
      ],
      [
        [10, 0, 0],
        [21, 0, 0]
      ],
      [
        [10, 0, 0],
        [21, 0, 0]
      ],
      [
        [10, 0, 0],
        [18, 0, 0]
      ]
    ],
    "other": [
      {
        "closed": false,
        "date": [2025, 12, 24],
        "time": [
          [10, 0, 0],
          [13, 0, 0]
        ],
        "text": "Julaften"
      },
      {
        "closed": true,
        "date": [2025, 12, 25],
        "text": "Første Juledag"
      },
      {
        "closed": true,
        "date": [2025, 12, 26],
        "text": "Andre Juledag"
      },
      {
        "closed": false,
        "date": [2025, 12, 31],
        "time": [
          [10, 0, 0],
          [16, 0, 0]
        ],
        "text": "Nyttårsaften"
      },
      {
        "closed": true,
        "date": [2026, 1, 1],
        "text": "Første Nyttårsdag"
      },
      {
        "closed": true,
        "date": [2026, 4, 2],
        "text": "Skjærtorsdag"
      },
      {
        "closed": true,
        "date": [2026, 4, 3],
        "text": "Langfredag"
      },
      {
        "closed": true,
        "date": [2026, 4, 5],
        "text": "Første Påskedag"
      },
      {
        "closed": true,
        "date": [2026, 4, 6],
        "text": "Andre Påskedag"
      },
      {
        "closed": true,
        "date": [2026, 5, 1],
        "text": "1. mai"
      },
      {
        "closed": true,
        "date": [2026, 5, 14],
        "text": "Kristi Himmelfartsdag"
      },
      {
        "closed": true,
        "date": [2026, 5, 17],
        "text": "17. mai"
      },
      {
        "closed": true,
        "date": [2026, 5, 24],
        "text": "Første Pinsedag"
      },
      {
        "closed": true,
        "date": [2026, 5, 25],
        "text": "Andre Pinsedag"
      }
    ]
  },
  "shipToStore": false,
  "collectAtStore": {
    "prePaid": true,
    "leadTime": 60
  },
  "onlineId": "1092",
  "shipFromStore": {
    "post": true,
    "home": true,
    "leadTime": 0
  }
}
`

func TestStoreJSONUnmarshal(t *testing.T) {
	var s Store
	if err := json.Unmarshal([]byte(storeJSONExample), &s); err != nil {
		t.Fatalf("failed to unmarshal store JSON: %v", err)
	}

	// Top-level fields
	if s.DisplayName != "Elkjøp Lørenskog" {
		t.Fatalf("DisplayName mismatch: %s", s.DisplayName)
	}
	if s.ID != "1017" {
		t.Fatalf("ID mismatch: %s", s.ID)
	}
	if s.URL != "/store/elkjop-lorenskog" {
		t.Fatalf("URL mismatch: %s", s.URL)
	}

	// Address
	if s.Address.Street != "Solheimveien" ||
		s.Address.Nr != "10" ||
		s.Address.Zip != "1461" ||
		s.Address.City != "Lørenskog" {
		t.Fatalf("Address mismatch: %+v", s.Address)
	}

	// Location (allow tiny float differences)
	if math.Abs(s.Address.Location.Latitude-59.93517) > 1e-6 ||
		math.Abs(s.Address.Location.Longitude-10.93726) > 1e-6 {
		t.Fatalf("Location mismatch: %+v", s.Address.Location)
	}

	// OpenHours.Days
	if len(s.OpenHours.Days) != 7 {
		t.Fatalf("expected 7 day entries, got %d", len(s.OpenHours.Days))
	}
	if s.OpenHours.Days[0] != nil {
		t.Fatalf("expected first day to be nil (closed), got %#v", s.OpenHours.Days[0])
	}

	// Helper to assert a StoreDay
	assertDay := func(idx int, openH, openM, openS, closeH, closeM, closeS int) {
		d := s.OpenHours.Days[idx]
		if d == nil {
			t.Fatalf("day %d unexpectedly nil", idx)
		}
		if d.Open.Hour != openH || d.Open.Minute != openM || d.Open.Second != openS {
			t.Fatalf("day %d open mismatch: %+v", idx, d.Open)
		}
		if d.Close.Hour != closeH || d.Close.Minute != closeM || d.Close.Second != closeS {
			t.Fatalf("day %d close mismatch: %+v", idx, d.Close)
		}
	}

	// Days 1-5 should be 10:00:00 - 21:00:00, day 6 is 10:00:00 - 18:00:00
	for i := 1; i <= 5; i++ {
		assertDay(i, 10, 0, 0, 21, 0, 0)
	}
	assertDay(6, 10, 0, 0, 18, 0, 0)

	// OpenHours.Other
	if len(s.OpenHours.Other) != 14 {
		t.Fatalf("expected 14 'other' entries, got %d", len(s.OpenHours.Other))
	}

	// Validate a few specific 'other' entries
	first := s.OpenHours.Other[0]
	if first.Closed {
		t.Fatalf("first special day should not be closed")
	}
	if first.Date.Year != 2025 || first.Date.Month != 12 || first.Date.Day != 24 {
		t.Fatalf("first special date mismatch: %+v", first.Date)
	}
	if len(first.Time) != 2 {
		t.Fatalf("expected first special time slice length 2, got %d", len(first.Time))
	}
	if first.Time[0].Hour != 10 || first.Time[1].Hour != 13 {
		t.Fatalf("first special times mismatch: %+v", first.Time)
	}

	// A closed day with no times
	closedXmas := s.OpenHours.Other[1]
	if !closedXmas.Closed || len(closedXmas.Time) != 0 {
		t.Fatalf("expected closed day with no time: %+v", closedXmas)
	}
	if closedXmas.Date.Year != 2025 || closedXmas.Date.Day != 25 {
		t.Fatalf("closedXmas date mismatch: %+v", closedXmas.Date)
	}

	// New Year's Eve partial hours
	var nyeFound bool
	for _, o := range s.OpenHours.Other {
		if o.Text == "Nyttårsaften" {
			nyeFound = true
			if o.Date.Year != 2025 || o.Date.Month != 12 || o.Date.Day != 31 {
				t.Fatalf("Nyttårsaften date mismatch: %+v", o.Date)
			}
			if len(o.Time) != 2 || o.Time[0].Hour != 10 || o.Time[1].Hour != 16 {
				t.Fatalf("Nyttårsaften times mismatch: %+v", o.Time)
			}
		}
	}
	if !nyeFound {
		t.Fatalf("did not find Nyttårsaften entry")
	}

	// Ship / collect / online fields
	if s.ShipToStore != false {
		t.Fatalf("ShipToStore mismatch: %v", s.ShipToStore)
	}
	if !s.CollectAtStore.PrePaid || s.CollectAtStore.LeadTime != 60 {
		t.Fatalf("CollectAtStore mismatch: %+v", s.CollectAtStore)
	}
	if s.OnlineID != "1092" {
		t.Fatalf("OnlineID mismatch: %s", s.OnlineID)
	}
	if !s.ShipFromStore.Post || !s.ShipFromStore.Home || s.ShipFromStore.LeadTime != 0 {
		t.Fatalf("ShipFromStore mismatch: %+v", s.ShipFromStore)
	}
}
