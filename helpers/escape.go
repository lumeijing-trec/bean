// MIT License

// Copyright (c) The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package helpers

import (
	"bytes"
	"encoding/json"
	"errors"
	"html"
	"io"
	"net/http"
	"strings"
)

func PostDataStripTags(req *http.Request, trimSpace bool) (map[string]interface{}, error) {

	// Hold the POST json parameters as an interface
	var data interface{}

	// Hold the POST data as a map with string index
	var postdatamap map[string]interface{}

	// Get Content-Type parameter from request header
	contentType := req.Header.Get("Content-Type")

	if strings.Contains(strings.ToLower(contentType), "application/json") {

		// XXX: IMPORTANT - c.Request().Body is a buffer, which means that once it has been read, it cannot be read again.
		if req.Body != nil {

			var err error

			bodyBytes := bytes.NewBuffer(make([]byte, 0))

			reader := io.TeeReader(req.Body, bodyBytes)
			if err = json.NewDecoder(reader).Decode(&data); err != nil {

				var syntaxError *json.SyntaxError
				var unmarshalTypeError *json.UnmarshalTypeError

				// XXX: IMPORTANT - JSON Syntax error handling
				switch {

				// Request body contains badly-formed JSON (at position %d), syntaxError.Offset
				case errors.As(err, &syntaxError):
					return nil, err

				// Request body contains badly-formed JSON
				case errors.Is(err, io.ErrUnexpectedEOF):
					return nil, err

				// Request body contains an invalid value for the %q field (at position %d)", unmarshalTypeError.Field, unmarshalTypeError.Offset
				case errors.As(err, &unmarshalTypeError):
					return nil, err

				// Request body is empty
				case errors.Is(err, io.EOF):
					return nil, err

				default:
					return nil, err
				}
			}

			// Restore the io.ReadCloser to its original state so that we can read c.Request().Body somewhere else
			req.Body = io.NopCloser(bodyBytes)

		} else {

			return nil, errors.New("ERROR: empty request body")
		}

		data = InterfaceStripTags(data, trimSpace)

		// XXX: IMPORTANT - Here we will check again that we able to decode the JSON and load the data into a map[string]interface.
		switch v := data.(type) {

		case map[string]interface{}:
			postdatamap = v
		default:
			return nil, errors.New("ERROR: JSON syntax error")
		}
	}

	return postdatamap, nil
}

func InterfaceStripTags(data interface{}, trimSpace bool) interface{} {

	if values, ok := data.([]interface{}); ok {

		for i := range values {
			data.([]interface{})[i] = InterfaceStripTags(values[i], trimSpace)
		}

	} else if values, ok := data.(map[string]interface{}); ok {

		for k, v := range values {
			data.(map[string]interface{})[k] = InterfaceStripTags(v, trimSpace)
		}

	} else if value, ok := data.(string); ok {

		if trimSpace {
			value = strings.TrimSpace(value)
		}

		data = html.EscapeString(value)
	}

	return data
}

// Structure is a data type, so you must pass structure address (&) to the following function as the first parameter.
// Example:
//  	test := struct {
// 		Firstname	string
// 		Lastname	string
// 		Age			int
// 	}{
// 		Firstname: "Taro",
// 		Lastname: "<script>alert()</script>Yamada",
// 		Age: 40,
// 	}

// helpers.StructStripTags(&test, true)
func StructStripTags(data interface{}, trimSpace bool) error {

	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var mapSI map[string]interface{}
	if err := json.Unmarshal(bytes, &mapSI); err != nil {
		return err
	}

	mapSI = InterfaceStripTags(mapSI, trimSpace).(map[string]interface{})
	bytes2, err := json.Marshal(mapSI)

	if err != nil {
		return err
	}

	if err := json.Unmarshal(bytes2, data); err != nil {
		return err
	}

	return nil
}
