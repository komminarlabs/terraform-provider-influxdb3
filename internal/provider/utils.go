package provider

import (
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/thulasirajkomminar/influxdb3-management-go/cloud"
)

// formatErrorResponse formats the error response from the InfluxDB API.
// It looks for the generated JSON<status> error field on the response and
// falls back to the raw response body for undocumented status codes.
func formatErrorResponse(rsp any, statusCode int) string {
	v := reflect.ValueOf(rsp)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if field := v.FieldByName("JSON" + strconv.Itoa(statusCode)); field.IsValid() {
		if errorDetail, ok := field.Interface().(*influxdb3cloud.Error); ok && errorDetail != nil {
			return fmt.Sprintf("HTTP Status Code: %d\nError Code: %d\nError Message: %s\n", statusCode, errorDetail.Code, errorDetail.Message)
		}
	}

	if field := v.FieldByName("Body"); field.IsValid() {
		if body, ok := field.Interface().([]byte); ok && len(body) > 0 {
			return fmt.Sprintf("HTTP Status Code: %d\nResponse Body: %s", statusCode, body)
		}
	}

	return fmt.Sprintf("HTTP Status Code: %d", statusCode)
}

// formatEmptyResponse describes a response with a success status code but no
// parseable JSON body, which can happen when a proxy or load balancer
// intercepts the request. Used to fail cleanly instead of dereferencing a nil
// JSON200 field.
func formatEmptyResponse(rsp any, statusCode int) string {
	return "The InfluxDB API returned a success status code without a valid JSON body.\n" + formatErrorResponse(rsp, statusCode)
}

// newProviderData extracts the providerData set by the provider Configure
// method. It returns false when the provider is not yet configured (nil data)
// or, with an error diagnostic, when the data has an unexpected type.
func newProviderData(raw any, diags *diag.Diagnostics) (providerData, bool) {
	if raw == nil {
		return providerData{}, false
	}

	pd, ok := raw.(providerData)
	if !ok {
		diags.AddError(
			"Unexpected Provider Data Type",
			fmt.Sprintf("Expected providerData, got: %T. Please report this issue to the provider developers.", raw),
		)
		return providerData{}, false
	}
	return pd, true
}
