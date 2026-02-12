package channel

import "errors"

var (
	ErrInvalidChannelFormat = errors.New("invalid channel format")
	ErrUnknownChannelType   = errors.New("unknown channel type")
	ErrInvalidCFXUserID     = errors.New("invalid cfx_user_id format")
)
