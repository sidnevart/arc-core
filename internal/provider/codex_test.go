package provider

import "testing"

func TestSandboxForRequestUsesReadOnlyForReplyOnlyChat(t *testing.T) {
	got := sandboxForRequest(TaskRequest{
		Mode:      "hero",
		ReplyOnly: true,
	})
	if got != "read-only" {
		t.Fatalf("expected read-only sandbox for reply-only chat, got %q", got)
	}
}
