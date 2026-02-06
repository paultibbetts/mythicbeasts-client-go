package pi

import (
	"errors"
	"fmt"
)

// ErrEmptyIdentifier is returned when an identifier is not used.
// Identifiers are required for all Pi resources.
var ErrEmptyIdentifier = errors.New("identifier is required")

// ErrIdentifierConflict indicates the requested resource identifier
// has already been used.
type ErrIdentifierConflict struct {
	Identifier string
}

func (e *ErrIdentifierConflict) Error() string {
	return fmt.Sprintf("identifier %q already in use", e.Identifier)
}
