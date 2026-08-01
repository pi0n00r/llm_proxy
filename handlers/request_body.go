package handlers

import (
	"errors"
	"io"
	"net/http"

	"llm_proxy/config"
)

func readRequestBody(w http.ResponseWriter, r *http.Request, configuredLimit int64) ([]byte, int, error) {
	limit := configuredLimit
	if limit <= 0 {
		limit = config.DefaultMaxRequestBodyBytes
	}

	r.Body = http.MaxBytesReader(w, r.Body, limit)
	body, err := io.ReadAll(r.Body)
	if err == nil {
		return body, 0, nil
	}

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return nil, http.StatusRequestEntityTooLarge, err
	}
	return nil, http.StatusBadRequest, err
}

func requestBodyErrorMessage(status int) string {
	if status == http.StatusRequestEntityTooLarge {
		return "Request body too large"
	}
	return "Failed to read request body"
}
