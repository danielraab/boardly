package memos

import (
	"encoding/json"
	"testing"
)

func TestMatchesFilter(t *testing.T) {
	for filter, want := range map[string]bool{
		"":                                      true,
		`creator == "users/u1"`:                 true,
		`creator == "users/u2"`:                 false,
		`visibility in ["PUBLIC", "PROTECTED"]`: false,
		`visibility in ["PRIVATE"]`:             true,
		`visibility == "PRIVATE"`:               true,
		`visibility == PUBLIC`:                  false,
		`creator == "users/u1" && visibility in ["PUBLIC"]`: false,
		`creator == "users/u1" && tag in ["x"]`:             true,
	} {
		if got := matchesFilter(filter, "u1"); got != want {
			t.Errorf("%s = %v, want %v", filter, got, want)
		}
	}
}

func TestUpdateFromBody(t *testing.T) {
	body := map[string]json.RawMessage{
		"content": json.RawMessage(`"text"`), "pinned": json.RawMessage(`true`),
		"state": json.RawMessage(`"ARCHIVED"`), "visibility": json.RawMessage(`"PUBLIC"`), "location": json.RawMessage(`null`),
	}
	in, err := UpdateFromBody(body, "")
	if err != nil || in.Content == nil || in.Pinned == nil || in.State == nil {
		t.Fatalf("without a mask = %+v, %v", in, err)
	}
	in, err = UpdateFromBody(body, "pinned,update_time")
	if err != nil || in.Content != nil || in.Pinned == nil || in.State != nil {
		t.Errorf("with a mask = %+v, %v", in, err)
	}
	if _, err := UpdateFromBody(map[string]json.RawMessage{"pinned": json.RawMessage(`"yes"`)}, ""); err == nil {
		t.Error("a string as pinned was accepted")
	}
}

func TestPageToken(t *testing.T) {
	if n, err := pageOffset(pageToken(200)); err != nil || n != 200 {
		t.Errorf("round trip = %d, %v", n, err)
	}
	for _, bad := range []string{"!", pageToken(-1)} {
		if _, err := pageOffset(bad); err == nil {
			t.Errorf("token %q accepted", bad)
		}
	}
}
