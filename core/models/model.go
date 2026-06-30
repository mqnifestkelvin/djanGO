package models

import (
	"github.com/mqnifestkelvin/djanGO/client/orm"
)

// Model is the base struct every djanGO model embeds — mirrors Django's Model base class.
//
// Django:
//
//	class Post(models.Model):
//	    title = models.CharField(max_length=200)
//	    slug  = models.SlugField(unique=True)
//
// djanGO:
//
//	type Post struct {
//	    models.Model
//	    Title string `orm:"size(200)"`
//	    Slug  string `orm:"unique"`
//	}
//	var Objects = models.NewManager(func() Post { return Post{} })
//
// The embedded Model provides Save(), Delete(), and Pk().
// The package-level Objects variable provides all() / filter() / get() / create().
type Model struct{}

// Save mirrors Django's Model.save() — INSERT if new, UPDATE if existing.
// Beego's ORM determines insert vs update based on whether the PK is zero.
//
// Django:
//
//	post.title = "Updated"
//	post.save()
//
// djanGO:
//
//	post.Title = "Updated"
//	post.Save()
func (m *Model) Save(instance interface{}) error {
	o := orm.NewOrm()

	// Beego: Insert if pk is zero, Update otherwise —
	// mirrors Django's Model.save(force_insert / force_update logic)
	_, err := o.InsertOrUpdate(instance)
	return err
}

// Delete mirrors Django's Model.delete() — removes the row from the database.
//
// Django:
//
//	post.delete()
//
// djanGO:
//
//	post.Delete()
func (m *Model) Delete(instance interface{}) error {
	o := orm.NewOrm()
	_, err := o.Delete(instance)
	return err
}
