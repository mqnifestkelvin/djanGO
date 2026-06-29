# djanGO — Project Goals

djanGO is a fork of Beego that aims to be a 1:1 Go equivalent of Django.
The goal is to bring Django's full feature set to Go — keeping Go's performance
and concurrency advantages while matching Django's developer experience.

---

## Vision

> "djanGO — Django's batteries-included philosophy, powered by Go's speed."

- MVC pattern (Models, Views, URLs) exactly like Django
- Auto-generated admin panel
- Built-in user authentication & permissions
- Declarative forms with validation
- Signal/event system
- `manage.py`-style CLI (`djangocli`)
- Fixtures, migrations, management commands
- Django-like project structure out of the box

---

## What Beego Already Provides (Foundation)

- [x] ORM (models, relationships, migrations, raw SQL)
- [x] URL routing (pattern matching, namespaces, RESTful)
- [x] Controllers/Views (base controller, MVC)
- [x] Middleware/Filters (CORS, rate limiting, auth, tracing)
- [x] Template engine (with custom functions/tags)
- [x] Sessions (Redis, file, memcache, cookie backends)
- [x] Admin dashboard (basic, built-in)
- [x] Configuration system (YAML, TOML, JSON, ENV)
- [x] Caching (Redis, Memcache, SSDB)
- [x] Validation (built-in validators)
- [x] Background task scheduling
- [x] Static file serving
- [x] Logging (multiple outputs)
- [x] Pagination
- [x] CAPTCHA support
- [x] Flash messages

---

## What We Need to Build (DjanGO Additions)

### 1. User Authentication & Authorization System
- [ ] Built-in `User` model (username, email, password, is_active, is_staff, is_superuser)
- [ ] `Group` model
- [ ] `Permission` model
- [ ] Password hashing (bcrypt/argon2)
- [ ] Login / logout / password reset flows
- [ ] `@login_required` decorator equivalent
- [ ] `@permission_required` decorator equivalent
- [ ] Anonymous user support
- [ ] Remember me / session expiry

### 2. Declarative Form Classes
- [ ] Base `Form` struct with field definitions
- [ ] `ModelForm` — auto-generate form from model
- [ ] Field types: CharField, EmailField, IntegerField, BooleanField, ChoiceField, FileField, etc.
- [ ] Field-level validation
- [ ] Form-level validation (cross-field)
- [ ] Widget system (HTML rendering)
- [ ] Error collection and display
- [ ] CSRF protection built into forms

### 3. Signal / Event System
- [ ] `pre_save` / `post_save` signals
- [ ] `pre_delete` / `post_delete` signals
- [ ] `m2m_changed` signal (ManyToMany)
- [ ] Custom signal registration
- [ ] Signal receiver decorators
- [ ] Async signal support

### 4. CLI Command Framework (`djangocli`)
- [ ] `djangocli startproject <name>` — scaffold new project
- [ ] `djangocli startapp <name>` — scaffold new app
- [ ] `djangocli runserver` — start dev server with hot reload
- [ ] `djangocli makemigrations` — generate migration files
- [ ] `djangocli migrate` — apply migrations
- [ ] `djangocli createsuperuser` — create admin user
- [ ] `djangocli shell` — interactive Go REPL with project context
- [ ] `djangocli collectstatic` — gather static files
- [ ] `djangocli loaddata` — load fixtures
- [ ] `djangocli dumpdata` — export data to fixture
- [ ] `djangocli test` — run project tests
- [ ] Pluggable custom management commands

### 5. App System
- [ ] Reusable `app` packages with own models, urls, views, admin
- [ ] `INSTALLED_APPS` equivalent in config
- [ ] App registry (auto-discover models, signals, admin)
- [ ] App-level middleware registration
- [ ] App-level URL inclusion (`include()` equivalent)

### 6. Fixtures System
- [ ] Fixture format support (JSON, YAML)
- [ ] `loaddata` — load fixtures into DB
- [ ] `dumpdata` — export DB data to fixture
- [ ] Initial data fixtures on migration
- [ ] Natural keys support

### 7. Advanced ORM Features
- [ ] `aggregate()` — SUM, COUNT, AVG, MIN, MAX
- [ ] `annotate()` — per-object aggregations
- [ ] `values()` / `values_list()` — dict/tuple querysets
- [ ] `select_related()` — JOIN-based eager loading
- [ ] `prefetch_related()` — separate query prefetching
- [ ] `Q` objects — complex OR/AND query expressions
- [ ] `F` expressions — field-to-field comparisons
- [ ] Custom model managers
- [ ] Model-level `clean()` validation
- [ ] `get_or_create()` / `update_or_create()`
- [ ] Soft delete support
- [ ] Abstract base models

### 8. Enhanced Admin Panel
- [ ] Auto-register models to admin
- [ ] `ModelAdmin` customization (list_display, search_fields, filters)
- [ ] Inline admin (edit related models on same page)
- [ ] Admin actions (bulk operations)
- [ ] Admin permissions (per-model, per-user)
- [ ] Custom admin views
- [ ] Export to CSV/Excel from admin
- [ ] Admin audit log (who changed what)

### 9. URL Configuration (Django-style)
- [ ] Explicit `urls.go` per app (like `urls.py`)
- [ ] `path()` and `re_path()` equivalents
- [ ] `include()` for app URL inclusion
- [ ] Named URL patterns (`reverse()` equivalent)
- [ ] URL namespaces

### 10. Middleware Improvements
- [ ] `process_request` / `process_response` / `process_exception` hooks
- [ ] Security middleware (clickjacking, XSS, content-type sniffing)
- [ ] CSRF middleware (global, not just form-level)
- [ ] GZip middleware
- [ ] Locale/language middleware

### 11. Testing Framework
- [ ] Full-featured test client (simulate requests with sessions)
- [ ] `TestCase` base struct with DB transaction rollback
- [ ] Fixture loading in tests
- [ ] Response assertions (status, content, redirects)
- [ ] Mock request builder

### 12. Django-style Project Structure
- [ ] Scaffold generates this layout:
```
myproject/
├── manage.go          (djangocli entry point)
├── settings/
│   ├── base.go
│   ├── development.go
│   └── production.go
├── urls.go            (root URL conf)
├── apps/
│   └── myapp/
│       ├── models.go
│       ├── views.go
│       ├── urls.go
│       ├── admin.go
│       ├── forms.go
│       └── tests/
├── templates/
├── static/
└── migrations/
```

---

## Phase Plan

| Phase | Focus | Status |
|---|---|---|
| 1 | Fork cleanup + rebrand + project structure scaffold | 🔲 Not started |
| 2 | Django-style URL conf + `path()`/`include()` | 🔲 Not started |
| 3 | User auth system (User, Group, Permission models) | 🔲 Not started |
| 4 | Declarative form classes + ModelForm | 🔲 Not started |
| 5 | Signal/event system | 🔲 Not started |
| 6 | CLI command framework (`djangocli`) | 🔲 Not started |
| 7 | Advanced ORM features (Q, F, aggregate, annotate) | 🔲 Not started |
| 8 | Enhanced admin panel (ModelAdmin, inlines, actions) | 🔲 Not started |
| 9 | App system + INSTALLED_APPS | 🔲 Not started |
| 10 | Fixtures system | 🔲 Not started |
| 11 | Testing framework | 🔲 Not started |
| 12 | Middleware improvements | 🔲 Not started |
