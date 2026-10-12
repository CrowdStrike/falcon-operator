package internalErrors

import (
	"errors"
)

var (
	ErrNilFalconAPIConfiguration  = errors.New("missing falcon_api in CRD spec - falcon_api cannot be nil")
	ErrMissingCIDWithoutFalconAPI = errors.New("missing Falcon CID - cid must be set when Falcon API credentials are not configured")
)
