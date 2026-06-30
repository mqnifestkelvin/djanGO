// Package models mirrors Django's django.db.models module.
//
// Django's ORM centres on three objects:
//   - Model      — the base class every model embeds
//   - Manager    — attached as Model.objects, entry point for queries
//   - QuerySet   — lazy, chainable query builder returned by the manager
//
// djanGO wraps Beego's ORM (which handles the actual SQL) behind Django's
// exact API so that view code reads identically to Django:
//
// Django:
//
//	posts = Post.objects.all()
//	post  = Post.objects.get(slug="hello")
//	posts = Post.objects.filter(published=True).order_by("-created_at")
//	post.save()
//	post.delete()
//
// djanGO:
//
//	posts, _ := Post.Objects.All()
//	post,  _ := Post.Objects.Get("slug", "hello")
//	posts, _ := Post.Objects.Filter("published", true).OrderBy("-created_at").All()
//	post.Save()
//	post.Delete()
package models

import (
	"fmt"

	"github.com/mqnifestkelvin/djanGO/client/orm"
)

// QuerySet mirrors Django's QuerySet — a lazy, chainable query builder.
//
// Django:  Post.objects.filter(published=True).order_by("-created_at")[:10]
// djanGO:  Post.Objects.Filter("published", true).OrderBy("-created_at").Limit(10)
type QuerySet[T any] struct {
	o        orm.Ormer
	model    T           // zero value used to tell Beego the table
	filters  [][2]interface{} // [field, value] pairs — mirroring Q objects
	excludes [][2]interface{}
	orderBy  []string
	limit    int64
	offset   int64
}

func newQuerySet[T any](o orm.Ormer, model T) *QuerySet[T] {
	return &QuerySet[T]{o: o, model: model}
}

// Filter mirrors Django's QuerySet.filter(**kwargs).
//
// Django:  Post.objects.filter(published=True)
// djanGO:  Post.Objects.Filter("published", true)
func (qs *QuerySet[T]) Filter(field string, value interface{}) *QuerySet[T] {
	clone := qs.clone()
	clone.filters = append(clone.filters, [2]interface{}{field, value})
	return clone
}

// Exclude mirrors Django's QuerySet.exclude(**kwargs).
//
// Django:  Post.objects.exclude(published=False)
// djanGO:  Post.Objects.Exclude("published", false)
func (qs *QuerySet[T]) Exclude(field string, value interface{}) *QuerySet[T] {
	clone := qs.clone()
	clone.excludes = append(clone.excludes, [2]interface{}{field, value})
	return clone
}

// OrderBy mirrors Django's QuerySet.order_by(*fields).
// Prefix with "-" for descending — identical to Django.
//
// Django:  Post.objects.order_by("-created_at", "title")
// djanGO:  Post.Objects.OrderBy("-created_at", "title")
func (qs *QuerySet[T]) OrderBy(fields ...string) *QuerySet[T] {
	clone := qs.clone()
	clone.orderBy = append(clone.orderBy, fields...)
	return clone
}

// Limit mirrors Django's QuerySet slicing: Post.objects.all()[:10]
//
// Django:  Post.objects.all()[:10]
// djanGO:  Post.Objects.All().Limit(10)  — or  Post.Objects.Limit(10).All()
func (qs *QuerySet[T]) Limit(n int64) *QuerySet[T] {
	clone := qs.clone()
	clone.limit = n
	return clone
}

// Offset mirrors Django's QuerySet slicing: Post.objects.all()[10:20]
//
// Django:  Post.objects.all()[10:20]
// djanGO:  Post.Objects.Offset(10).Limit(10)
func (qs *QuerySet[T]) Offset(n int64) *QuerySet[T] {
	clone := qs.clone()
	clone.offset = n
	return clone
}

// All mirrors Django's QuerySet.all() — evaluates the query and returns all results.
//
// Django:  posts = Post.objects.all()
// djanGO:  posts, err := Post.Objects.All()
func (qs *QuerySet[T]) All() ([]T, error) {
	q := qs.build()
	var results []T
	_, err := q.All(&results)
	if err != nil {
		return nil, err
	}
	return results, nil
}

// Get mirrors Django's QuerySet.get(**kwargs) — returns exactly one result.
// Raises DoesNotExist if no match, MultipleObjectsReturned if more than one.
//
// Django:  post = Post.objects.get(slug="hello-world")
// djanGO:  post, err := Post.Objects.Get("slug", "hello-world")
func (qs *QuerySet[T]) Get(field string, value interface{}) (*T, error) {
	q := qs.build().Filter(field, value).Limit(2)
	var results []T
	n, err := q.All(&results)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, &DoesNotExist{Model: fmt.Sprintf("%T", qs.model)}
	}
	if n > 1 {
		return nil, &MultipleObjectsReturned{Model: fmt.Sprintf("%T", qs.model)}
	}
	return &results[0], nil
}

// First mirrors Django's QuerySet.first().
//
// Django:  post = Post.objects.order_by("id").first()
// djanGO:  post, err := Post.Objects.OrderBy("id").First()
func (qs *QuerySet[T]) First() (*T, error) {
	q := qs.build().Limit(1)
	var results []T
	n, err := q.All(&results)
	if err != nil || n == 0 {
		return nil, err
	}
	return &results[0], nil
}

// Count mirrors Django's QuerySet.count().
//
// Django:  n = Post.objects.filter(published=True).count()
// djanGO:  n, err := Post.Objects.Filter("published", true).Count()
func (qs *QuerySet[T]) Count() (int64, error) {
	return qs.build().Count()
}

// Exists mirrors Django's QuerySet.exists().
//
// Django:  Post.objects.filter(slug="x").exists()
// djanGO:  Post.Objects.Filter("slug", "x").Exists()
func (qs *QuerySet[T]) Exists() (bool, error) {
	n, err := qs.Count()
	return n > 0, err
}

// Delete mirrors Django's QuerySet.delete() — bulk delete all matching rows.
//
// Django:  Post.objects.filter(published=False).delete()
// djanGO:  Post.Objects.Filter("published", false).Delete()
func (qs *QuerySet[T]) Delete() (int64, error) {
	return qs.build().Delete()
}

// build applies all accumulated filters/excludes/ordering to Beego's QuerySeter.
func (qs *QuerySet[T]) build() orm.QuerySeter {
	q := qs.o.QueryTable(&qs.model)
	for _, f := range qs.filters {
		q = q.Filter(f[0].(string), f[1])
	}
	for _, f := range qs.excludes {
		q = q.Exclude(f[0].(string), f[1])
	}
	if len(qs.orderBy) > 0 {
		q = q.OrderBy(qs.orderBy...)
	}
	if qs.limit > 0 {
		q = q.Limit(qs.limit)
	}
	if qs.offset > 0 {
		q = q.Offset(qs.offset)
	}
	return q
}

func (qs *QuerySet[T]) clone() *QuerySet[T] {
	c := *qs
	c.filters = append([][2]interface{}{}, qs.filters...)
	c.excludes = append([][2]interface{}{}, qs.excludes...)
	c.orderBy = append([]string{}, qs.orderBy...)
	return &c
}

// DoesNotExist mirrors Django's Model.DoesNotExist exception.
type DoesNotExist struct{ Model string }

func (e *DoesNotExist) Error() string {
	return fmt.Sprintf("%s matching query does not exist.", e.Model)
}

// MultipleObjectsReturned mirrors Django's Model.MultipleObjectsReturned exception.
type MultipleObjectsReturned struct{ Model string }

func (e *MultipleObjectsReturned) Error() string {
	return fmt.Sprintf("get() returned more than one %s -- it returned multiple!", e.Model)
}
