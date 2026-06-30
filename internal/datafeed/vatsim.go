package datafeed

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type VatsimFeed struct {
	// General         VatsimGeneralInfo      `json:"general"`
	Pilots []VatsimPilot `json:"pilots"`
	ATISs  []VatsimATIS  `json:"atis"`
	// Prefiles        []VatimsPrefile        `json:"prefiles"`
	// Ratings         []VatsimRating         `json:"rating"`
	// PilotRatings    []VatsimPilotRating    `json:"pilotRating"`
	// MilitaryRatings []VatsimMilitaryRating `json:"militaryRating"`
}

type VatsimGeneralInfo struct {
	Version          int       `json:"version"`
	Reload           int       `json:"reload"`
	Update           string    `json:"update"`
	UpdateTimestamp  time.Time `json:"update_timestamp"`
	ConnectedClients int       `json:"connected_clients"`
	UniqueUsers      int       `json:"unique_users"`
}

type VatsimPilot struct {
	CID      int    `json:"cid"`
	Name     string `json:"name"`
	Callsign string `json:"callsign"`
	// Server         string            `json:"server"`
	// PilotRating    int               `json:"pilot_rating"`
	// MilitaryRating int               `json:"military_rating"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	Altitude    int     `json:"altitude"`
	Groundspeed int     `json:"groundspeed"`
	Transponder string  `json:"transponder"`
	Heading     int     `json:"heading"`
	// QnhInHg        float64           `json:"qnh_i_hg"`
	// QnhInMb        float64           `json:"qnh_mb"`
	FlightPlan *VatsimFlightPlan `json:"flight_plan"`
	// LogonTime      time.Time         `json:"logon_time"`
	// LastUpdated    time.Time         `json:"last_updated"`
}

type VatsimFlightPlan struct {
	// FlightRules         string `json:"flight_rules"`
	// Aircraft            string `json:"aircraft"`
	AircraftFAA string `json:"aircraft_faa"`
	// AircraftShort       string `json:"aircraft_short"`
	Departure string `json:"departure"`
	Arrival   string `json:"arrival"`
	// Alternate           string `json:"alternate"`
	// CruiseTAS           string `json:"cruise_tas"`
	RequestedAltitude string `json:"altitude"`
	// DepartureTime       string `json:"deptime"`
	// EnrouteTime         string `json:"enroute_time"`
	// FuelTime            string `json:"fuel_time"`
	Remarks string `json:"remarks"`
	Route   string `json:"route"`
	// RevisionID          int    `json:"revision_id"`
	// AssignedTransponder string `json:"assigned_transponder"`
}

type VatsimATIS struct {
	// CID         int       `json:"cid"`
	// Name        string    `json:"name"`
	Callsign string `json:"callsign"`
	// Frequency   string    `json:"frequency"`
	// Facility    int       `json:"facility"`
	// Rating      int       `json:"rating"`
	// Server      string    `json:"server"`
	// VisualRange int       `json:"visual_range"`
	ATISCode string `json:"atis_code"`
	// TextATIS    []string  `json:"text_atis"`
	// LastUpdated time.Time `json:"last_updated"`
	// LogonTime   time.Time `json:"logon_time"`
}

type VatimsPrefile struct {
	CID         int              `json:"cid"`
	Name        string           `json:"name"`
	Callsign    string           `json:"callsign"`
	FlightPlan  VatsimFlightPlan `json:"flight_plan"`
	LastUpdated time.Time        `json:"last_updated"`
}

type VatsimRating struct {
	ID        int    `json:"id"`
	ShortName string `json:"short"`
	LongName  string `json:"long"`
}

type VatsimPilotRating struct {
	ID        int    `json:"id"`
	ShortName string `json:"short_name"`
	LongName  string `json:"long_name"`
}

type VatsimMilitaryRating struct {
	ID        int    `json:"id"`
	ShortName string `json:"short_name"`
	LongName  string `json:"long_name"`
}

var vatsimURL = fmt.Sprintf(
	"https://data.vatsim.net/v3/vatsim-data.json?t=%d",
	time.Now().UnixMilli(),
)

func FetchVatsimDatafeed(ctx context.Context, client *http.Client) (VatsimFeed, error) {
	cty, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return doRequest[VatsimFeed](cty, client, vatsimURL)
}
