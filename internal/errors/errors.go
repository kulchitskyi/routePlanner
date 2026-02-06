package errors

import (
	"errors"
)

var (
	// 400 Errors
	ErrInvalidInputData = errors.New("Invalid input data. Please check your request payload.")
	ErrInvalidPlaceData = errors.New("Invalid place data.")
	ErrInvalidUserData  = errors.New("Invalid user data.")
	ErrXSSDetected      = errors.New("Input contains potentially unsafe content.")
	// 401 Errors
	ErrUnauthorized       = errors.New("Authentication required.")
	ErrInvalidCredentials = errors.New("Invalid email or password.")
	ErrInvalidToken       = errors.New("Invalid or expired token.")
	// 404 Errors
	ErrUserNotFound  = errors.New("User not found.")
	ErrPlaceNotFound = errors.New("Place not found.")
	// 409 Errors
	ErrConflict = errors.New("Username or email already exists.")

	// Route Errors
	ErrNoPlacesFound    = errors.New("no places found for the selected categories in this radius")
	ErrExtractionFailed = errors.New("failed to identify your preferences. try to use more specific words")
	// 500 Errors
	ErrInternalServer    = errors.New("An unexpected server error occurred.")
	ErrJSONMarshalFailed = errors.New("Failed to process internal data.")
)
