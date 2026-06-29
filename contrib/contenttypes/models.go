// Package contenttypes mirrors Django's django.contrib.contenttypes.
//
// Django reference: django/contrib/contenttypes/models.py
//
// ContentType tracks every model registered in the application.
// Django stores one row per model: (app_label, model_name) → id.
// This id is used by GenericForeignKey to create cross-model relations —
// e.g. the Permission model references a ContentType to say
// "this permission applies to the auth.User model".
//
// Table: django_content_type
// Django:
//
//	from django.contrib.contenttypes.models import ContentType
//	ct = ContentType.objects.get_for_model(MyModel)
//	ct.app_label   # "blog"
//	ct.model       # "post"
package contenttypes

import (
	"sync"

	"github.com/mqnifestkelvin/djanGO/client/orm"
	"github.com/mqnifestkelvin/djanGO/core/models"
)

// ContentType mirrors Django's ContentType model.
//
// Django:
//
//	class ContentType(models.Model):
//	    app_label = models.CharField(max_length=100)
//	    model = models.CharField(max_length=100)
type ContentType struct {
	models.Model
	Id        int    `orm:"auto;pk"`
	AppLabel  string `orm:"size(100)"`
	ModelName string `orm:"size(100);column(model)"`
}

func (ct *ContentType) TableName() string { return "django_content_type" }

// TableUnique mirrors Django's unique_together = [("app_label", "model")] on ContentType.
func (ct *ContentType) TableUnique() [][]string {
	return [][]string{{"app_label", "model"}}
}

// Name returns "app_label.model" — mirrors Django's ContentType.__str__.
func (ct *ContentType) Name() string { return ct.AppLabel + "." + ct.ModelName }

var (
	cache   = map[[2]string]*ContentType{}
	cacheMu sync.RWMutex
)

// GetForModel returns the ContentType for the given app/model pair,
// creating it if it doesn't exist — mirrors Django's ContentType.objects.get_for_model().
//
// Django:
//
//	ContentType.objects.get_for_model(Post)
//	# → ContentType(app_label="blog", model="post")
func GetForModel(appLabel, modelName string) (*ContentType, error) {
	key := [2]string{appLabel, modelName}

	cacheMu.RLock()
	if ct, ok := cache[key]; ok {
		cacheMu.RUnlock()
		return ct, nil
	}
	cacheMu.RUnlock()

	o := orm.NewOrm()
	ct := &ContentType{}
	err := o.QueryTable("django_content_type").
		Filter("AppLabel", appLabel).
		Filter("ModelName", modelName).
		One(ct)

	if err != nil {
		// Not found — create it (mirrors Django's get_or_create path)
		ct = &ContentType{AppLabel: appLabel, ModelName: modelName}
		if _, insertErr := o.Insert(ct); insertErr != nil {
			return nil, insertErr
		}
	}

	cacheMu.Lock()
	cache[key] = ct
	cacheMu.Unlock()

	return ct, nil
}

// GetByID returns a ContentType by its primary key.
func GetByID(id int) (*ContentType, error) {
	o := orm.NewOrm()
	ct := &ContentType{Id: id}
	if err := o.Read(ct); err != nil {
		return nil, err
	}
	return ct, nil
}

// Objects is the default manager.
var Objects = models.NewManager(func() ContentType { return ContentType{} })

func init() {
	orm.RegisterModel(&ContentType{})
}
