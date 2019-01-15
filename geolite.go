package main

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"sync/atomic"

	"github.com/demskie/subnetmath"
)

// https://dev.maxmind.com/geoip/geoip2/geolite2/

const cityLocationsPath = basePath + "GeoLite2-City-Locations-en.csv"
const cityBlocksV4Path = basePath + "GeoLite2-City-Blocks-IPv4.csv"
const cityBlocksV6Path = basePath + "GeoLite2-City-Blocks-IPv6.csv"

type GeoPosition struct {
	Latitude  float64      `json:"latitude"`
	Longitude float64      `json:"longitude"`
	Location  *GeoLocation `json:"location"`
}

type GeoLocation struct {
	CityName    string `json:"cityName"`
	SubdivName  string `json:"subdivName"`
	CountryISO  string `json:"countryISO"`
	CountryName string `json:"countryName"`
	IsPartOfEU  bool   `json:"isPartOfEU"`
}

func ingestGeoliteData(tree *Tree) {
	locationMap := getAllGeoLocations()
	gopath, _ := os.LookupEnv("GOPATH")
	for _, blocks := range []string{cityBlocksV4Path, cityBlocksV6Path} {
		txtFile, err := os.Open(path.Join(gopath, blocks))
		if err != nil {
			log.Fatalf("unable to ingest city location data because: %v", err)
		}
		reader := csv.NewReader(bufio.NewReader(txtFile))
		reader.Read() // skip the first line
		for i := 1; true; i++ {
			lineColumns, err := reader.Read()
			if err == io.EOF {
				break
			}
			if len(lineColumns) > 9 {
				network := subnetmath.ParseNetworkCIDR(lineColumns[0])
				if network == nil {
					log.Fatalf("network '%v' is not valid", lineColumns[0])
				}
				if lineColumns[1] == "" && lineColumns[2] == "" {
					continue
				}
				geoLocation := locationMap[lineColumns[1]]
				if geoLocation == nil {
					geoLocation = locationMap[lineColumns[2]]
					if geoLocation == nil {
						log.Fatalf("geoname_id '%v' and '%v' on line %v not found in city locations",
							lineColumns[1], lineColumns[2], i)
					}
				}
				latitude, latError := strconv.ParseFloat(lineColumns[7], 64)
				longitude, longError := strconv.ParseFloat(lineColumns[8], 64)
				if latError != nil || longError != nil {
					coarsePosition, exists := coarseCountryPositions[geoLocation.CountryISO]
					if !exists {
						log.Fatalf("latitude '%v' is not valid and countrycode '%v' is unsupported",
							lineColumns[7], geoLocation.CountryISO)
					}
					latitude = coarsePosition.Latitude
					longitude = coarsePosition.Longitude
				}
				geoPosition := &GeoPosition{
					Latitude:  latitude,
					Longitude: longitude,
					Location:  geoLocation,
				}
				atomic.AddUint64(&counters.rate, 1)
				tree.insert(geoPosition, network)
			}
		}
	}
}

func getAllGeoLocations() map[string]*GeoLocation {
	result := map[string]*GeoLocation{}
	gopath, _ := os.LookupEnv("GOPATH")
	txtFile, err := os.Open(path.Join(gopath, cityLocationsPath))
	if err != nil {
		log.Fatalf("unable to ingest city location data because: %v", err)
	}
	reader := csv.NewReader(bufio.NewReader(txtFile))
	reader.Read() // skip the first line
	for {
		lineColumns, err := reader.Read()
		if err == io.EOF {
			break
		}
		if len(lineColumns) > 13 {
			result[lineColumns[0]] = &GeoLocation{
				CityName:    lineColumns[10],
				SubdivName:  lineColumns[7],
				CountryISO:  lineColumns[4],
				CountryName: lineColumns[5],
				IsPartOfEU:  (lineColumns[13] != "0"),
			}
		}
	}
	return result
}

var coarseCountryPositions = map[string]*GeoPosition{
	"AL": &GeoPosition{41, 20, nil},
	"DZ": &GeoPosition{28, 3, nil},
	"AS": &GeoPosition{-14.3333, -170, nil},
	"AD": &GeoPosition{42.5, 1.6, nil},
	"AO": &GeoPosition{-12.5, 18.5, nil},
	"AI": &GeoPosition{18.25, -63.1667, nil},
	"AQ": &GeoPosition{-90, 0, nil},
	"AG": &GeoPosition{17.05, -61.8, nil},
	"AR": &GeoPosition{-34, -64, nil},
	"AM": &GeoPosition{40, 45, nil},
	"AW": &GeoPosition{12.5, -69.9667, nil},
	"AU": &GeoPosition{-27, 133, nil},
	"AT": &GeoPosition{47.3333, 13.3333, nil},
	"AZ": &GeoPosition{40.5, 47.5, nil},
	"BS": &GeoPosition{24.25, -76, nil},
	"BH": &GeoPosition{26, 50.55, nil},
	"BD": &GeoPosition{24, 90, nil},
	"BB": &GeoPosition{13.1667, -59.5333, nil},
	"BY": &GeoPosition{53, 28, nil},
	"BE": &GeoPosition{50.8333, 4, nil},
	"BZ": &GeoPosition{17.25, -88.75, nil},
	"BJ": &GeoPosition{9.5, 2.25, nil},
	"BM": &GeoPosition{32.3333, -64.75, nil},
	"BT": &GeoPosition{27.5, 90.5, nil},
	"BO": &GeoPosition{-17, -65, nil},
	"BA": &GeoPosition{44, 18, nil},
	"BW": &GeoPosition{-22, 24, nil},
	"BV": &GeoPosition{-54.4333, 3.4, nil},
	"BR": &GeoPosition{-10, -55, nil},
	"IO": &GeoPosition{-6, 71.5, nil},
	"BN": &GeoPosition{4.5, 114.6667, nil},
	"BG": &GeoPosition{43, 25, nil},
	"BF": &GeoPosition{13, -2, nil},
	"BI": &GeoPosition{-3.5, 30, nil},
	"KH": &GeoPosition{13, 105, nil},
	"CM": &GeoPosition{6, 12, nil},
	"CA": &GeoPosition{60, -95, nil},
	"CV": &GeoPosition{16, -24, nil},
	"KY": &GeoPosition{19.5, -80.5, nil},
	"CF": &GeoPosition{7, 21, nil},
	"TD": &GeoPosition{15, 19, nil},
	"CL": &GeoPosition{-30, -71, nil},
	"CN": &GeoPosition{35, 105, nil},
	"CX": &GeoPosition{-10.5, 105.6667, nil},
	"CC": &GeoPosition{-12.5, 96.8333, nil},
	"CO": &GeoPosition{4, -72, nil},
	"KM": &GeoPosition{-12.1667, 44.25, nil},
	"CG": &GeoPosition{-1, 15, nil},
	"CD": &GeoPosition{0, 25, nil},
	"CK": &GeoPosition{-21.2333, -159.7667, nil},
	"CR": &GeoPosition{10, -84, nil},
	"CI": &GeoPosition{8, -5, nil},
	"HR": &GeoPosition{45.1667, 15.5, nil},
	"CU": &GeoPosition{21.5, -80, nil},
	"CY": &GeoPosition{35, 33, nil},
	"CZ": &GeoPosition{49.75, 15.5, nil},
	"DK": &GeoPosition{56, 10, nil},
	"DJ": &GeoPosition{11.5, 43, nil},
	"DM": &GeoPosition{15.4167, -61.3333, nil},
	"DO": &GeoPosition{19, -70.6667, nil},
	"EC": &GeoPosition{-2, -77.5, nil},
	"EG": &GeoPosition{27, 30, nil},
	"SV": &GeoPosition{13.8333, -88.9167, nil},
	"GQ": &GeoPosition{2, 10, nil},
	"ER": &GeoPosition{15, 39, nil},
	"EE": &GeoPosition{59, 26, nil},
	"ET": &GeoPosition{8, 38, nil},
	"FK": &GeoPosition{-51.75, -59, nil},
	"FO": &GeoPosition{62, -7, nil},
	"FJ": &GeoPosition{-18, 175, nil},
	"FI": &GeoPosition{64, 26, nil},
	"FR": &GeoPosition{46, 2, nil},
	"GF": &GeoPosition{4, -53, nil},
	"PF": &GeoPosition{-15, -140, nil},
	"TF": &GeoPosition{-43, 67, nil},
	"GA": &GeoPosition{-1, 11.75, nil},
	"GM": &GeoPosition{13.4667, -16.5667, nil},
	"GE": &GeoPosition{42, 43.5, nil},
	"DE": &GeoPosition{51, 9, nil},
	"GH": &GeoPosition{8, -2, nil},
	"GI": &GeoPosition{36.1833, -5.3667, nil},
	"GR": &GeoPosition{39, 22, nil},
	"GL": &GeoPosition{72, -40, nil},
	"GD": &GeoPosition{12.1167, -61.6667, nil},
	"GP": &GeoPosition{16.25, -61.5833, nil},
	"GU": &GeoPosition{13.4667, 144.7833, nil},
	"GT": &GeoPosition{15.5, -90.25, nil},
	"GG": &GeoPosition{49.5, -2.56, nil},
	"GN": &GeoPosition{11, -10, nil},
	"GW": &GeoPosition{12, -15, nil},
	"GY": &GeoPosition{5, -59, nil},
	"HT": &GeoPosition{19, -72.4167, nil},
	"HM": &GeoPosition{-53.1, 72.5167, nil},
	"VA": &GeoPosition{41.9, 12.45, nil},
	"HN": &GeoPosition{15, -86.5, nil},
	"HK": &GeoPosition{22.25, 114.1667, nil},
	"HU": &GeoPosition{47, 20, nil},
	"IS": &GeoPosition{65, -18, nil},
	"IN": &GeoPosition{20, 77, nil},
	"ID": &GeoPosition{-5, 120, nil},
	"IR": &GeoPosition{32, 53, nil},
	"IQ": &GeoPosition{33, 44, nil},
	"IE": &GeoPosition{53, -8, nil},
	"IM": &GeoPosition{54.23, -4.55, nil},
	"IL": &GeoPosition{31.5, 34.75, nil},
	"IT": &GeoPosition{42.8333, 12.8333, nil},
	"JM": &GeoPosition{18.25, -77.5, nil},
	"JP": &GeoPosition{36, 138, nil},
	"JE": &GeoPosition{49.21, -2.13, nil},
	"JO": &GeoPosition{31, 36, nil},
	"KZ": &GeoPosition{48, 68, nil},
	"KE": &GeoPosition{1, 38, nil},
	"KI": &GeoPosition{1.4167, 173, nil},
	"KP": &GeoPosition{40, 127, nil},
	"KR": &GeoPosition{37, 127.5, nil},
	"KW": &GeoPosition{29.3375, 47.6581, nil},
	"KG": &GeoPosition{41, 75, nil},
	"LA": &GeoPosition{18, 105, nil},
	"LV": &GeoPosition{57, 25, nil},
	"LB": &GeoPosition{33.8333, 35.8333, nil},
	"LS": &GeoPosition{-29.5, 28.5, nil},
	"LR": &GeoPosition{6.5, -9.5, nil},
	"LY": &GeoPosition{25, 17, nil},
	"LI": &GeoPosition{47.1667, 9.5333, nil},
	"LT": &GeoPosition{56, 24, nil},
	"LU": &GeoPosition{49.75, 6.1667, nil},
	"MO": &GeoPosition{22.1667, 113.55, nil},
	"MK": &GeoPosition{41.8333, 22, nil},
	"MG": &GeoPosition{-20, 47, nil},
	"MW": &GeoPosition{-13.5, 34, nil},
	"MY": &GeoPosition{2.5, 112.5, nil},
	"MV": &GeoPosition{3.25, 73, nil},
	"ML": &GeoPosition{17, -4, nil},
	"MT": &GeoPosition{35.8333, 14.5833, nil},
	"MH": &GeoPosition{9, 168, nil},
	"MQ": &GeoPosition{14.6667, -61, nil},
	"MR": &GeoPosition{20, -12, nil},
	"MU": &GeoPosition{-20.2833, 57.55, nil},
	"YT": &GeoPosition{-12.8333, 45.1667, nil},
	"MX": &GeoPosition{23, -102, nil},
	"FM": &GeoPosition{6.9167, 158.25, nil},
	"MD": &GeoPosition{47, 29, nil},
	"MC": &GeoPosition{43.7333, 7.4, nil},
	"MN": &GeoPosition{46, 105, nil},
	"ME": &GeoPosition{42, 19, nil},
	"MS": &GeoPosition{16.75, -62.2, nil},
	"MA": &GeoPosition{32, -5, nil},
	"MZ": &GeoPosition{-18.25, 35, nil},
	"MM": &GeoPosition{22, 98, nil},
	"NA": &GeoPosition{-22, 17, nil},
	"NR": &GeoPosition{-0.5333, 166.9167, nil},
	"NP": &GeoPosition{28, 84, nil},
	"NL": &GeoPosition{52.5, 5.75, nil},
	"AN": &GeoPosition{12.25, -68.75, nil},
	"NC": &GeoPosition{-21.5, 165.5, nil},
	"NZ": &GeoPosition{-41, 174, nil},
	"NI": &GeoPosition{13, -85, nil},
	"NE": &GeoPosition{16, 8, nil},
	"NG": &GeoPosition{10, 8, nil},
	"NU": &GeoPosition{-19.0333, -169.8667, nil},
	"NF": &GeoPosition{-29.0333, 167.95, nil},
	"MP": &GeoPosition{15.2, 145.75, nil},
	"NO": &GeoPosition{62, 10, nil},
	"OM": &GeoPosition{21, 57, nil},
	"PK": &GeoPosition{30, 70, nil},
	"PW": &GeoPosition{7.5, 134.5, nil},
	"PS": &GeoPosition{32, 35.25, nil},
	"PA": &GeoPosition{9, -80, nil},
	"PG": &GeoPosition{-6, 147, nil},
	"PY": &GeoPosition{-23, -58, nil},
	"PE": &GeoPosition{-10, -76, nil},
	"PH": &GeoPosition{13, 122, nil},
	"PN": &GeoPosition{-24.7, -127.4, nil},
	"PL": &GeoPosition{52, 20, nil},
	"PT": &GeoPosition{39.5, -8, nil},
	"PR": &GeoPosition{18.25, -66.5, nil},
	"QA": &GeoPosition{25.5, 51.25, nil},
	"RE": &GeoPosition{-21.1, 55.6, nil},
	"RO": &GeoPosition{46, 25, nil},
	"RU": &GeoPosition{60, 100, nil},
	"RW": &GeoPosition{-2, 30, nil},
	"SH": &GeoPosition{-15.9333, -5.7, nil},
	"KN": &GeoPosition{17.3333, -62.75, nil},
	"LC": &GeoPosition{13.8833, -61.1333, nil},
	"PM": &GeoPosition{46.8333, -56.3333, nil},
	"VC": &GeoPosition{13.25, -61.2, nil},
	"WS": &GeoPosition{-13.5833, -172.3333, nil},
	"SM": &GeoPosition{43.7667, 12.4167, nil},
	"ST": &GeoPosition{1, 7, nil},
	"SA": &GeoPosition{25, 45, nil},
	"SN": &GeoPosition{14, -14, nil},
	"RS": &GeoPosition{44, 21, nil},
	"SC": &GeoPosition{-4.5833, 55.6667, nil},
	"SL": &GeoPosition{8.5, -11.5, nil},
	"SG": &GeoPosition{1.3667, 103.8, nil},
	"SK": &GeoPosition{48.6667, 19.5, nil},
	"SI": &GeoPosition{46, 15, nil},
	"SB": &GeoPosition{-8, 159, nil},
	"SO": &GeoPosition{10, 49, nil},
	"ZA": &GeoPosition{-29, 24, nil},
	"GS": &GeoPosition{-54.5, -37, nil},
	"ES": &GeoPosition{40, -4, nil},
	"LK": &GeoPosition{7, 81, nil},
	"SD": &GeoPosition{15, 30, nil},
	"SR": &GeoPosition{4, -56, nil},
	"SJ": &GeoPosition{78, 20, nil},
	"SZ": &GeoPosition{-26.5, 31.5, nil},
	"SE": &GeoPosition{62, 15, nil},
	"CH": &GeoPosition{47, 8, nil},
	"SY": &GeoPosition{35, 38, nil},
	"TW": &GeoPosition{23.5, 121, nil},
	"TJ": &GeoPosition{39, 71, nil},
	"TZ": &GeoPosition{-6, 35, nil},
	"TH": &GeoPosition{15, 100, nil},
	"TL": &GeoPosition{-8.55, 125.5167, nil},
	"TG": &GeoPosition{8, 1.1667, nil},
	"TK": &GeoPosition{-9, -172, nil},
	"TO": &GeoPosition{-20, -175, nil},
	"TT": &GeoPosition{11, -61, nil},
	"TN": &GeoPosition{34, 9, nil},
	"TR": &GeoPosition{39, 35, nil},
	"TM": &GeoPosition{40, 60, nil},
	"TC": &GeoPosition{21.75, -71.5833, nil},
	"TV": &GeoPosition{-8, 178, nil},
	"UG": &GeoPosition{1, 32, nil},
	"UA": &GeoPosition{49, 32, nil},
	"AE": &GeoPosition{24, 54, nil},
	"GB": &GeoPosition{54, -2, nil},
	"US": &GeoPosition{38, -97, nil},
	"UM": &GeoPosition{19.2833, 166.6, nil},
	"UY": &GeoPosition{-33, -56, nil},
	"UZ": &GeoPosition{41, 64, nil},
	"VU": &GeoPosition{-16, 167, nil},
	"VE": &GeoPosition{8, -66, nil},
	"VN": &GeoPosition{16, 106, nil},
	"VG": &GeoPosition{18.5, -64.5, nil},
	"VI": &GeoPosition{18.3333, -64.8333, nil},
	"WF": &GeoPosition{-13.3, -176.2, nil},
	"EH": &GeoPosition{24.5, -13, nil},
	"YE": &GeoPosition{15, 48, nil},
	"ZM": &GeoPosition{-15, 30, nil},
	"ZW": &GeoPosition{-20, 30, nil},
	"AF": &GeoPosition{33, 65, nil},
	"ZZ": nil,
	"EU": &GeoPosition{54.5260, 15.2551, nil},
	"SS": &GeoPosition{7.8627, 29.6949, nil},
	"CW": &GeoPosition{12.1696, 68.9900, nil},
	"MF": &GeoPosition{18.0826, 63.0523, nil},
	"SX": &GeoPosition{18.0425, 63.0548, nil},
	"BQ": &GeoPosition{12.1784, 68.2385, nil},
	"AP": &GeoPosition{34.0479, 100.6197, nil},
	"AX": &GeoPosition{60.1785, 19.9156, nil},
	"BL": &GeoPosition{17.9000, 62.8333, nil},
}
