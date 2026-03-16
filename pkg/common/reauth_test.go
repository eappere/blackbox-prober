package common

import (
	"errors"
	"testing"
	"time"
)

func TestReauthStateShouldReauth(t *testing.T) {
	state := ReauthState{}
	now := time.Now()

	if state.ShouldReauth(DefaultReauthInterval, now) {
		t.Fatal("expected zero-value state to skip reauth")
	}

	state.MarkConnected(now.Add(-DefaultReauthInterval))
	if !state.ShouldReauth(DefaultReauthInterval, now) {
		t.Fatal("expected reauth when the interval is reached")
	}

	if state.ShouldReauth(0, now) {
		t.Fatal("expected disabled interval to skip reauth")
	}
}

func TestReauthIfNeeded(t *testing.T) {
	state := ReauthState{}
	now := time.Now()
	state.MarkConnected(now.Add(-DefaultReauthInterval))

	closeCalls := 0
	connectCalls := 0

	reauthed, err := ReauthIfNeeded(now, DefaultReauthInterval, &state, func() error {
		closeCalls++
		return nil
	}, func() error {
		connectCalls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reauthed {
		t.Fatal("expected reauth to run")
	}
	if closeCalls != 1 || connectCalls != 1 {
		t.Fatalf("expected close/connect to be called once, got %d/%d", closeCalls, connectCalls)
	}
}

func TestReauthIfNeededPropagatesErrors(t *testing.T) {
	state := ReauthState{}
	now := time.Now()
	state.MarkConnected(now.Add(-DefaultReauthInterval))

	expected := errors.New("connect failed")
	_, err := ReauthIfNeeded(now, DefaultReauthInterval, &state, func() error {
		return nil
	}, func() error {
		return expected
	})
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestLoadBasicAuthCredentials(t *testing.T) {
	t.Setenv("TEST_USER", "alice")
	t.Setenv("TEST_PASS", "secret")

	username, password, err := LoadBasicAuthCredentials(true, "TEST_USER", "TEST_PASS")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if username != "alice" || password != "secret" {
		t.Fatalf("unexpected credentials: %s/%s", username, password)
	}
}
