package datafeed

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vzauartcc/data-parser/config"
)

var metarURL = "https://metar.vatsim.net/"

func FetchMetarFeed(ctx context.Context) ([]string, error) {
	cty, cancel := context.WithTimeout(ctx, 5*time.Second)
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
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrInvalidStatus, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(body), "\n"), nil
}
