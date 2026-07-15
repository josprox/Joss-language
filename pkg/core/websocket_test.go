package core

import "testing"

func TestMatchWebSocketRoutePattern(t *testing.T) {
	params, ok := matchRoutePattern("/rooms/{room}/users/{id}", "/rooms/general/users/42")
	if !ok || len(params) != 2 || params[0] != "general" || params[1] != "42" {
		t.Fatalf("matchRoutePattern returned %v, %v", params, ok)
	}
	if _, ok := matchRoutePattern("/rooms/{room}", "/rooms/a/messages"); ok {
		t.Fatal("route pattern accepted extra path segments")
	}
}

func TestWebSocketCloseInvokesCloser(t *testing.T) {
	r := NewRuntime()
	closed := false
	instance := &Instance{Fields: map[string]interface{}{"_closer": func() error {
		closed = true
		return nil
	}}}
	if got := r.executeWebSocketMethod(instance, "close", nil); got != true || !closed {
		t.Fatalf("close result=%v closed=%v", got, closed)
	}
}
