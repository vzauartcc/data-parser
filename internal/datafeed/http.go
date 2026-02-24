package datafeed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var ErrInvalidStatus = errors.New("invalid status code returned")

func doRequest[T any](ctx context.Context, client *http.Client, uri string) (T, error) {
	var result T

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return result, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return result, err
	}

	defer func() {
		// Ensure body is read completely before closing
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return result, fmt.Errorf("%w: %s", ErrInvalidStatus, resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}
