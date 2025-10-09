package main

import (
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/matst80/slask-finder/pkg/common"
	"github.com/matst80/slask-finder/pkg/storage"
	"github.com/oschwald/geoip2-golang/v2"
)

var country = "se"

func init() {
	c, ok := os.LookupEnv("COUNTRY")
	if ok {
		country = c
	}
}

func getIpFromRequest(r *http.Request) (*netip.Addr, error) {
	// 1. Allow explicit IP override for debugging via ?ip=1.2.3.4
	rawIP := r.URL.Query().Get("ip")

	// 2. If not provided, try common proxy headers
	if rawIP == "" {
		for _, h := range []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"} {
			if v := r.Header.Get(h); v != "" {
				if h == "X-Forwarded-For" {
					// May be a list; take the first
					if idx := strings.IndexByte(v, ','); idx >= 0 {
						v = v[:idx]
					}
				}
				rawIP = strings.TrimSpace(v)
				break
			}
		}
	}

	// 3. Fall back to RemoteAddr
	if rawIP == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			rawIP = host
		} else {
			rawIP = r.RemoteAddr
		}
	}

	addr, err := netip.ParseAddr(rawIP)
	if err != nil {
		return nil, err
	}
	return &addr, nil

}

func parseCoordinate(s string, min, max float64) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if v < min || v > max {
		return 0, strconv.ErrRange
	}
	return v, nil
}

func getLocationFromRequest(r *http.Request, db *geoip2.Reader, zipMap map[string]Location) (*Location, error) {
	latString := r.URL.Query().Get("lat")
	lonString := r.URL.Query().Get("lon")
	postalCode := r.URL.Query().Get("zip")
	if latString != "" && lonString != "" {
		lat, err := parseCoordinate(latString, -90, 90)
		if err != nil {
			return nil, err
		}
		lon, err := parseCoordinate(lonString, -180, 180)
		if err != nil {
			return nil, err
		}
		return &Location{Latitude: lat, Longitude: lon}, nil
	}
	locationCookie, err := r.Cookie("location")
	if err == nil && locationCookie.Value != "" {
		parts := strings.Split(locationCookie.Value, ";")
		if len(parts) >= 2 {
			lat, err := parseCoordinate(parts[0], -90, 90)
			if err == nil {
				lon, err := parseCoordinate(parts[1], -180, 180)
				if err == nil {
					return &Location{Latitude: lat, Longitude: lon}, nil
				}
			}
			if len(parts) >= 3 && postalCode == "" {
				postalCode = parts[2]
			}
		}
	}

	if postalCode != "" {
		if location, ok := zipMap[postalCode]; ok {
			return &location, nil
		}
	}

	parsedIP, err := getIpFromRequest(r)
	if err != nil {
		return nil, err
	}

	rec, err := db.City(*parsedIP)
	if err != nil {
		log.Printf("geoip lookup failed for %s: %v", parsedIP, err)
		return nil, err
	}
	if !rec.Location.HasCoordinates() || rec.Location.Latitude == nil || rec.Location.Longitude == nil {
		return nil, nil
	}
	return &Location{Latitude: *rec.Location.Latitude, Longitude: *rec.Location.Longitude}, nil
}

func main() {
	diskStorage := storage.NewDiskStorage(country, "data")
	stores := []Store{}
	if err := diskStorage.LoadJson(&stores, "stores.json"); err != nil {
		panic(err)
	}

	db, err := geoip2.Open("data/GeoLite2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	zip2Location := make(map[string]Location)

	f, _ := os.Open("data/se/postcode-map.csv")
	defer f.Close()
	ctx := context.Background()
	err = StreamPostalCodeLocations(ctx, f, func(p PostalCodeLocation) error {
		// fmt.Printf("%s %s (%f,%f)\n", p.PostalCode, p.City, p.Location.Latitude, p.Location.Longitude)
		zip2Location[p.PostalCode] = p.Location
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/stores", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Expires", time.Now().Add(time.Hour).Format(time.RFC1123))
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(stores); err != nil {
			log.Printf("Could not stream stores %v", err)
		}
		// if _, err := diskStorage.StreamContent(w, "stores.json"); err != nil {
		// 	log.Printf("Failed to stream stores.json: %v", err)
		// }
	})
	mux.HandleFunc("GET /api/lookup", func(w http.ResponseWriter, r *http.Request) {
		parsedIP, err := getIpFromRequest(r)
		if err != nil {
			http.Error(w, "invalid ip", http.StatusBadRequest)
			return
		}

		rec, err := db.City(*parsedIP)
		if err != nil {
			log.Printf("geoip lookup failed for %s: %v", parsedIP, err)
			http.Error(w, "lookup failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("Expires", time.Now().Add(time.Hour).Format(time.RFC1123))
		w.Header().Set("X-Detected-IP", parsedIP.String())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(rec); err != nil {
			log.Printf("failed to encode location response: %v", err)
		}
	})
	mux.HandleFunc("GET /api/location", func(w http.ResponseWriter, r *http.Request) {
		location, err := getLocationFromRequest(r, db, zip2Location)
		if err != nil {
			http.Error(w, "no location found", http.StatusBadRequest)
			return
		}
		if location == nil {
			http.Error(w, "could not determine location", http.StatusBadRequest)
			return
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(location); err != nil {
			log.Printf("Could not stream location %v", err)
		}
	})
	mux.HandleFunc("GET /api/closest-stores", func(w http.ResponseWriter, r *http.Request) {
		location, err := getLocationFromRequest(r, db, zip2Location)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// idOnly := r.URL.Query().Get("ids")

		if location == nil {
			http.Error(w, "could not determine location", http.StatusBadRequest)
			return
		}

		distances := make([]StoreDistance, 0, len(stores))
		for _, store := range stores {
			s := &store
			d := location.DistanceTo(*s.Address.Location)
			distances = append(distances, StoreDistance{Store: s, Distance: d})
		}
		slices.SortFunc(distances, func(a, b StoreDistance) int {
			if a.Distance < b.Distance {
				return -1
			} else if a.Distance > b.Distance {
				return 1
			}
			return 0
		})

		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(distances); err != nil {
			log.Printf("Could not stream closest stores %v", err)
		}

	})
	cfg := common.LoadTimeoutConfig(common.TimeoutConfig{
		ReadHeader: 5 * time.Second,
		Read:       15 * time.Second,
		Write:      30 * time.Second,
		Idle:       60 * time.Second,
		Shutdown:   15 * time.Second,
		Hook:       5 * time.Second,
	})
	server := common.NewServerWithTimeouts(&http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: cfg.ReadHeader}, cfg)

	common.RunServerWithShutdown(server, "reader server", cfg.Shutdown, cfg.Hook)
}
