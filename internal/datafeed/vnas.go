package datafeed

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type VnasUser struct {
	VnasController

	CID int
}

type VnasControllerFeed struct {
	UpdatedAt   time.Time
	Controllers []VnasUser
}

type ControllerFeed struct {
	UpdatedAt   time.Time        `json:"updatedAt"`
	Controllers []VnasController `json:"controllers"`
}

type VnasController struct {
	ArtccID string `json:"artccId"`
	// PrimaryFacilityID string         `json:"primaryFacilityId"`
	// PrimaryPositionID string         `json:"primaryPositionId"`
	Role string `json:"role"`
	// Positions         []VnasPosition `json:"positions"`
	IsActive   bool           `json:"isActive"`
	IsObserver bool           `json:"isObserver"`
	LoginTime  time.Time      `json:"loginTime"`
	VatsimData VnasVatsimData `json:"vatsimData"`
}

type VnasPosition struct {
	Facility        string `json:"facility"`
	FacilityName    string `json:"facilityName"`
	PositionName    string `json:"positionName"`
	PositionType    string `json:"positionType"`
	RadioName       string `json:"radioName"`
	DefaultCallsign string `json:"defaultCallsign"`
	Frequency       int64  `json:"frequency"`
	IsPrimary       bool   `json:"isPrimary"`
	IsActive        bool   `json:"isActive"`
	EramData        struct {
		SectorID string `json:"sectorId"`
	} `json:"eramData"`
	StarsData struct {
		Subset   int    `json:"subset"`
		SectorID string `json:"sectorId"`
		AreaID   string `json:"areaId"`
	} `json:"starsData"`
}

type VnasVatsimData struct {
	CID            string `json:"cid"`
	RealName       string `json:"realName"`
	ControllerInfo string `json:"controllerInfo"`
	// UserRating       string `json:"userRating"`
	RequestedRating string `json:"requestedRating"`
	Callsign        string `json:"callsign"`
	// FacilityType     string `json:"facilityType"`
	PrimaryFrequency int64 `json:"primaryFrequency"`
}

var vNasURL = fmt.Sprintf(
	"https://live.env.vnas.vatsim.net/data-feed/controllers.json?t=%d",
	time.Now().UnixMilli(),
)

func FetchVnasFeed(ctx context.Context) (VnasControllerFeed, error) {
	cty, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(cty, http.MethodGet,
		vNasURL, nil)
	if err != nil {
		return VnasControllerFeed{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return VnasControllerFeed{}, err
	}
	defer resp.Body.Close()

	var feed ControllerFeed

	err = json.NewDecoder(resp.Body).Decode(&feed)
	if err != nil {
		return VnasControllerFeed{}, err
	}

	retval := VnasControllerFeed{
		UpdatedAt:   feed.UpdatedAt,
		Controllers: make([]VnasUser, 0),
	}

	for _, controller := range feed.Controllers {
		cid, err := strconv.Atoi(controller.VatsimData.CID)
		if err != nil {
			continue
		}

		retval.Controllers = append(retval.Controllers, VnasUser{
			CID:            cid,
			VnasController: controller,
		})
	}

	return retval, nil
}

func GetRating(rating string) int {
	switch rating {
	case "Observer":
		return 1
	case "Student1":
		return 2
	case "Student2":
		return 3
	case "Student3":
		return 4
	case "Controller1":
		return 5
	case "Controller2":
		return 6
	case "Controller3":
		return 7
	case "Instructor1":
		return 8
	case "Instructor2":
		return 9
	case "Instructor3":
		return 10
	case "Supervisor":
		return 11
	case "Administrator":
		return 12
	default:
		return 0
	}
}
