package bars

import "errors"

// General errors
var (
	ErrNoAuth          = errors.New("authorization in BARS has not been completed")
	ErrWrongGradesPage = errors.New("user's grades page is not the main one")
)
