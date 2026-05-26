package service

import (
	"testing"

	"AIM/internal/model"
)

func TestStatusVisibleUserIDsIncludesFriendsAndSharedGroupMembers(t *testing.T) {
	_, groupSvc := setupChatSecurityTest(t)
	alice := createSecurityUser(t, "status_alice")
	bob := createSecurityUser(t, "status_bob")
	carol := createSecurityUser(t, "status_carol")
	dave := createSecurityUser(t, "status_dave")
	erin := createSecurityUser(t, "status_erin")

	createAcceptedFriendPair(t, alice.ID, bob.ID)
	createAcceptedFriendPair(t, alice.ID, erin.ID)

	group, err := groupSvc.CreateGroup("status-group", "", alice.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, carol.ID); err != nil {
		t.Fatalf("join carol: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, erin.ID); err != nil {
		t.Fatalf("join erin: %v", err)
	}

	ids, err := StatusVisibleUserIDs(alice.ID)
	if err != nil {
		t.Fatalf("status visible users: %v", err)
	}
	got := idSet(ids)
	for _, want := range []uint{bob.ID, carol.ID, erin.ID} {
		if !got[want] {
			t.Fatalf("expected user %d to see status, got ids=%v", want, ids)
		}
	}
	if got[alice.ID] {
		t.Fatalf("user should not receive its own status, got ids=%v", ids)
	}
	if got[dave.ID] {
		t.Fatalf("unrelated user should not see status, got ids=%v", ids)
	}
}

func TestStatusVisibleUserIDsIgnoresPendingFriends(t *testing.T) {
	_, _ = setupChatSecurityTest(t)
	alice := createSecurityUser(t, "status_pending_alice")
	bob := createSecurityUser(t, "status_pending_bob")

	if err := model.DB.Create(&model.FriendRelation{
		UserID:   alice.ID,
		FriendID: bob.ID,
		Status:   "pending",
	}).Error; err != nil {
		t.Fatalf("create pending relation: %v", err)
	}

	ids, err := StatusVisibleUserIDs(alice.ID)
	if err != nil {
		t.Fatalf("status visible users: %v", err)
	}
	if idSet(ids)[bob.ID] {
		t.Fatalf("pending friend should not see status, got ids=%v", ids)
	}
}

func TestStatusVisibleUserIDsAcceptsReverseFriendRelation(t *testing.T) {
	_, _ = setupChatSecurityTest(t)
	alice := createSecurityUser(t, "status_reverse_alice")
	bob := createSecurityUser(t, "status_reverse_bob")

	if err := model.DB.Create(&model.FriendRelation{
		UserID:   bob.ID,
		FriendID: alice.ID,
		Status:   "accepted",
	}).Error; err != nil {
		t.Fatalf("create reverse accepted relation: %v", err)
	}

	ids, err := StatusVisibleUserIDs(alice.ID)
	if err != nil {
		t.Fatalf("status visible users: %v", err)
	}
	if !idSet(ids)[bob.ID] {
		t.Fatalf("reverse accepted friend should see status, got ids=%v", ids)
	}
}

func idSet(ids []uint) map[uint]bool {
	result := make(map[uint]bool, len(ids))
	for _, id := range ids {
		result[id] = true
	}
	return result
}
