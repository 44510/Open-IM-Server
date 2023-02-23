package relation

import (
	"OpenIM/pkg/common/constant"
	"OpenIM/pkg/common/db/table/relation"
	"OpenIM/pkg/common/tracelog"
	"OpenIM/pkg/utils"
	"context"
	"gorm.io/gorm"
)

var _ relation.GroupMemberModelInterface = (*GroupMemberGorm)(nil)

type GroupMemberGorm struct {
	DB *gorm.DB
}

func NewGroupMemberDB(db *gorm.DB) relation.GroupMemberModelInterface {
	return &GroupMemberGorm{DB: db}
}

func (g *GroupMemberGorm) NewTx(tx any) relation.GroupMemberModelInterface {
	return &GroupMemberGorm{DB: tx.(*gorm.DB)}
}

func (g *GroupMemberGorm) Create(ctx context.Context, groupMemberList []*relation.GroupMemberModel) (err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupMemberList", groupMemberList)
	}()
	return utils.Wrap(g.DB.Create(&groupMemberList).Error, "")
}

func (g *GroupMemberGorm) Delete(ctx context.Context, groupID string, userIDs []string) (err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupID", groupID, "userIDs", userIDs)
	}()
	return utils.Wrap(g.DB.Where("group_id = ? and user_id in (?)", groupID, userIDs).Delete(&relation.GroupMemberModel{}).Error, "")
}

func (g *GroupMemberGorm) DeleteGroup(ctx context.Context, groupIDs []string) (err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupIDs", groupIDs)
	}()
	return utils.Wrap(g.DB.Where("group_id in (?)", groupIDs).Delete(&relation.GroupMemberModel{}).Error, "")
}

func (g *GroupMemberGorm) Update(ctx context.Context, groupID string, userID string, data map[string]any) (err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupID", groupID, "userID", userID, "data", data)
	}()
	return utils.Wrap(g.DB.Model(&relation.GroupMemberModel{}).Where("group_id = ? and user_id = ?", groupID, userID).Updates(data).Error, "")
}

func (g *GroupMemberGorm) UpdateRoleLevel(ctx context.Context, groupID string, userID string, roleLevel int32) (rowsAffected int64, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupID", groupID, "userID", userID, "roleLevel", roleLevel)
	}()
	db := g.DB.Model(&relation.GroupMemberModel{}).Where("group_id = ? and user_id = ?", groupID, userID).Updates(map[string]any{
		"role_level": roleLevel,
	})
	return db.RowsAffected, utils.Wrap(db.Error, "")
}

func (g *GroupMemberGorm) Find(ctx context.Context, groupIDs []string, userIDs []string, roleLevels []int32) (groupList []*relation.GroupMemberModel, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupIDs", groupIDs, "userIDs", userIDs, "groupList", groupList)
	}()
	db := g.DB
	if len(groupIDs) > 0 {
		db = db.Where("group_id in (?)", groupIDs)
	}
	if len(userIDs) > 0 {
		db = db.Where("user_id in (?)", userIDs)
	}
	if len(roleLevels) > 0 {
		db = db.Where("role_level in (?)", roleLevels)
	}
	return groupList, utils.Wrap(db.Find(&groupList).Error, "")
}

func (g *GroupMemberGorm) Take(ctx context.Context, groupID string, userID string) (groupMember *relation.GroupMemberModel, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupID", groupID, "userID", userID, "groupMember", *groupMember)
	}()
	groupMember = &relation.GroupMemberModel{}
	return groupMember, utils.Wrap(g.DB.Where("group_id = ? and user_id = ?", groupID, userID).Take(groupMember).Error, "")
}

func (g *GroupMemberGorm) TakeOwner(ctx context.Context, groupID string) (groupMember *relation.GroupMemberModel, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupID", groupID, "groupMember", *groupMember)
	}()
	groupMember = &relation.GroupMemberModel{}
	return groupMember, utils.Wrap(g.DB.Where("group_id = ? and role_level = ?", groupID, constant.GroupOwner).Take(groupMember).Error, "")
}

func (g *GroupMemberGorm) SearchMember(ctx context.Context, keyword string, groupIDs []string, userIDs []string, roleLevels []int32, pageNumber, showNumber int32) (total uint32, groupList []*relation.GroupMemberModel, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "keyword", keyword, "groupIDs", groupIDs, "userIDs", userIDs, "roleLevels", roleLevels, "pageNumber", pageNumber, "showNumber", showNumber, "total", total, "groupList", groupList)
	}()
	db := g.DB
	gormIn(&db, "group_id", groupIDs)
	gormIn(&db, "user_id", userIDs)
	gormIn(&db, "role_level", roleLevels)
	return gormSearch[relation.GroupMemberModel](db, []string{"nickname"}, keyword, pageNumber, showNumber)
}

func (g *GroupMemberGorm) MapGroupMemberNum(ctx context.Context, groupIDs []string) (count map[string]uint32, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupIDs", groupIDs, "count", count)
	}()
	return mapCount(g.DB.Where("group_id in (?)", groupIDs), "group_id")
}

func (g *GroupMemberGorm) FindJoinUserID(ctx context.Context, groupIDs []string) (groupUsers map[string][]string, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupIDs", groupIDs, "groupUsers", groupUsers)
	}()
	var items []struct {
		GroupID string `gorm:"group_id"`
		UserID  string `gorm:"user_id"`
	}
	if err := g.DB.Model(&relation.GroupMemberModel{}).Where("group_id in (?)", groupIDs).Find(&items).Error; err != nil {
		return nil, utils.Wrap(err, "")
	}
	groupUsers = make(map[string][]string)
	for _, item := range items {
		groupUsers[item.GroupID] = append(groupUsers[item.GroupID], item.UserID)
	}
	return groupUsers, nil
}

func (g *GroupMemberGorm) FindMemberUserID(ctx context.Context, groupID string) (userIDs []string, err error) {
	defer func() {
		tracelog.SetCtxDebug(ctx, utils.GetFuncName(1), err, "groupID", groupID, "userIDs", userIDs)
	}()
	return userIDs, utils.Wrap(g.DB.Model(&relation.GroupMemberModel{}).Where("group_id = ?", groupID).Pluck("user_id", &userIDs).Error, "")
}
