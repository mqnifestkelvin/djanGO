package auth

// Groups subsystem — mirrors Django's User.groups M2M and Group API.
//
// Django:
//
//	user.groups.add(group)
//	user.groups.remove(group)
//	user.groups.all()
//	group.permissions.add(permission)
//	Group.objects.create(name="editors")

import (
	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/core/models"
)

// GroupObjects is the default manager for Group — mirrors Group.objects.
var GroupObjects = models.NewManager(func() Group { return Group{} })

// CreateGroup creates a new group — mirrors Group.objects.create(name="editors").
//
// Django:
//
//	from django.contrib.auth.models import Group
//	editors = Group.objects.create(name="editors")
func CreateGroup(name string) (*Group, error) {
	g := &Group{Name: name}
	o := orm.NewOrm()
	if _, err := o.Insert(g); err != nil {
		return nil, err
	}
	return g, nil
}

// GetGroup fetches a group by name — mirrors Group.objects.get(name="editors").
func GetGroup(name string) (*Group, error) {
	o := orm.NewOrm()
	g := &Group{}
	if err := o.QueryTable("auth_group").Filter("Name", name).One(g); err != nil {
		return nil, err
	}
	return g, nil
}

// AddToGroup adds the user to a group — mirrors user.groups.add(group).
//
// Django:
//
//	user.groups.add(editors_group)
func (u *User) AddToGroup(g *Group) error {
	o := orm.NewOrm()
	// Check if already a member.
	var count int
	_ = o.Raw(`SELECT COUNT(*) FROM auth_user_groups WHERE user_id=? AND group_id=?`,
		u.Id, g.Id).QueryRow(&count)
	if count > 0 {
		return nil // already in group
	}
	ug := &UserGroup{UserId: u.Id, GroupId: g.Id}
	_, err := o.Insert(ug)
	return err
}

// RemoveFromGroup removes the user from a group — mirrors user.groups.remove(group).
//
// Django:
//
//	user.groups.remove(editors_group)
func (u *User) RemoveFromGroup(g *Group) error {
	o := orm.NewOrm()
	_, err := o.Raw(`DELETE FROM auth_user_groups WHERE user_id=? AND group_id=?`,
		u.Id, g.Id).Exec()
	return err
}

// UserGroups returns all groups this user belongs to — mirrors user.groups.all().
//
// Django:
//
//	user.groups.all()
func (u *User) UserGroups() ([]*Group, error) {
	o := orm.NewOrm()
	var groups []*Group
	_, err := o.Raw(`
		SELECT g.id, g.name FROM auth_group g
		JOIN auth_user_groups ug ON ug.group_id = g.id
		WHERE ug.user_id = ?
	`, u.Id).QueryRows(&groups)
	return groups, err
}

// AddPermission adds a direct permission to a user — mirrors user.user_permissions.add(perm).
//
// Django:
//
//	from django.contrib.auth.models import Permission
//	perm = Permission.objects.get(codename="add_post")
//	user.user_permissions.add(perm)
func (u *User) AddPermission(p *Permission) error {
	o := orm.NewOrm()
	var count int
	_ = o.Raw(`SELECT COUNT(*) FROM auth_user_user_permissions WHERE user_id=? AND permission_id=?`,
		u.Id, p.Id).QueryRow(&count)
	if count > 0 {
		return nil
	}
	up := &UserPermission{UserId: u.Id, PermissionId: p.Id}
	_, err := o.Insert(up)
	return err
}

// AddPermissionToGroup adds a permission to a group — mirrors group.permissions.add(perm).
//
// Django:
//
//	editors.permissions.add(perm)
func (g *Group) AddPermission(p *Permission) error {
	o := orm.NewOrm()
	var count int
	_ = o.Raw(`SELECT COUNT(*) FROM auth_group_permissions WHERE group_id=? AND permission_id=?`,
		g.Id, p.Id).QueryRow(&count)
	if count > 0 {
		return nil
	}
	gp := &GroupPermission{GroupId: g.Id, PermissionId: p.Id}
	_, err := o.Insert(gp)
	return err
}

// GetPermission fetches a Permission by app_label and codename —
// mirrors Permission.objects.get(content_type__app_label="blog", codename="add_post").
//
// Django:
//
//	from django.contrib.auth.models import Permission
//	perm = Permission.objects.get(content_type__app_label="blog", codename="add_post")
func GetPermission(appLabel, codename string) (*Permission, error) {
	o := orm.NewOrm()
	p := &Permission{}
	err := o.Raw(`
		SELECT p.id, p.name, p.content_type_id, p.codename
		FROM auth_permission p
		JOIN django_content_type ct ON ct.id = p.content_type_id
		WHERE ct.app_label = ? AND p.codename = ?
	`, appLabel, codename).QueryRow(p)
	if err != nil {
		return nil, err
	}
	return p, nil
}
