package models

import (
	"github.com/mqnifestkelvin/djanGO/client/orm"
)

// Manager mirrors Django's Manager class — the entry point for database queries.
// Attached to every Model as the `Objects` field (Django uses lowercase `objects`).
//
// Django:
//
//	Post.objects.all()
//	Post.objects.filter(published=True)
//	Post.objects.get(slug="hello")
//	Post.objects.create(title="Hello", slug="hello")
//
// djanGO:
//
//	Post.Objects.All()
//	Post.Objects.Filter("published", true)
//	Post.Objects.Get("slug", "hello")
//	Post.Objects.Create(&Post{Title: "Hello", Slug: "hello"})
type Manager[T any] struct {
	model func() T // factory so each call gets a fresh zero value
}

// NewManager creates a Manager for a model type.
// Call this when defining the Objects field on your model.
func NewManager[T any](factory func() T) Manager[T] {
	return Manager[T]{model: factory}
}

func (m *Manager[T]) ormer() orm.Ormer {
	return orm.NewOrm()
}

// All mirrors Django's Manager.all() — returns a QuerySet of all objects.
//
// Django:  Post.objects.all()
// djanGO:  Post.Objects.All()
func (m *Manager[T]) All() ([]T, error) {
	return m.QuerySet().All()
}

// Filter mirrors Django's Manager.filter(**kwargs).
//
// Django:  Post.objects.filter(published=True)
// djanGO:  Post.Objects.Filter("published", true)
func (m *Manager[T]) Filter(field string, value interface{}) *QuerySet[T] {
	return m.QuerySet().Filter(field, value)
}

// Exclude mirrors Django's Manager.exclude(**kwargs).
//
// Django:  Post.objects.exclude(published=False)
// djanGO:  Post.Objects.Exclude("published", false)
func (m *Manager[T]) Exclude(field string, value interface{}) *QuerySet[T] {
	return m.QuerySet().Exclude(field, value)
}

// Get mirrors Django's Manager.get(**kwargs).
//
// Django:  post = Post.objects.get(slug="hello")
// djanGO:  post, err := Post.Objects.Get("slug", "hello")
func (m *Manager[T]) Get(field string, value interface{}) (*T, error) {
	return m.QuerySet().Get(field, value)
}

// Create mirrors Django's Manager.create(**kwargs) — inserts and returns the object.
//
// Django:  post = Post.objects.create(title="Hello", slug="hello")
// djanGO:  err := Post.Objects.Create(&post)
func (m *Manager[T]) Create(obj interface{}) error {
	_, err := m.ormer().Insert(obj)
	return err
}

// GetOrCreate mirrors Django's Manager.get_or_create(**kwargs).
//
// Django:  post, created = Post.objects.get_or_create(slug="hello", defaults={...})
// djanGO:  created, err := Post.Objects.GetOrCreate(&post, "slug")
func (m *Manager[T]) GetOrCreate(obj interface{}, col string) (created bool, err error) {
	created, _, err = m.ormer().ReadOrCreate(obj, col)
	return
}

// OrderBy mirrors Django's Manager.order_by(*fields).
//
// Django:  Post.objects.order_by("-created_at")
// djanGO:  Post.Objects.OrderBy("-created_at")
func (m *Manager[T]) OrderBy(fields ...string) *QuerySet[T] {
	return m.QuerySet().OrderBy(fields...)
}

// Count mirrors Django's Manager.count().
//
// Django:  Post.objects.count()
// djanGO:  Post.Objects.Count()
func (m *Manager[T]) Count() (int64, error) {
	return m.QuerySet().Count()
}

// QuerySet returns a base QuerySet for this manager's model —
// mirrors Django's Manager.get_queryset().
func (m *Manager[T]) QuerySet() *QuerySet[T] {
	return newQuerySet(m.ormer(), m.model())
}
