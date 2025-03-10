package provider

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/komminarlabs/influxdb3"
)

// formatErrorResponse formats the error response from the InfluxDB API.
func formatErrorResponse(rsp any, statusCode int) (string, error) {
	v := reflect.ValueOf(rsp)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	fieldName := "JSON" + strconv.Itoa(statusCode)
	field := v.FieldByName(fieldName)
	if !field.IsValid() {
		return "", fmt.Errorf("field %s not found", fieldName)
	}

	errorDetail, ok := field.Interface().(*influxdb3.Error)
	if !ok {
		return "", fmt.Errorf("field %s is not of type *influxdb3.Error %s", fieldName, field)
	}

	if errorDetail == nil {
		return fmt.Sprintf("HTTP Status Code: %d", statusCode), nil
	}
	return fmt.Sprintf("HTTP Status Code: %d\nError Code: %d\nError Message: %s\n", statusCode, errorDetail.Code, errorDetail.Message), nil
}
