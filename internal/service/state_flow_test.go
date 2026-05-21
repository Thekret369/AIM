package service

import (
	"testing"

	"AIM/internal/model"
)

func TestHandleRequestAcceptsExistingReverseWithoutDuplicate(t *testing.T) {
	_, _ = setupChatSecurityTest(t)
	alice := createSecurityUser(t, "state_friend_alice")
	bob := createSecurityUser(t, "state_friend_bob")
	friendSvc := &FriendService{}

	request := model.FriendRelation{UserID: alice.ID, FriendID: bob.ID, Status: "pending"}
	reverse := model.FriendRelation{UserID: bob.ID, FriendID: alice.ID, Status: "pending"}
	if err := model.DB.Create(&request).Error; err != nil {
		t.Fatalf("create request: %v", err)
	}
	if err := model.DB.Create(&reverse).Error; err != nil {
		t.Fatalf("create reverse request: %v", err)
	}

	if err := friendSvc.HandleRequest(request.ID, bob.ID, true); err != nil {
		t.Fatalf("accept request: %v", err)
	}

	var count int64
	if err := model.DB.Model(&model.FriendRelation{}).
		Where("((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)) AND status = ?",
			alice.ID, bob.ID, bob.ID, alice.ID, "accepted").
		Count(&count).Error; err != nil {
		t.Fatalf("count accepted relations: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected two accepted relation rows, got %d", count)
	}
}

func TestJoinGroupRejectsDuplicateMember(t *testing.T) {
	_, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "state_group_owner")
	member := createSecurityUser(t, "state_group_member")
	group, err := groupSvc.CreateGroup("状态流转群", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}

	if err := groupSvc.JoinGroup(group.ID, member.ID); err != nil {
		t.Fatalf("join group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, member.ID); err == nil {
		t.Fatal("expected duplicate join to be rejected")
	}

	var count int64
	if err := model.DB.Model(&model.GroupMember{}).
		Where("group_id = ? AND user_id = ?", group.ID, member.ID).
		Count(&count).Error; err != nil {
		t.Fatalf("count group members: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one group member row, got %d", count)
	}
}

func TestTransferOwnerUpdatesGroupAndRoles(t *testing.T) {
	_, groupSvc := setupChatSecurityTest(t)
	owner := createSecurityUser(t, "state_transfer_owner")
	nextOwner := createSecurityUser(t, "state_transfer_next")
	group, err := groupSvc.CreateGroup("群主转让群", "", owner.ID)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if err := groupSvc.JoinGroup(group.ID, nextOwner.ID); err != nil {
		t.Fatalf("join next owner: %v", err)
	}

	if err := groupSvc.TransferOwner(group.ID, owner.ID, nextOwner.ID); err != nil {
		t.Fatalf("transfer owner: %v", err)
	}

	var updated model.Group
	if err := model.DB.First(&updated, group.ID).Error; err != nil {
		t.Fatalf("load group: %v", err)
	}
	if updated.OwnerID != nextOwner.ID {
		t.Fatalf("expected owner %d, got %d", nextOwner.ID, updated.OwnerID)
	}

	var oldMember, newMember model.GroupMember
	if err := model.DB.Where("group_id = ? AND user_id = ?", group.ID, owner.ID).First(&oldMember).Error; err != nil {
		t.Fatalf("load old owner member: %v", err)
	}
	if err := model.DB.Where("group_id = ? AND user_id = ?", group.ID, nextOwner.ID).First(&newMember).Error; err != nil {
		t.Fatalf("load new owner member: %v", err)
	}
	if oldMember.Role != model.RoleAdmin || newMember.Role != model.RoleOwner {
		t.Fatalf("unexpected roles: old=%s new=%s", oldMember.Role, newMember.Role)
	}
}

func TestContactGroupDeleteClearsFriendAssignments(t *testing.T) {
	_, _ = setupChatSecurityTest(t)
	alice := createSecurityUser(t, "state_contact_alice")
	bob := createSecurityUser(t, "state_contact_bob")
	contactSvc := &ContactGroupService{}
	friendSvc := &FriendService{}

	group, err := contactSvc.Create(alice.ID, "同事")
	if err != nil {
		t.Fatalf("create contact group: %v", err)
	}
	createAcceptedFriendPair(t, alice.ID, bob.ID)
	if err := friendSvc.UpdateRemark(alice.ID, bob.ID, "", "", &group.ID); err != nil {
		t.Fatalf("assign contact group: %v", err)
	}

	if err := contactSvc.Delete(group.ID, alice.ID); err != nil {
		t.Fatalf("delete contact group: %v", err)
	}

	var rel model.FriendRelation
	if err := model.DB.Where("user_id = ? AND friend_id = ?", alice.ID, bob.ID).First(&rel).Error; err != nil {
		t.Fatalf("load relation: %v", err)
	}
	if rel.GroupID != nil {
		t.Fatalf("expected group_id to be cleared, got %d", *rel.GroupID)
	}
}
