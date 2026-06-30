package signals

// Built-in signals mirror Django's django.db.models.signals and
// django.core.signals.
//
// Django:
//
//	from django.db.models.signals import post_save, pre_save, post_delete, pre_delete
//	from django.core.signals import request_started, request_finished
//
// Usage:
//
//	signals.PostSave.Connect(func(sender string, kwargs signals.Kwargs) {
//	    instance := kwargs["instance"]
//	    created := kwargs["created"].(bool)
//	    _ = instance
//	    _ = created
//	}, signals.WithSender("auth.User"))

// Model signals — mirrors django.db.models.signals.

// PostSave is fired after a model is saved — mirrors post_save.
//
// Django:
//
//	from django.db.models.signals import post_save
//
//	@receiver(post_save, sender=User)
//	def on_user_save(sender, instance, created, **kwargs): ...
//
// Kwargs: "instance" (the model), "created" (bool: True on INSERT, False on UPDATE)
var PostSave = New("post_save")

// PreSave is fired before a model is saved — mirrors pre_save.
//
// Kwargs: "instance" (the model)
var PreSave = New("pre_save")

// PostDelete is fired after a model is deleted — mirrors post_delete.
//
// Kwargs: "instance" (the model)
var PostDelete = New("post_delete")

// PreDelete is fired before a model is deleted — mirrors pre_delete.
//
// Kwargs: "instance" (the model)
var PreDelete = New("pre_delete")

// PostMigrate is fired after migrate completes — mirrors post_migrate.
//
// Django:
//
//	from django.db.models.signals import post_migrate
//
//	@receiver(post_migrate)
//	def create_permissions(sender, **kwargs): ...
//
// Kwargs: "app_label" (string), "verbosity" (int)
var PostMigrate = New("post_migrate")

// PreMigrate is fired before migrate starts — mirrors pre_migrate.
var PreMigrate = New("pre_migrate")

// Request signals — mirrors django.core.signals.

// RequestStarted is fired at the start of each HTTP request.
var RequestStarted = New("request_started")

// RequestFinished is fired at the end of each HTTP request.
var RequestFinished = New("request_finished")

// GotRequestException is fired when an exception occurs during request handling.
//
// Kwargs: "request" (*http.Request), "exception" (error)
var GotRequestException = New("got_request_exception")
