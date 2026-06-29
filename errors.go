package rtx5sdk

import (
	"errors"
	"fmt"
)

var (
	ErrNotConnected    = errors.New("manager session is not connected; call Connect first")
	ErrMissingToken    = errors.New("auth response did not contain a token")
	ErrInvalidResponse = errors.New("unexpected non-JSON response body")
)

type MissingConfigError struct {
	Name string
}

func (e MissingConfigError) Error() string {
	return "missing required config: " + e.Name
}

type InvalidInputError struct {
	Message string
}

func (e InvalidInputError) Error() string {
	return "invalid input: " + e.Message
}

type InsecureBaseURLError struct {
	URL string
}

func (e InsecureBaseURLError) Error() string {
	return "insecure base url: " + e.URL + " is plaintext http; pass a https url or call AllowInsecureHTTP(true) to opt in (only safe on loopback/VPN)"
}

type APIError struct {
	StatusCode int
	Body       string
}

func (e APIError) Error() string {
	return fmt.Sprintf("api request failed with status %d: %s", e.StatusCode, e.Body)
}

func (e APIError) APIBody() string {
	return e.Body
}

type ResponseTooLargeError struct {
	MaxBytes int64
}

func (e ResponseTooLargeError) Error() string {
	return fmt.Sprintf("api response exceeded maximum size of %d bytes", e.MaxBytes)
}
