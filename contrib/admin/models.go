// Package admin mirrors django.contrib.admin.
//
// Django reference: django/contrib/admin/models.py
//                   django/contrib/admin/sites.py
//
// Django's admin has two layers:
//   1. LogEntry model  — records every add/change/delete action in django_admin_log
//   2. AdminSite       — registers models and auto-generates CRUD views
//
// djanGO implements:
//   - LogEntry model + LogActions() (same table, django_admin_log)
//   - site.Register() / admin.site pattern
//   - Auto-generated list/detail/add/change/delete views per registered model
//   - Staff-only access (mirrors Django's is_staff check on every admin view)
//
// Usage (mirrors Django):
//
//	// In your app's admin.go:
//	import "github.com/mqnifestkelvin/djanGO/contrib/admin"
//
//	func init() {
//	    admin.Site.Register("blog", "post", admin.ModelAdmin{})
//	}
//
// Django equivalent:
//
//	from django.contrib import admin
//	from .models import Post
//	admin.site.register(Post)
package admin

import (
	"time"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/core/models"
)

// Action flag constants mirror Django's ADDITION, CHANGE, DELETION.
//
// Django:
//
//	from django.contrib.admin.models import ADDITION, CHANGE, DELETION
const (
	ActionAddition = 1
	ActionChange   = 2
	ActionDeletion = 3
)

// LogEntry mirrors Django's django.contrib.admin.models.LogEntry.
//
// Django:
//
//	class LogEntry(models.Model):
//	    action_time    = DateTimeField(auto_now=True)
//	    user           = ForeignKey(User)
//	    content_type   = ForeignKey(ContentType, null=True)
//	    object_id      = TextField(null=True)
//	    object_repr    = CharField(max_length=200)
//	    action_flag    = PositiveSmallIntegerField()
//	    change_message = TextField(blank=True)
//
// Table: django_admin_log
type LogEntry struct {
	models.Model
	Id            int       `orm:"auto;pk"`
	ActionTime    time.Time `orm:"type(datetime);auto_now_add"`
	UserId        int       `orm:"column(user_id)"`
	ContentTypeId int       `orm:"column(content_type_id);null"`
	ObjectId      string    `orm:"type(text);null"`
	ObjectRepr    string    `orm:"size(200)"`
	ActionFlag    int       `orm:""`
	ChangeMessage string    `orm:"type(text)"`
}

func (l *LogEntry) TableName() string { return "django_admin_log" }

// IsAddition returns true if this log entry records an object creation.
func (l *LogEntry) IsAddition() bool { return l.ActionFlag == ActionAddition }

// IsChange returns true if this log entry records an object edit.
func (l *LogEntry) IsChange() bool { return l.ActionFlag == ActionChange }

// IsDeletion returns true if this log entry records an object deletion.
func (l *LogEntry) IsDeletion() bool { return l.ActionFlag == ActionDeletion }

// LogAction records an admin action in django_admin_log —
// mirrors Django's LogEntry.objects.log_actions().
//
// Django:
//
//	LogEntry.objects.log_actions(
//	    user_id=request.user.pk,
//	    queryset=[obj],
//	    action_flag=ADDITION,
//	    change_message="",
//	)
func LogAction(userID, contentTypeID int, objectID, objectRepr string, actionFlag int, changeMessage string) error {
	o := orm.NewOrm()
	entry := &LogEntry{
		UserId:        userID,
		ContentTypeId: contentTypeID,
		ObjectId:      objectID,
		ObjectRepr:    objectRepr,
		ActionFlag:    actionFlag,
		ChangeMessage: changeMessage,
	}
	_, err := o.Insert(entry)
	return err
}

func init() {
	orm.RegisterModel(&LogEntry{})
}
