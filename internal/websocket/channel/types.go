package channel

import (
	"regexp"
	"strings"
)

// Channel prefixes
const (
	PrefixUser = "user:"
)

// Valid user channel types
var ValidUserChannels = map[string]bool{
	"margin":   true,
	"position": true,
}

// Ajaib ID validation pattern
var ajaibIDPattern = regexp.MustCompile(`^[0-9]{1,10}$`)

// ChannelInfo contains parsed information about a channel
type ChannelInfo struct {
	Name       string
	Prefix     string
	UserID     string
	AjaibID    string
	ChannelSub string
}

func ParseChannel(channel string) (*ChannelInfo, error) {
	info := &ChannelInfo{
		Name: channel,
	}

	if !strings.HasPrefix(channel, PrefixUser) {
		return nil, ErrUnknownChannelType
	}

	info.Prefix = PrefixUser

	// Format: user:{user_id}:{channel_type}
	parts := strings.Split(strings.TrimPrefix(channel, PrefixUser), ":")
	if len(parts) != 2 {
		return nil, ErrInvalidChannelFormat
	}

	ajaibID := parts[0]
	channelSub := parts[1]

	if ajaibID == "" {
		return nil, ErrInvalidChannelFormat
	}

	if channelSub == "" {
		return nil, ErrInvalidChannelFormat
	}

	if !isValidAjaibID(ajaibID) {
		return nil, ErrInvalidCFXUserID
	}

	if !ValidUserChannels[channelSub] {
		return nil, ErrUnknownChannelType
	}

	info.AjaibID = ajaibID
	info.ChannelSub = channelSub

	return info, nil
}

// isValidAjaibID validates Ajaib ID
func isValidAjaibID(userID string) bool {
	return ajaibIDPattern.MatchString(userID)
}
