package handlers

import (
	"asily_blog/internal/models"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestResolveRootCommentID(t *testing.T) {
	topID := primitive.NewObjectID()
	replyID := primitive.NewObjectID()

	topComment := models.Comment{
		ID: topID,
	}
	if got := resolveRootCommentID(topComment); got != topID {
		t.Fatalf("top-level root id mismatch: got %s want %s", got.Hex(), topID.Hex())
	}

	replyComment := models.Comment{
		ID:       replyID,
		ParentID: &topID,
		RootID:   &topID,
	}
	if got := resolveRootCommentID(replyComment); got != topID {
		t.Fatalf("reply root id mismatch: got %s want %s", got.Hex(), topID.Hex())
	}
}

func TestBuildReplyContent(t *testing.T) {
	prefix := buildReplyPrefix("alice")
	if prefix != "回复 @alice：" {
		t.Fatalf("unexpected prefix: %s", prefix)
	}

	display := buildDisplayContent(prefix, "你好")
	if display != "回复 @alice：你好" {
		t.Fatalf("unexpected display content: %s", display)
	}

	if got := buildReplyPrefix(""); got != "" {
		t.Fatalf("expected empty prefix, got %s", got)
	}
}
