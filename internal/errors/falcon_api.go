package internalErrors

import (
	"errors"
)

var (
	ErrNilFalconAPIConfiguration = errors.New("missing falcon_api in CRD spec - falcon_api cannot be nil")
)
