package internalErrors

import (
	"errors"
)

var (
	ErrMissingFalconAPICredentialsInSecret = errors.New("missing Falcon API credentials in falcon secret")
	ErrMissingFalconCIDInSecret            = errors.New("missing Falcon CID in falcon secret")
)
