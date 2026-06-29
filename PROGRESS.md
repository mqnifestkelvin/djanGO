# djanGO — Progress, Status & Test Plan

## What We Have Built

### 1. `djanGO-admin` CLI (`cmd/djang0/`)

The command-line tool that mirrors `django-admin`.

| Command | Status | Django equivalent |
|---|---|---|
| `djanGO-admin startproject <name>` | ✅ Working | `django-admin startproject <name>` |
| `djanGO-admin startapp <name>` | ✅ Working | `django-admin startapp <name>` |

**`startproject` generates:**
```
mysite/
├── manage.go               ← mirrors manage.py
├── go.mod
└── mysite/                 ← inner config package (mirrors Django's inner folder)
    ├── init.go             ← mirrors __init__.py
    ├── settings.go         ← mirrors settings.py (full boilerplate)
    ├── urls.go             ← mirrors urls.py
    ├── wsgi.go             ← mirrors wsgi.py
    └── asgi.go             ← mirrors asgi.py
```

**`startapp` generates:**
```
blog/
├── init.go                 ← mirrors __init__.py
├── apps.go                 ← mirrors apps.py (AppConfig registration)
├── admin.go                ← mirrors admin.py
├── models.go               ← mirrors models.py
├── views.go                ← mirrors views.py
├── tests.go                ← mirrors tests.py
└── migrations/
    └── init.go             ← mirrors migrations/__init__.py
```

---

### 2. Management Command Framework (`core/management/`)

The engine behind `go run manage.go <command>` — mirrors Django's `django.core.management`.

| File | Purpose | Status |
|---|---|---|
| `core/management/base.go` | `Command` interface, `BaseCommand` struct, `CommandError` | ✅ Done |
| `core/management/registry.go` | Global command registry, `Execute()`, typo suggestions | ✅ Done |
| `core/management/commands/runserver.go` | Built-in `runserver` command | ✅ Done |
| `core/management/commands/startapp.go` | Built-in `startapp` command via `manage.go` | ✅ Done |

**How it works:**
- Any app can register its own commands by calling `management.Register()` from `init()`
- `go run manage.go <command>` calls `management.Execute()` which finds and runs the command
- Unknown commands show a did-you-mean suggestion (like Django)

---

### 3. App Registry / INSTALLED_APPS System (`core/apps/`)

Mirrors Django's `django.apps` — makes `InstalledApps` actually do something.

| Feature | Status | Django equivalent |
|---|---|---|
| `AppConfig` struct | ✅ Done | `AppConfig` class |
| `Register()` / `MustRegister()` | ✅ Done | Auto-discovery via `INSTALLED_APPS` |
| `Setup(installedApps)` | ✅ Done | `django.setup()` |
| `GetAppConfigs()` | ✅ Done | `apps.get_app_configs()` |
| `GetAppConfig(name)` | ✅ Done | `apps.get_app_config(label)` |
| `IsInstalled(name)` | ✅ Done | `apps.is_installed(app_name)` |
| `Ready()` hook | ✅ Done | `AppConfig.ready()` |

---

### 4. Module Rename

| Item | Status |
|---|---|
| Module renamed from `github.com/beego/beego/v2` → `github.com/mqnifestkelvin/djanGO` | ✅ Done |
| All 189 Go files updated | ✅ Done |
| `go.mod` updated | ✅ Done |
| Pushed to GitHub as `github.com/mqnifestkelvin/djanGO` | ✅ Done |
| Tagged as `v0.1.0` | ✅ Done |
| Django source added as git submodule (`django-reference/`) | ✅ Done |

---

### 5. Inherited from Beego (Already Working)

These come from Beego and work out of the box:

| Feature | Package | Django equivalent |
|---|---|---|
| ORM (models, queries, relationships) | `client/orm/` | `django.db.models` |
| Migrations (up/down) | `client/orm/migration/` | `django.db.migrations` |
| URL routing | `server/web/router.go` | `django.urls` |
| Controllers/Views | `server/web/controller.go` | `django.views` |
| Middleware/Filters | `server/web/filter/` | `django.middleware` |
| Templates | `server/web/template.go` | `django.template` |
| Sessions (Redis, file, DB, cookie) | `server/web/session/` | `django.contrib.sessions` |
| Admin dashboard | `server/web/admin.go` | `django.contrib.admin` (basic) |
| Static file serving | `server/web/staticfile.go` | `django.contrib.staticfiles` |
| Caching (Redis, Memcache) | `client/cache/` | `django.core.cache` |
| Logging | `core/logs/` | `django.utils.log` |
| Config system | `core/config/` | `django.conf` |
| Validation | `core/validation/` | `django.core.validators` |
| Background tasks | `task/` | Celery (not built-in Django) |
| CORS, Rate limiting, Auth filters | `server/web/filter/` | Various Django middleware |

---

## What Still Needs to Be Built

Tracked in `DJANGO_GOALS.md`. Priority order:

| Priority | Feature | Django source to port |
|---|---|---|
| 1 | `makemigrations` command | `management/commands/makemigrations.py` |
| 2 | `migrate` command | `management/commands/migrate.py` |
| 3 | Django-style `urls.go` with `path()`, `include()`, `reverse()` | `django/urls/` |
| 4 | Signal system (`pre_save`, `post_save`, `pre_delete`, `post_delete`) | `django/dispatch/` |
| 5 | User / Group / Permission models | `django/contrib/auth/models.py` |
| 6 | Auth middleware + `@login_required` | `django/contrib/auth/` |
| 7 | Declarative Form classes + `ModelForm` | `django/forms/` |
| 8 | Advanced ORM (`Q`, `F`, `aggregate`, `annotate`) | `django/db/models/` |
| 9 | Enhanced admin (`ModelAdmin`, inlines, actions, audit log) | `django/contrib/admin/` |
| 10 | Fixtures (`loaddata`, `dumpdata`) | `django/core/serializers/` |
| 11 | Testing framework (test client, `TestCase` with DB rollback) | `django/test/` |
| 12 | Middleware improvements (security, GZip, locale, CSRF global) | `django/middleware/` |
| 13 | `createsuperuser` command | `django/contrib/auth/management/commands/` |
| 14 | `shell` command | `django/core/management/commands/shell.py` |

---

## Tests We Need to Write

### `cmd/djang0/` — CLI Tests

| Test | What to verify |
|---|---|
| `TestStartProjectCreatesFiles` | `startproject mysite` creates exactly the right files and no extras |
| `TestStartProjectInvalidName` | Rejects names like `123abc`, `my-site`, empty string |
| `TestStartProjectAlreadyExists` | Errors cleanly if directory already exists |
| `TestStartProjectSecretKeyIsUnique` | Two runs produce different secret keys |
| `TestStartProjectInnerFolder` | Inner config folder is named after the project |
| `TestStartAppCreatesFiles` | `startapp blog` creates exactly the right 7 files |
| `TestStartAppInvalidName` | Rejects invalid app names |
| `TestStartAppAlreadyExists` | Errors if app directory already exists |
| `TestStartAppMigrationsFolder` | `migrations/init.go` is created |

### `core/management/` — Command Framework Tests

| Test | What to verify |
|---|---|
| `TestRegisterCommand` | `Register()` adds command to registry |
| `TestExecuteUnknownCommand` | Unknown command exits with error and suggestion |
| `TestExecuteKnownCommand` | Known command runs successfully |
| `TestTypoSuggestion` | `runservr` suggests `runserver` |
| `TestCommandFlags` | Flags are parsed and passed to `Execute()` |
| `TestDuplicateRegister` | Registering same command name twice is handled |
| `TestAllCommands` | Returns sorted list of all registered commands |

### `core/apps/` — App Registry Tests

| Test | What to verify |
|---|---|
| `TestRegisterApp` | `Register()` adds app to registry |
| `TestRegisterDuplicateApp` | Returns error on duplicate app name |
| `TestMustRegisterPanics` | `MustRegister()` panics on duplicate |
| `TestGetAppConfig` | Returns correct config for registered app |
| `TestGetAppConfigNotFound` | Returns error for unknown app name |
| `TestIsInstalled` | Returns true for registered app, false otherwise |
| `TestSetup` | Calls `Ready()` on all apps in `InstalledApps` |
| `TestSetupUnregisteredApp` | Returns error if app in `InstalledApps` was never registered |
| `TestGetAppConfigs` | Returns all apps in registration order |
| `TestAppNames` | Returns correct list of app names |
| `TestReadyFnCalled` | `SetReady()` function is called during `Setup()` |

### Integration Tests

| Test | What to verify |
|---|---|
| `TestFullProjectWorkflow` | `startproject` → `startapp` → `go build` succeeds |
| `TestManageGoRunsStartapp` | `go run manage.go startapp blog` creates app |
| `TestManageGoRunsRunserver` | `go run manage.go runserver` starts server |

---

## How to Run Existing Tests

```bash
# Run all tests in the framework
cd /home/mannie/Desktop/Projects/djanGO
go test ./core/... -v

# Run a specific package
go test ./core/apps/... -v
go test ./core/management/... -v

# Run with coverage
go test ./core/... -cover
```

## How to Install djanGO-admin

```bash
# Build from source
cd /home/mannie/Desktop/Projects/djanGO
sudo go build -o /usr/local/bin/djanGO-admin ./cmd/djang0/

# Or go install (once v0.1.0 is indexed by pkg.go.dev)
go install github.com/mqnifestkelvin/djanGO/cmd/djang0@v0.1.0
```
