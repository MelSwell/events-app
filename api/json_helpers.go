package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-playground/validator"
)

type successJSON struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

type errorJSON struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func marshalAndSend(w http.ResponseWriter, jsonRes interface{}, statusCode int) error {
	switch jsonRes.(type) {
	case successJSON, errorJSON:
		payload, err := json.Marshal(jsonRes)
		if err != nil {
			return err
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)

		// write the json out
		_, err = w.Write(payload)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported type: %T", jsonRes)
	}
	return nil
}

func (app *application) SendSuccessJSON(w http.ResponseWriter, statusCode int, data interface{}, wrap ...string) error {
	jsonRes := successJSON{
		Status: "success",
	}

	if len(wrap) > 0 {
		jsonRes.Data = map[string]interface{}{wrap[0]: data}
	} else {
		jsonRes.Data = data
	}

	return marshalAndSend(w, jsonRes, statusCode)
}

func (app *application) SendErrorJSON(w http.ResponseWriter, statusCode int, err error) error {
	jsonRes := errorJSON{}
	if statusCode >= 500 {
		jsonRes.Status = "error"
	} else {
		jsonRes.Status = "fail"
	}

	jsonRes.Message = err.Error()

	return marshalAndSend(w, jsonRes, statusCode)
}

var validate = validator.New()

func (app *application) ReadJSON(w http.ResponseWriter, r *http.Request, data interface{}, validationReq bool) error {
	maxBytes := 1024 * 1024 // one megabyte
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	// attempt to decode the data
	err := dec.Decode(data)
	if err != nil {
		return err
	}

	// make sure only one JSON value in payload
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	if validationReq {
		err := validate.Struct(data)
		if err != nil {
			return err
		}
	}

	return nil
}
