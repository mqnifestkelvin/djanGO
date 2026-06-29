# djanGO

Django's batteries-included philosophy, powered by Go's speed.

djanGO is a fork of [Beego](https://github.com/beego/beego) that brings Django's full developer experience to Go — Models, Views, URLs, admin panel, migrations, management commands, and more.

## Quick Start

### 1. Install djanGO-admin

```bash
sudo go install github.com/beego/beego/v2/cmd/djang0@latest
sudo mv $(go env GOPATH)/bin/djang0 /usr/local/bin/djanGO-admin
```

Or build from source:

```bash
git clone https://github.com/mqnifestkelvin/beego
cd beego
sudo go build -o /usr/local/bin/djanGO-admin ./cmd/djang0/
```

### 2. Create a project

```bash
djanGO-admin startproject mysite
cd mysite
```

This generates:

```
mysite/
├── manage.go              ← management commands entry point
├── go.mod
├── urls.go                ← root URL configuration
├── settings/
│   ├── base.go            ← shared settings (AppName, SecretKey, DB, etc.)
│   ├── development.go     ← dev overrides (Debug=true)
│   └── production.go      ← production overrides
├── apps/                  ← your djanGO apps live here
├── templates/             ← HTML templates
├── static/                ← CSS, JS, images
└── media/                 ← user uploaded files
```

### 3. Create an app

```bash
djanGO-admin startapp blog
```

This generates:

```
blog/
├── apps.go        ← app config + route registration
├── models.go      ← data models (ORM)
├── views.go       ← controllers / view handlers
├── urls.go        ← app URL patterns
├── admin.go       ← admin registrations
├── forms.go       ← form definitions
├── migrations/    ← database migrations
├── templates/
│   └── blog/      ← app-specific templates
└── tests/
    └── tests.go
```

### 4. Add the app to your project

In `settings/base.go`, add the app name to `InstalledApps`:

```go
InstalledApps: []string{
    "blog",
},
```

Then import it in `manage.go`:

```go
_ "github.com/you/mysite/apps/blog"
```

### 5. Run the development server

```bash
go run manage.go runserver
```

Visit [http://127.0.0.1:8080](http://127.0.0.1:8080)

---

## How it mirrors Django

| Django | djanGO |
|---|---|
| `django-admin startproject mysite` | `djanGO-admin startproject mysite` |
| `python manage.py startapp blog` | `djanGO-admin startapp blog` |
| `python manage.py runserver` | `go run manage.go runserver` |
| `INSTALLED_APPS` in `settings.py` | `InstalledApps` in `settings/base.go` |
| `models.py` | `models.go` |
| `views.py` | `views.go` |
| `urls.py` | `urls.go` |
| `admin.py` | `admin.go` |
| `forms.py` | `forms.go` |
| `AppConfig` in `apps.py` | `apps.AppConfig` in `apps.go` |

## Features

* RESTful support
* [MVC architecture](https://github.com/beego/beedoc/tree/master/en-US/mvc)
* Modularity
* [Auto API documents](https://github.com/beego/beedoc/blob/master/en-US/advantage/docs.md)
* [Annotation router](https://github.com/beego/beedoc/blob/master/en-US/mvc/controller/router.md)
* [Namespace](https://github.com/beego/beedoc/blob/master/en-US/mvc/controller/router.md#namespace)
* [Powerful development tools](https://github.com/beego/bee)
* Full stack for Web & API

## Modules

* [orm](https://github.com/beego/beedoc/tree/master/en-US/mvc/model)
* [session](https://github.com/beego/beedoc/blob/master/en-US/module/session.md)
* [logs](https://github.com/beego/beedoc/blob/master/en-US/module/logs.md)
* [config](https://github.com/beego/beedoc/blob/master/en-US/module/config.md)
* [cache](https://github.com/beego/beedoc/blob/master/en-US/module/cache.md)
* [context](https://github.com/beego/beedoc/blob/master/en-US/module/context.md)
* [admin](https://github.com/beego/beedoc/blob/master/en-US/module/admin.md)
* [httplib](https://github.com/beego/beedoc/blob/master/en-US/module/httplib.md)
* [task](https://github.com/beego/beedoc/blob/master/en-US/module/task.md)
* [i18n](https://github.com/beego/beedoc/blob/master/en-US/module/i18n.md)

## Community

* Welcome to join us in Slack: [https://beego.slack.com invite](https://join.slack.com/t/beego/shared_invite/zt-fqlfjaxs-_CRmiITCSbEqQG9NeBqXKA),
* QQ Group ID:523992905
* [Contribution Guide](https://github.com/beego/beedoc/blob/master/en-US/intro/contributing.md).

## License

beego source code is licensed under the Apache Licence, Version 2.0
([https://www.apache.org/licenses/LICENSE-2.0.html](https://www.apache.org/licenses/LICENSE-2.0.html)).
