package datafeed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrInvalidStatus = errors.New("invalid status code returned")
)

type Pirep struct {
	// Date                 time.Time `json:"receiptTime"`
	ObservationTime int `json:"obsTime"`
	// QCField              int       `json:"qcField"`
	// IcaoID               string    `json:"icaoId"`
	AircraftType string  `json:"acType"`
	Latitude     float64 `json:"lat"`
	Longitude    float64 `json:"lon"`
	FlightLevel  int     `json:"fltLvl"`
	// FlightLevelType      string   `json:"fltLvelType"`
	Clouds     *[]Cloud `json:"clouds"`
	Visibility *int     `json:"visib"`
	// Weather        string   `json:"wxString"`
	Temperature   *int `json:"temp"`
	WindDirection *int `json:"wdir"`
	WindSpeed     *int `json:"wspd"`
	// IcingBase1     *int     `json:"icgBas1"`
	// IcingTops1     *int     `json:"icgTop1"`
	IcingInterval1 string `json:"icgInt1"`
	IcingType1     string `json:"icgType1"`
	// IcingBase2           *int     `json:"icgBas2"`
	// IcingTops2           *int     `json:"icgTop2"`
	// IcingInterval2       string   `json:"icgInt2"`
	// IcingType2           string   `json:"icgType2"`
	// TurbulenceBase1      *int   `json:"tbBas1"`
	// TurbulenceTops1      *int   `json:"tbTop1"`
	TurbulenceInterval1  string `json:"tbInt1"`
	TurbulenceType1      string `json:"tbType1"`
	TurbulenceFrequency1 string `json:"tbFreq1"`
	// TurbulenceBase2      *int   `json:"tbBas2"`
	// TurbulenceTops2      *int   `json:"tbTop2"`
	// TurbulenceInterval2  string `json:"tbInt2"`
	// TurbulenceType2      string `json:"tbType2"`
	// TurbulenceFrequency2 string `json:"tbFreq2"`
	// VerticalGust         *int     `json:"vertGust"`
	// BrakingAction  string `json:"brkAction"`
	PirepType      string `json:"pirepType"`
	RawObservation string `json:"rawOb"`
}

type Cloud struct {
	Cover string `json:"cover"`
	Base  int    `json:"base"`
	Tops  int    `json:"top"`
}

var pirepURL = fmt.Sprintf(
	"https://aviationweather.gov/api/data/pirep?id=KRFD&distance=200&age=2&format=json&t=%d",
	time.Now().UnixMilli(),
)

func FetchPirepFeed(ctx context.Context) ([]Pirep, error) {
	cty, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(cty, http.MethodGet,
		pirepURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, resp.Status)
	}

	retval := make([]Pirep, 0)

	err = json.NewDecoder(resp.Body).Decode(&retval)
	if err != nil {
		return nil, err
	}

	return retval, nil
}
