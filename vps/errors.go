package vps

import (
	"errors"
	"fmt"
)

// ErrEmptyIdentifier is returned when an identifier is not used.
// Identifiers are required for all VPS resources.
var ErrEmptyIdentifier = errors.New("identifier is required")

// ErrIdentifierConflict indicates the requested resource identifier
// has already been used.
type ErrIdentifierConflict struct {
	Identifier string
}

func (e *ErrIdentifierConflict) Error() string {
	return fmt.Sprintf("identifier %q already in use", e.Identifier)
}

// ErrUserDataNotFound indicates the requested user data name
// could not be found.
type ErrUserDataNotFound struct {
	Name string
}

func (e *ErrUserDataNotFound) Error() string {
	return fmt.Sprintf("could not find user data with the name %q", e.Name)
}

// ErrInvalidProductPeriod indicates the product period used
// was invalid. See ProductPeriod or
// https://www.mythic-beasts.com/support/api/vps#sec-parameters14
// for valid periods.
type ErrInvalidProductPeriod struct {
	Period ProductPeriod
}

func (e *ErrInvalidProductPeriod) Error() string {
	return fmt.Sprintf("invalid product period: %q", e.Period)
}
