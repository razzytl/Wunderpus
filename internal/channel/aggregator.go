package channel

import (
	"fmt"
)

// ChannelAggregator aggregates multiple channels and provides unified access
type ChannelAggregator struct {
	channels []Channel
}

// NewChannelAggregator creates a new channel aggregator
func NewChannelAggregator(channels []Channel) *ChannelAggregator {
	return &ChannelAggregator{
		channels: channels,
	}
}

// GetChannel returns the first available channel (for file sending)
func (a *ChannelAggregator) GetChannel() Channel {
	if len(a.channels) > 0 {
		return a.channels[0]
	}
	return nil
}

// SendFile sends a file through the first available channel
func (a *ChannelAggregator) SendFile(sessionID, filePath, caption string) error {
	channel := a.GetChannel()
	if channel == nil {
		return fmt.Errorf("no channels available")
	}

	// Try to get FileSender interface
	if fs, ok := channel.(FileSender); ok {
		return fs.SendFile(sessionID, filePath, caption)
	}

	// Try each channel until one works
	for _, ch := range a.channels {
		if fs, ok := ch.(FileSender); ok {
			return fs.SendFile(sessionID, filePath, caption)
		}
	}

	return fmt.Errorf("no channel supports file sending")
}

// ChannelNames returns the names of all registered channels
func (a *ChannelAggregator) ChannelNames() []string {
	names := make([]string, len(a.channels))
	for i, ch := range a.channels {
		names[i] = ch.Name()
	}
	return names
}
