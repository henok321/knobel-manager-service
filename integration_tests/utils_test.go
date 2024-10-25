package integration_tests

import (
	"bytes"
	"database/sql"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name               string
	method             string
	endpoint           string
	requestBody        string
	requestHeaders     map[string]string
	setup              func(db *sql.DB)
	expectedStatusCode int
	expectedBody       string
	expectedHeaders    map[string]string
}

func newTestRequest(t *testing.T, tc testCase, server *httptest.Server) {
	var requestBody io.Reader
	if tc.requestBody != "" {
		requestBody = bytes.NewBuffer([]byte(tc.requestBody))
	}

	// Create the HTTP request
	req, err := http.NewRequest(tc.method, server.URL+tc.endpoint, requestBody)
	if err != nil {
		t.Fatalf("Failed to create %s request: %v", tc.method, err)
	}

	for key, value := range tc.requestHeaders {
		req.Header.Set(key, value)
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to perform %s request: %v", tc.method, err)
	}
	defer resp.Body.Close()

	responseBodyBytes, _ := io.ReadAll(resp.Body)
	responseBodyString := string(responseBodyBytes)

	assert.Equal(t, tc.expectedStatusCode, resp.StatusCode, "Expected status code %d", tc.expectedStatusCode)

	if tc.expectedBody != "" {
		assert.JSONEq(t, tc.expectedBody, responseBodyString)
	}
}

func cleanupSetup(t *testing.T, db *sql.DB, filepath string) {
	err := executeSQLFile(t, db, filepath)
	if err != nil {
		log.Fatalf("Failed to execute SQL file: %v", err)
	}

}
