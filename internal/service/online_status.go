package service

import "LanLine/internal/model"

// StatusVisibleUserIDs returns users allowed to receive userID's online status.
func StatusVisibleUserIDs(userID uint) ([]uint, error) {
	if userID == 0 {
		return nil, nil
	}

	visible := make(map[uint]struct{})

	var friendIDs []uint
	if err := model.DB.Model(&model.FriendRelation{}).
		Where("user_id = ? AND status = ?", userID, "accepted").
		Pluck("friend_id", &friendIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range friendIDs {
		if id != 0 && id != userID {
			visible[id] = struct{}{}
		}
	}
	friendIDs = friendIDs[:0]
	if err := model.DB.Model(&model.FriendRelation{}).
		Where("friend_id = ? AND status = ?", userID, "accepted").
		Pluck("user_id", &friendIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range friendIDs {
		if id != 0 && id != userID {
			visible[id] = struct{}{}
		}
	}

	var groupMemberIDs []uint
	if err := model.DB.Model(&model.GroupMember{}).
		Select("DISTINCT peer.user_id").
		Joins("JOIN group_members self ON self.group_id = group_members.group_id AND self.user_id = ?", userID).
		Joins("JOIN group_members peer ON peer.group_id = group_members.group_id").
		Where("peer.user_id <> ?", userID).
		Pluck("peer.user_id", &groupMemberIDs).Error; err != nil {
		return nil, err
	}
	for _, id := range groupMemberIDs {
		if id != 0 && id != userID {
			visible[id] = struct{}{}
		}
	}

	result := make([]uint, 0, len(visible))
	for id := range visible {
		result = append(result, id)
	}
	return result, nil
}
