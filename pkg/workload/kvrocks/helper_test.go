package kvrocks

import (
	"context"
	"fmt"
	"testing"

	redis "github.com/go-redis/redis/v8"
)

func TestNamespace(t *testing.T) {
	cli := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", "192.168.50.32", 6666),
		Password: "aaa",
		// other options with default
	})

	newcli := &kvrClient{cli}
	ns, err := newcli.ListNamespace(context.Background())
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	fmt.Printf("result: %v", ns[0].Token)
}
