package datafeed

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	metartafparser "github.com/ryansavara/go-metar-taf-parser"
	"github.com/vzauartcc/data-parser/config"
)

var metarURL = "https://metar.vatsim.net/"

func FetchMetarFeed(ctx context.Context, client *http.Client, retryInterval time.Duration) ([]metartafparser.Metar, error) {
	cty, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var builder strings.Builder
	for k := range config.Airports {
		if builder.Len() > 0 {
			builder.WriteString(",")
		}

		builder.WriteString(k)
	}

	// Not json data, so we can't use doRequest.

	req, err := http.NewRequestWithContext(cty, http.MethodGet,
		metarURL+builder.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(cty.Err(), context.DeadlineExceeded) {
			log.Printf("Metar fetch timed out")

			if retryInterval != 0 {
				time.Sleep(retryInterval)

				return FetchMetarFeed(ctx, client, 0*time.Second)
			}

			return nil, nil
		}

		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode != http.StatusInternalServerError && retryInterval != 0 {
			return FetchMetarFeed(ctx, client, 0*time.Second)
		}

		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, resp.Status)
	}

	var metars []string

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		metars = append(metars, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	var retval []metartafparser.Metar

	for i := range metars {
		given := metars[i]

		metar, err := metartafparser.ParseMetar(given, nil)
		if err == nil {
			retval = append(retval, *metar)
		}
	}

	return retval, nil
}
