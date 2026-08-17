package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func CheckProviderContract(t *testing.T, provider Provider) {
	t.Helper()
	ctx := context.Background()
	ns := fmt.Sprintf("contract/%d", time.Now().UnixNano())

	t.Run("Capabilities", func(t *testing.T) {
		caps := provider.Capabilities()
		if !caps.Streaming {
			t.Error("expected Streaming capability")
		}
		if !caps.CopyObject {
			t.Error("expected CopyObject capability")
		}
	})

	key := ns + "/hello.txt"
	body := "hello, world"

	t.Run("PutObject", func(t *testing.T) {
		out, err := provider.PutObject(ctx, PutObjectInput{
			Key:         key,
			Body:        strings.NewReader(body),
			ContentType: "text/plain",
			Metadata:    map[string]string{"env": "test"},
		})
		if err != nil {
			t.Fatalf("PutObject: %v", err)
		}
		if out.Key != key {
			t.Errorf("Key = %q, want %q", out.Key, key)
		}
		if out.Size != int64(len(body)) {
			t.Errorf("Size = %d, want %d", out.Size, len(body))
		}
	})

	t.Run("HeadObject", func(t *testing.T) {
		head, err := provider.HeadObject(ctx, HeadObjectInput{Key: key})
		if err != nil {
			t.Fatalf("HeadObject: %v", err)
		}
		if head.ContentLength != int64(len(body)) {
			t.Errorf("ContentLength = %d, want %d", head.ContentLength, len(body))
		}
		if head.ContentType != "text/plain" {
			t.Errorf("ContentType = %q, want %q", head.ContentType, "text/plain")
		}
		if head.Metadata["env"] != "test" {
			t.Errorf("Metadata[env] = %q, want %q", head.Metadata["env"], "test")
		}
	})

	t.Run("GetObject", func(t *testing.T) {
		obj, err := provider.GetObject(ctx, GetObjectInput{Key: key})
		if err != nil {
			t.Fatalf("GetObject: %v", err)
		}
		defer obj.Body.Close()
		got, err := io.ReadAll(obj.Body)
		if err != nil {
			t.Fatalf("reading body: %v", err)
		}
		if string(got) != body {
			t.Errorf("body = %q, want %q", got, body)
		}
		if obj.ContentType != "text/plain" {
			t.Errorf("ContentType = %q, want %q", obj.ContentType, "text/plain")
		}
		if obj.Metadata["env"] != "test" {
			t.Errorf("Metadata[env] = %q, want %q", obj.Metadata["env"], "test")
		}
	})

	t.Run("GetObjectMissing", func(t *testing.T) {
		_, err := provider.GetObject(ctx, GetObjectInput{Key: ns + "/missing.txt"})
		if !errors.Is(err, ErrObjectNotFound) {
			t.Errorf("err = %v, want ErrObjectNotFound", err)
		}
	})

	t.Run("HeadObjectMissing", func(t *testing.T) {
		_, err := provider.HeadObject(ctx, HeadObjectInput{Key: ns + "/missing.txt"})
		if !errors.Is(err, ErrObjectNotFound) {
			t.Errorf("err = %v, want ErrObjectNotFound", err)
		}
	})

	t.Run("InvalidKeys", func(t *testing.T) {
		for _, bad := range []string{"", "../escape", "a/../b", "a//b", "/"} {
			if _, err := provider.PutObject(ctx, PutObjectInput{Key: bad, Body: strings.NewReader("x")}); !errors.Is(err, ErrInvalidObjectKey) {
				t.Errorf("PutObject(%q) err = %v, want ErrInvalidObjectKey", bad, err)
			}
		}
	})

	t.Run("Overwrite", func(t *testing.T) {
		ow := ns + "/overwrite.txt"
		if _, err := provider.PutObject(ctx, PutObjectInput{Key: ow, Body: strings.NewReader("v1")}); err != nil {
			t.Fatalf("PutObject v1: %v", err)
		}
		if _, err := provider.PutObject(ctx, PutObjectInput{Key: ow, Body: strings.NewReader("v2")}); err != nil {
			t.Fatalf("PutObject v2: %v", err)
		}
		obj, err := provider.GetObject(ctx, GetObjectInput{Key: ow})
		if err != nil {
			t.Fatalf("GetObject: %v", err)
		}
		defer obj.Body.Close()
		data, _ := io.ReadAll(obj.Body)
		if string(data) != "v2" {
			t.Errorf("after overwrite body = %q, want %q", data, "v2")
		}
	})

	t.Run("NestedKeys", func(t *testing.T) {
		nested := ns + "/a/b/c/deep.txt"
		if _, err := provider.PutObject(ctx, PutObjectInput{Key: nested, Body: strings.NewReader("deep")}); err != nil {
			t.Fatalf("PutObject nested: %v", err)
		}
		obj, err := provider.GetObject(ctx, GetObjectInput{Key: nested})
		if err != nil {
			t.Fatalf("GetObject nested: %v", err)
		}
		defer obj.Body.Close()
		data, _ := io.ReadAll(obj.Body)
		if string(data) != "deep" {
			t.Errorf("nested body = %q, want %q", data, "deep")
		}
	})

	t.Run("CopyObject", func(t *testing.T) {
		dst := ns + "/copied.txt"
		out, err := provider.CopyObject(ctx, CopyObjectInput{SourceKey: key, DestinationKey: dst})
		if err != nil {
			t.Fatalf("CopyObject: %v", err)
		}
		if out.SourceKey != key || out.DestinationKey != dst {
			t.Errorf("copy keys = %q -> %q", out.SourceKey, out.DestinationKey)
		}
		obj, err := provider.GetObject(ctx, GetObjectInput{Key: dst})
		if err != nil {
			t.Fatalf("GetObject copy: %v", err)
		}
		defer obj.Body.Close()
		data, _ := io.ReadAll(obj.Body)
		if string(data) != body {
			t.Errorf("copy body = %q, want %q", data, body)
		}
	})

	t.Run("ListObjects", func(t *testing.T) {
		prefix := ns + "/"
		out, err := provider.ListObjects(ctx, ListObjectsInput{Prefix: prefix, Limit: 100})
		if err != nil {
			t.Fatalf("ListObjects: %v", err)
		}
		if len(out.Objects) == 0 {
			t.Fatal("expected at least one object under prefix")
		}
		for _, o := range out.Objects {
			if !strings.HasPrefix(o.Key, prefix) {
				t.Errorf("Key %q does not start with prefix %q", o.Key, prefix)
			}
		}
	})

	t.Run("ListObjectsPagination", func(t *testing.T) {
		prefix := ns + "/pages/"
		for i := 0; i < 5; i++ {
			k := fmt.Sprintf("%sfile-%d.txt", prefix, i)
			if _, err := provider.PutObject(ctx, PutObjectInput{Key: k, Body: strings.NewReader("p")}); err != nil {
				t.Fatalf("PutObject page %d: %v", i, err)
			}
		}
		seen := map[string]bool{}
		cursor := ""
		for {
			out, err := provider.ListObjects(ctx, ListObjectsInput{Prefix: prefix, Limit: 2, Cursor: cursor})
			if err != nil {
				t.Fatalf("ListObjects page: %v", err)
			}
			for _, o := range out.Objects {
				seen[o.Key] = true
			}
			if out.NextCursor == "" {
				break
			}
			cursor = out.NextCursor
		}
		for i := 0; i < 5; i++ {
			k := fmt.Sprintf("%sfile-%d.txt", prefix, i)
			if !seen[k] {
				t.Errorf("pagination missed key %q", k)
			}
		}
	})

	t.Run("DeleteObject", func(t *testing.T) {
		del := ns + "/delete.txt"
		if _, err := provider.PutObject(ctx, PutObjectInput{Key: del, Body: strings.NewReader("bye")}); err != nil {
			t.Fatalf("PutObject delete: %v", err)
		}
		if err := provider.DeleteObject(ctx, DeleteObjectInput{Key: del}); err != nil {
			t.Fatalf("DeleteObject: %v", err)
		}
		if err := provider.DeleteObject(ctx, DeleteObjectInput{Key: del}); err != nil {
			t.Errorf("DeleteObject again must be idempotent, got %v", err)
		}
		if _, err := provider.HeadObject(ctx, HeadObjectInput{Key: del}); !errors.Is(err, ErrObjectNotFound) {
			t.Errorf("HeadObject after delete err = %v, want ErrObjectNotFound", err)
		}
	})
}
