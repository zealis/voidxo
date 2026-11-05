package cache

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNoopAlwaysMisses(t *testing.T) {
	var c Cache = Noop{}
	var target map[string]any
	ok, err := c.Get(context.Background(), "scope", "key", &target)
	if err != nil || ok {
		t.Fatalf("noop must miss: ok=%v err=%v", ok, err)
	}
	if err := c.Set(context.Background(), "scope", "key", target, time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Bump(context.Background(), "scope"); err != nil {
		t.Fatal(err)
	}
	if version, err := c.Version(context.Background(), "scope"); err != nil || version != 0 {
		t.Fatalf("noop version must be 0: %d %v", version, err)
	}
}

func TestKeyLayoutIncludesScopeAndVersion(t *testing.T) {
	if got := versionKey("content"); got != "voidxo:cache:v:content" {
		t.Fatalf("unexpected version key: %s", got)
	}
	key := dataKey("content", 3, "topics:1:20")
	if !strings.HasPrefix(key, "voidxo:cache:content:3:") || !strings.HasSuffix(key, "topics:1:20") {
		t.Fatalf("unexpected data key: %s", key)
	}
}
