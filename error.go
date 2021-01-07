package websocket

import "errors"

var ErrClientIsClosed = errors.New("websocket client is closed")
var ErrClientNotFound = errors.New("websocket client not found")
