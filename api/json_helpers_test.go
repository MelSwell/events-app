package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"events-app/data/models"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadJSON(t *testing.T) {
	app := &application{}
	tests := []struct {
		name          string
		body          string
		expectedError string
		validationReq bool
	}{
		{
			name:          "Valid JSON",
			body:          `{"email":"example@hello.com", "password":"password"}`,
			expectedError: "",
			validationReq: true,
		},
		{
			name:          "Invalid JSON",
			body:          `{"email":}`,
			expectedError: "invalid character '}' looking for beginning of value",
			validationReq: false,
		},
		{
			name:          "More than one JSON object",
			body:          `{"email":"example@hello.com"},{"whoops":"more data"}`,
			expectedError: "body must only contain a single JSON value",
			validationReq: false,
		},
		{
			name:          "Unknown Field",
			body:          `{"unknown":"field"}`,
			expectedError: "json: unknown field \"unknown\"",
			validationReq: false,
		},
		{
			name:          "Missing Required Field",
			body:          `{"email":"", "password":"password"}`,
			expectedError: "Key: 'User.Email' Error:Field validation for 'Email' failed on the 'required' tag",
			validationReq: true,
		},
		{
			name:          "Invalid Field",
			body:          `{"email":"example@hello", "password":"password"}`,
			expectedError: "Key: 'User.Email' Error:Field validation for 'Email' failed on the 'email' tag",
			validationReq: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", bytes.NewBufferString(tt.body))
			w := httptest.NewRecorder()

			var data models.User
			err := app.ReadJSON(w, req, &data, tt.validationReq)
			if tt.expectedError == "" {
				assert.NoError(t, err)
				assert.Equal(t, "example@hello.com", data.Email)
			} else {
				assert.EqualError(t, err, tt.expectedError)
			}
		})
	}
}

func TestMarshalAndSend_UnsupportedType(t *testing.T) {
	err := marshalAndSend(httptest.NewRecorder(), struct{ Name string }{Name: "test"}, http.StatusOK)
	assert.EqualError(t, err, "unsupported type: struct { Name string }")
}

func TestSendSuccessJSON(t *testing.T) {
	app := &application{}
	tests := []struct {
		name         string
		data         interface{}
		wrap         []string
		expectedData interface{}
	}{
		{
			name:         "Normal Data",
			data:         map[string]string{"key": "value"},
			wrap:         nil,
			expectedData: map[string]interface{}{"key": "value"},
		},
		{
			name:         "Wrapped Data",
			data:         map[string]string{"key": "value"},
			wrap:         []string{"wrapped"},
			expectedData: map[string]interface{}{"wrapped": map[string]interface{}{"key": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := app.SendSuccessJSON(w, http.StatusOK, tt.data, tt.wrap...)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response successJSON
			err = json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Equal(t, "success", response.Status)
			assert.Equal(t, tt.expectedData, response.Data)
		})
	}
}

func TestSendErrorJSON(t *testing.T) {
	app := &application{}
	tests := []struct {
		name           string
		statusCode     int
		er             error
		expectedStatus string
	}{
		{
			name:           "Client Error",
			statusCode:     http.StatusBadRequest,
			er:             errors.New("An error occurred"),
			expectedStatus: "fail",
		},
		{
			name:           "Server Error",
			statusCode:     http.StatusInternalServerError,
			er:             errors.New("Internal server error"),
			expectedStatus: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := app.SendErrorJSON(w, tt.statusCode, tt.er)
			assert.NoError(t, err)
			assert.Equal(t, tt.statusCode, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var response errorJSON
			err = json.NewDecoder(w.Body).Decode(&response)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, response.Status)
			assert.Equal(t, tt.er.Error(), response.Message)
		})
	}
}
