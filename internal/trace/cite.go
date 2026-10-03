package trace

import (
	"errors"
	"strings"
)

// ErrCiteDisabled is returned when cite is called with log_commands off.
var ErrCiteDisabled = errors.New("log_commands is false; cite was not recorded")

// PrepareCite checks that logging is on and effect is a single non-empty line.
func PrepareCite(loggingOn bool, effect string) (string, error) {
	if !loggingOn {
		return "", ErrCiteDisabled
	}
	effect = strings.TrimSpace(effect)
	if effect == "" {
		return "", errors.New("cite effect is required")
	}
	if strings.ContainsAny(effect, "\n\r") {
		return "", errors.New("cite effect must be one line")
	}
	return effect, nil
}
