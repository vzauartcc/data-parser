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

	"github.com/vzauartcc/data-parser/config"
)

var metarURL = "https://metar.vatsim.net/"
var retry = true

func FetchMetarFeed(ctx context.Context) ([]string, error) {
	cty, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	var builder strings.Builder
	for k := range config.Airports {
		if builder.Len() > 0 {
			builder.WriteString(",")
		}

		builder.WriteString(k)
	}

	req, err := http.NewRequestWithContext(cty, http.MethodGet,
		metarURL+builder.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(cty.Err(), context.DeadlineExceeded) {
			log.Printf("Metar fetch timed out")

			if retry {
				retry = false

				time.Sleep(30 * time.Second)

				return FetchMetarFeed(ctx)
			}

			return nil, nil
		}

		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode != http.StatusInternalServerError && retry {
			retry = false

			time.Sleep(30 * time.Second)

			return FetchMetarFeed(ctx)
		}

		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, resp.Status)
	}

	var retval []string

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		retval = append(retval, scanner.Text())
	}

	err = scanner.Err()
	if err != nil {
		return nil, err
	}

	return retval, nil
}
