package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// APIResponse standardizes success responses
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// APIError standardizes error responses with both error and message fields
type APIError struct {
	Success bool        `json:"success"`
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// SendSuccess sends a standard 200/201 JSON success response
func SendSuccess(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

// SendError sends a standard JSON error response with human-readable error and message
func SendError(c *gin.Context, statusCode int, errMsg string) {
	c.AbortWithStatusJSON(statusCode, APIError{
		Success: false,
		Error:   errMsg,
		Message: errMsg,
	})
}

// SendValidationError sends a 422 Unprocessable Entity error with details
func SendValidationError(c *gin.Context, errMsg string, details interface{}) {
	c.AbortWithStatusJSON(http.StatusUnprocessableEntity, APIError{
		Success: false,
		Error:   errMsg,
		Message: errMsg,
		Details: details,
	})
}

var humanReadableFieldNames = map[string]string{
	"email":           "Email address",
	"password":        "Password",
	"currentpassword": "Current password",
	"newpassword":     "New password",
	"name":            "Name",
	"phone":           "Phone number",
	"roomnumber":      "Room number",
	"floor":           "Floor number",
	"categoryid":      "Room category",
	"rate":            "Room rate",
	"defaultrate":     "Default rate",
	"baserate":        "Base rate",
	"maxoccupancy":    "Maximum occupancy",
	"roomid":          "Room",
	"reservationid":   "Reservation",
	"guestname":       "Guest name",
	"guestemail":      "Guest email address",
	"guestphone":      "Guest phone number",
	"checkin":         "Check-in date",
	"checkout":        "Check-out date",
	"numguests":       "Number of guests",
	"guests":          "Number of guests",
	"amount":          "Amount",
	"billid":          "Bill folio",
	"description":     "Description",
	"category":        "Category",
	"hotelid":         "Hotel property",
	"address":         "Address",
	"city":            "City",
	"state":           "State",
	"country":         "Country",
	"status":          "Status",
	"idtype":          "ID type",
	"idnumber":        "ID number",
}

// FormatBindingError converts JSON binding and validation errors into friendly human-readable strings
func FormatBindingError(err error) string {
	if err == nil {
		return "Invalid request"
	}

	if errors.Is(err, io.EOF) {
		return "Request body cannot be empty"
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		errMsgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			rawField := fe.Field()
			lookup := strings.ToLower(rawField)
			fieldName, exists := humanReadableFieldNames[lookup]
			if !exists {
				fieldName = rawField
			}

			switch fe.Tag() {
			case "required":
				errMsgs = append(errMsgs, fmt.Sprintf("%s is required", fieldName))
			case "email":
				errMsgs = append(errMsgs, "Please provide a valid email address")
			case "min":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be at least %s characters long", fieldName, fe.Param()))
			case "max":
				errMsgs = append(errMsgs, fmt.Sprintf("%s cannot exceed %s characters", fieldName, fe.Param()))
			case "gt":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be greater than %s", fieldName, fe.Param()))
			case "gte":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be at least %s", fieldName, fe.Param()))
			case "lt":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be less than %s", fieldName, fe.Param()))
			case "lte":
				errMsgs = append(errMsgs, fmt.Sprintf("%s cannot exceed %s", fieldName, fe.Param()))
			default:
				errMsgs = append(errMsgs, fmt.Sprintf("%s is invalid", fieldName))
			}
		}
		if len(errMsgs) > 0 {
			return strings.Join(errMsgs, "; ")
		}
	}

	errStr := err.Error()
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) || strings.Contains(errStr, "invalid character") || strings.Contains(errStr, "unexpected end of JSON") || strings.Contains(errStr, "looking for beginning of value") {
		return "Request body contains invalid or malformed JSON"
	}

	if strings.Contains(errStr, "cannot unmarshal") || strings.Contains(errStr, "parsing time") {
		return "Invalid data format provided. Please verify the submitted values."
	}

	return "Invalid request payload. Please check all submitted fields."
}

// SendBindingError sends a human-readable 400 Bad Request error for JSON parsing or binding failures
func SendBindingError(c *gin.Context, err error) {
	msg := FormatBindingError(err)
	SendError(c, http.StatusBadRequest, msg)
}
