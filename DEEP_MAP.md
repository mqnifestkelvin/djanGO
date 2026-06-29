# djanGO — Deep Codebase Map

This document is a full itemized map of the existing Beego codebase (our foundation)
and Django's source (our reference). It drives what we build, what we port, and what we skip.

---

## PROJECT STATS

- **Total Go files**: 363
- **Main packages**: ORM (82 files), Web Server (114 files), Core (105 files), Task (4 files)
- **Databases supported**: MySQL, PostgreSQL, SQLite, Oracle, TiDB
- **Django reference**: `django-reference/` — full Django source to port from

---

## PACKAGE 1: ORM (`client/orm/`) — 82 files

### What exists
| File | Purpose |
|---|---|
| `orm.go` | Core ORM engine — CRUD, raw SQL, query builder init |
| `db.go` | Driver base layer, field collection, type conversions |
| `types.go` | Interfaces: TableNameI, Driver, TxBeginner, TxCommitter |
| `db_alias.go` | DB alias cache, driver registry, connection pooling |
| `orm_queryset.go` | QuerySet: Filter, Exclude, Limit, Offset, OrderBy, GroupBy, Values |
| `orm_querym2m.go` | ManyToMany handler: Add, Remove, Clear, Count, Exist |
| `orm_raw.go` | Raw SQL executor: Prepare, Exec, Values, ValuesList |
| `orm_conds.go` | Condition operators: exact, contains, gt, gte, lt, in, between, isnull |
| `models_boot.go` | Model registration and bootstrap |
| `models_fields.go` | Field type constants: CharField, IntegerField, DateTimeField, JSONField, etc. |
| `migration/migration.go` | Migration struct with Up/Down versioning |
| `migration/ddl.go` | DDL operations: Column, Index, Unique, Foreign, RenameColumn |
| `db_mysql.go` | MySQL driver (CREATE TABLE, ALTER TABLE, constraints) |
| `db_postgres.go` | PostgreSQL driver (SERIAL, UUID, Array types) |
| `db_sqlite.go` | SQLite driver |
| `db_oracle.go` | Oracle driver (sequences, NUMBER type) |
| `db_tidb.go` | TiDB distributed database support |
| `hints/db_hints.go` | Query hints: ForceIndex, UseIndex, IgnoreIndex |
| `mock/` | Mock ORM implementations for testing |

### Field types available
- BooleanField, CharField, TextField
- TimeField, DateField, DateTimeField
- IntegerField, BigIntegerField, SmallIntegerField (positive/negative variants)
- FloatField, DecimalField, JSONField, JSONBField
- ForeignKey, OneToOne, ManyToMany

### Query operators available
- exact, iexact, strictexact
- contains, icontains, startswith, endswith (case variants)
- gt, gte, lt, lte, eq, ne
- in, between, isnull

### What's MISSING vs Django ORM
| Missing Feature | Django Source to Port |
|---|---|
| `aggregate()` — SUM, COUNT, AVG, MIN, MAX | `django/db/models/aggregates.py` |
| `annotate()` — per-object aggregations | `django/db/models/query.py` |
| `values()` / `values_list()` returns | `django/db/models/query.py` |
| `select_related()` JOIN-based eager loading | `django/db/models/query.py` |
| `prefetch_related()` separate query prefetch | `django/db/models/query.py` |
| `Q` objects — complex OR/AND expressions | `django/db/models/q.py` |
| `F` expressions — field-to-field comparisons | `django/db/models/expressions.py` |
| Custom model managers | `django/db/models/manager.py` |
| Model-level `clean()` validation | `django/db/models/base.py` |
| `get_or_create()` / `update_or_create()` | `django/db/models/query.py` |
| Abstract base models | `django/db/models/base.py` |
| Model inheritance (multi-table) | `django/db/models/base.py` |
| Window functions | `django/db/models/functions/window.py` |
| JSON field querying | `django/db/models/fields/json.py` |

---

## PACKAGE 2: WEB SERVER (`server/web/`) — 114 files

### What exists
| File | Purpose |
|---|---|
| `server.go` | HttpServer — HTTP/HTTPS/FCGI, TLS, graceful shutdown |
| `router.go` | ControllerRegister, routing tree, RESTful, URL pattern matching |
| `controller.go` | Base Controller — Init, Prepare, Get/Post/Put/Delete, Finish, Render |
| `config.go` | AppName, RunMode, Listen, template config, session config, XSRF |
| `filter.go` | FilterChain, FilterRouter, FilterFunc (middleware pipeline) |
| `context/context.go` | Context wrapping Request/Response with Input/Output |
| `context/input.go` | BeegoInput — request parsing, cookies, session access |
| `context/output.go` | BeegoOutput — JSON/XML/HTML response, redirect, cookies |
| `context/form.go` | Form parsing with struct binding |
| `template.go` | Template loading, compilation, caching, auto-reload in dev |
| `staticfile.go` | Static file serving with compression, range requests |
| `session/session.go` | Session Manager, Store, Provider interfaces |
| `session/sess_cookie.go` | Cookie-based sessions |
| `session/sess_file.go` | File-based sessions |
| `session/sess_mem.go` | In-memory sessions |
| `session/redis/` | Redis session backend |
| `session/mysql/` | MySQL session backend |
| `session/postgres/` | PostgreSQL session backend |
| `admin.go` | Admin console — QPS monitoring, task management |
| `flash.go` | Flash messages for transient cross-request data |
| `error.go` | Error handling, custom error pages |
| `namespace.go` | URL namespace support |
| `captcha/` | CAPTCHA image generation |
| `swagger/` | Swagger/OpenAPI doc support |
| `pagination/` | Pagination helper for controllers |

### Middleware execution order
```
Request → BeforeStatic → BeforeRouter → Route Match →
BeforeExec → Controller.Prepare → Action → Controller.Finish →
AfterExec → FinishRouter → Response
```

### Filter types available
| Filter | Location |
|---|---|
| Basic HTTP auth | `filter/auth/basic.go` |
| API key auth | `filter/apiauth/apiauth.go` |
| Authorization (Casbin RBAC) | `filter/authz/authz.go` |
| CORS | `filter/cors/cors.go` |
| Rate limiting (token bucket) | `filter/ratelimit/` |
| Session middleware | `filter/session/filter.go` |
| OpenTracing | `filter/opentracing/filter.go` |
| Prometheus metrics | `filter/prometheus/filter.go` |
| Graceful shutdown | `filter/grace/` |

### What's MISSING vs Django
| Missing Feature | Django Source to Port |
|---|---|
| Built-in User model | `django/contrib/auth/models.py` |
| Group model | `django/contrib/auth/models.py` |
| Permission model | `django/contrib/auth/models.py` |
| `@login_required` decorator | `django/contrib/auth/decorators.py` |
| `@permission_required` decorator | `django/contrib/auth/decorators.py` |
| Declarative Form classes | `django/forms/forms.py` |
| ModelForm | `django/forms/models.py` |
| Form field types | `django/forms/fields.py` |
| Form widgets | `django/forms/widgets.py` |
| Django-style `urls.py` per app | `django/urls/conf.py` |
| `path()` and `re_path()` | `django/urls/conf.py` |
| `include()` for app URLs | `django/urls/conf.py` |
| `reverse()` — named URL lookup | `django/urls/resolvers.py` |
| URL namespaces | `django/urls/resolvers.py` |
| Security middleware (clickjacking, XSS headers) | `django/middleware/security.py` |
| GZip middleware | `django/middleware/gzip.py` |
| Locale/language middleware | `django/middleware/locale.py` |
| CSRF middleware (global) | `django/middleware/csrf.py` |
| `process_exception` middleware hook | `django/core/handlers/base.py` |
| ModelAdmin customization | `django/contrib/admin/options.py` |
| Inline admin | `django/contrib/admin/options.py` |
| Admin actions (bulk ops) | `django/contrib/admin/actions.py` |
| Admin audit log | `django/contrib/admin/models.py` |
| Auto-register models to admin | `django/contrib/admin/sites.py` |
| Full test client | `django/test/client.py` |
| TestCase with DB rollback | `django/test/testcases.py` |
| Response assertions | `django/test/testcases.py` |

---

## PACKAGE 3: CORE (`core/`) — 105 files

### What exists
| Package | Files | Purpose |
|---|---|---|
| `core/logs/` | 22 files | Logging — console, file, Slack, SMTP, Elasticsearch, Alibaba Cloud |
| `core/config/` | 7 files | Config — JSON, YAML, TOML, XML, ENV, etcd |
| `core/validation/` | 1 file | Validators — Required, MaxSize, Range, Regex, Email, IP, URL |
| `core/utils/` | 3 files | SafeMap, pagination, caller/stack trace |
| `core/berror/` | 3 files | Custom error types with stack traces |
| `core/admin/` | 3 files | Health checks, profiling, command interface |
| `core/bean/` | 1 file | Simple service locator / dependency registration |

### What's MISSING vs Django
| Missing Feature | Django Source to Port |
|---|---|
| Signal/event system | `django/dispatch/dispatcher.py` |
| `pre_save` / `post_save` signals | `django/db/models/signals.py` |
| `pre_delete` / `post_delete` signals | `django/db/models/signals.py` |
| App registry | `django/apps/registry.py` |
| INSTALLED_APPS system | `django/apps/config.py` |
| Management command framework | `django/core/management/__init__.py` |
| `startproject` scaffolder | `django/core/management/commands/startproject.py` |
| `startapp` scaffolder | `django/core/management/commands/startapp.py` |
| `createsuperuser` command | `django/contrib/auth/management/commands/` |
| `makemigrations` command | `django/core/management/commands/makemigrations.py` |
| `migrate` command | `django/core/management/commands/migrate.py` |
| `loaddata` / `dumpdata` | `django/core/management/commands/loaddata.py` |
| `shell` command | `django/core/management/commands/shell.py` |
| Settings inheritance (base/dev/prod) | `django/conf/__init__.py` |
| Secret key management | `django/conf/global_settings.py` |
| Fixture system (JSON/YAML) | `django/core/serializers/` |

---

## PACKAGE 4: TASK (`task/`) — 4 files

### What exists
| File | Purpose |
|---|---|
| `task.go` | Task struct, Schedule (cron-style with second precision), error tracking |
| `govenor_command.go` | Task governor/manager command interface |

### Cron expression support
- Second, Minute, Hour, Day, Month, Week
- `*/5` interval syntax

---

## PACKAGE 5: CACHE (`client/cache/`) — 16 files

### What exists
| File | Purpose |
|---|---|
| `cache.go` | Cache interface — Set, Get, Delete, ClearAll, StartGC |
| `memory.go` | In-memory cache with TTL and GC |
| `file.go` | File-based cache |
| `write_through.go` | Write-through cache pattern |
| `write_delete.go` | Delete propagation pattern |
| `read_through.go` | Read-through (populate on miss) |
| `bloom_filter_cache.go` | Bloom filter to prevent cache stampede |
| `singleflight.go` | Deduplication for concurrent cache misses |
| `redis/redis.go` | Redis backend |
| `memcache/memcache.go` | Memcache backend |
| `ssdb/ssdb.go` | SSDB backend |

---

## DJANGO REFERENCE MAP

Django source is at `django-reference/django/`. Key modules to port:

### Auth System → `django-reference/django/contrib/auth/`
```
models.py          → User, Group, Permission models
backends.py        → Authentication backends
decorators.py      → @login_required, @permission_required
forms.py           → AuthenticationForm, UserCreationForm, PasswordChangeForm
hashers.py         → Password hashing (bcrypt, argon2, pbkdf2)
middleware.py      → AuthenticationMiddleware
signals.py         → user_logged_in, user_logged_out, user_login_failed
validators.py      → Password validators
views.py           → login, logout, password_change, password_reset views
```

### Forms System → `django-reference/django/forms/`
```
forms.py           → BaseForm, Form — declarative field definitions
models.py          → ModelForm — auto-generate form from model
fields.py          → CharField, EmailField, IntegerField, ChoiceField, FileField, etc.
widgets.py         → TextInput, Select, Checkbox, FileInput, etc.
validators.py      → Field-level validators
boundfield.py      → BoundField — field + value + errors
utils.py           → ErrorList, ErrorDict
```

### Admin System → `django-reference/django/contrib/admin/`
```
sites.py           → AdminSite, auto-registration with admin.register()
options.py         → ModelAdmin, InlineModelAdmin — all customization hooks
actions.py         → Bulk action system
filters.py         → List filters (SimpleListFilter, RelatedFieldListFilter)
models.py          → LogEntry — audit log model
views/             → CRUD views for admin
templates/admin/   → Admin HTML templates
```

### URL System → `django-reference/django/urls/`
```
conf.py            → path(), re_path(), include()
resolvers.py       → URLResolver, reverse(), resolve()
converters.py      → Path converters (int, str, slug, uuid, path)
```

### Signals → `django-reference/django/dispatch/`
```
dispatcher.py      → Signal class, receiver registration, send()
```

### Management Commands → `django-reference/django/core/management/`
```
__init__.py        → call_command(), execute_from_command_line()
base.py            → BaseCommand, CommandError
commands/          → All built-in commands (migrate, makemigrations, startproject, etc.)
```

### Fixtures → `django-reference/django/core/serializers/`
```
__init__.py        → serialize(), deserialize()
json.py            → JSON serializer/deserializer
python.py          → Python object serializer
base.py            → Base serializer classes
```

### App Registry → `django-reference/django/apps/`
```
registry.py        → Apps registry, app lookup, model registry
config.py          → AppConfig — per-app configuration
```

---

## BUILD PRIORITY ORDER

Based on what unlocks the most functionality for developers:

| Priority | Feature | Django Source | Unlocks |
|---|---|---|---|
| 1 | `djangocli` CLI + project scaffold | `management/` | Everything else |
| 2 | Django-style URL conf (`path`, `include`, `reverse`) | `urls/` | App routing |
| 3 | App system + INSTALLED_APPS | `apps/` | Modular apps |
| 4 | Signal system | `dispatch/` | ORM hooks |
| 5 | User/Group/Permission models | `contrib/auth/models.py` | Auth |
| 6 | Auth middleware + decorators | `contrib/auth/` | Protected views |
| 7 | Form classes + ModelForm | `forms/` | Data input |
| 8 | Advanced ORM (Q, F, aggregate, annotate) | `db/models/` | Complex queries |
| 9 | Enhanced admin (ModelAdmin, inlines) | `contrib/admin/` | Admin panel |
| 10 | Fixtures system | `core/serializers/` | Data loading |
| 11 | Testing framework | `test/` | Test tooling |
| 12 | Middleware improvements | `middleware/` | Security/i18n |
