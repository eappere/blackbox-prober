package common

import (
	"fmt"
	"os"
	"time"
)

const DefaultReauthInterval = 20 * time.Minute

type ReauthState struct {
	lastConnectAt time.Time
}

func (s *ReauthState) MarkConnected(now time.Time) {
	s.lastConnectAt = now
}

func (s *ReauthState) LastConnectAt() time.Time {
	return s.lastConnectAt
}

func (s *ReauthState) ShouldReauth(interval time.Duration, now time.Time) bool {
	if interval <= 0 || s.lastConnectAt.IsZero() {
		return false
	}

	return now.Sub(s.lastConnectAt) >= interval
}

func ReauthIfNeeded(now time.Time, interval time.Duration, state *ReauthState, closeFn func() error, connectFn func() error) (bool, error) {
	if !state.ShouldReauth(interval, now) {
		return false, nil
	}

	if err := closeFn(); err != nil {
		return false, err
	}
	if err := connectFn(); err != nil {
		return false, err
	}

	return true, nil
}

func LoadBasicAuthCredentials(enabled bool, usernameEnv string, passwordEnv string) (string, string, error) {
	if !enabled {
		return "", "", nil
	}

	username, ok := os.LookupEnv(usernameEnv)
	if !ok {
		return "", "", fmt.Errorf("error: username not found in env (%s)", usernameEnv)
	}

	password, ok := os.LookupEnv(passwordEnv)
	if !ok {
		return "", "", fmt.Errorf("error: password not found in env (%s)", passwordEnv)
	}

	return username, password, nil
}
