// Package signals mirrors Django's django.dispatch signal system.
//
// Django reference: django/dispatch/dispatcher.py
//
// Django's signal system allows decoupled components to get notified when
// certain events occur. Any code can send a signal; any code can listen for it.
//
// Django:
//
//	from django.db.models.signals import post_save
//	from django.dispatch import receiver
//
//	@receiver(post_save, sender=User)
//	def user_saved(sender, instance, created, **kwargs):
//	    if created:
//	        Profile.objects.create(user=instance)
//
// djanGO:
//
//	import "github.com/mqnifestkelvin/djanGO/core/signals"
//
//	// Define a signal
//	var PostSave = signals.New("post_save")
//
//	// Connect a receiver
//	PostSave.Connect(func(sender string, kwargs signals.Kwargs) {
//	    instance := kwargs["instance"]
//	    created := kwargs["created"].(bool)
//	    ...
//	})
//
//	// Send the signal
//	PostSave.Send("auth.User", signals.Kwargs{"instance": user, "created": true})
package signals

import "sync"

// Kwargs mirrors Django's **kwargs — arbitrary keyword arguments passed to receivers.
//
// Django:
//
//	def my_handler(sender, **kwargs):
//	    instance = kwargs["instance"]
type Kwargs map[string]interface{}

// Receiver is a function that handles a signal.
// sender is the dotted name of the sender (e.g. "auth.User", "blog.Post").
//
// Django:
//
//	def handler(sender, **kwargs): ...
type Receiver func(sender string, kwargs Kwargs)

// Signal mirrors Django's django.dispatch.Signal.
//
// Django:
//
//	my_signal = Signal()
//	my_signal.connect(handler)
//	my_signal.send(sender=MyModel, instance=obj)
type Signal struct {
	name      string
	mu        sync.RWMutex
	receivers []receiver
}

type receiver struct {
	uid  string
	fn   Receiver
	sender string // "" means any sender
}

// New creates a new Signal — mirrors Signal() constructor.
//
// Django:
//
//	from django.dispatch import Signal
//	my_signal = Signal()
func New(name string) *Signal {
	return &Signal{name: name}
}

// Connect registers a receiver function — mirrors Signal.connect().
//
// Django:
//
//	post_save.connect(my_handler, sender=User)
//	post_save.connect(my_handler)  # all senders
//
// uid is an optional unique ID to prevent duplicate registration (mirrors dispatch_uid).
// sender filters to a specific sender name; "" means receive from any sender.
func (s *Signal) Connect(fn Receiver, opts ...ConnectOption) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := receiver{fn: fn}
	for _, o := range opts {
		o(&r)
	}
	// If uid set, deduplicate.
	if r.uid != "" {
		for _, existing := range s.receivers {
			if existing.uid == r.uid {
				return
			}
		}
	}
	s.receivers = append(s.receivers, r)
}

// Disconnect removes a receiver by uid — mirrors Signal.disconnect().
//
// Django:
//
//	post_save.disconnect(dispatch_uid="my_handler")
func (s *Signal) Disconnect(uid string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	filtered := s.receivers[:0]
	for _, r := range s.receivers {
		if r.uid != uid {
			filtered = append(filtered, r)
		}
	}
	s.receivers = filtered
}

// Send fires the signal to all connected receivers — mirrors Signal.send().
// Receivers are called synchronously in registration order.
//
// Django:
//
//	post_save.send(sender=User, instance=user, created=True)
//	# → calls all connected receivers with sender="auth.User", kwargs={...}
func (s *Signal) Send(sender string, kwargs Kwargs) {
	s.mu.RLock()
	rcvs := make([]receiver, len(s.receivers))
	copy(rcvs, s.receivers)
	s.mu.RUnlock()

	for _, r := range rcvs {
		if r.sender == "" || r.sender == sender {
			r.fn(sender, kwargs)
		}
	}
}

// ConnectOption configures a Connect() call.
type ConnectOption func(*receiver)

// WithUID sets a unique ID for the receiver to prevent duplicate registration —
// mirrors Signal.connect(dispatch_uid="...").
func WithUID(uid string) ConnectOption {
	return func(r *receiver) { r.uid = uid }
}

// WithSender filters the receiver to only fire for a specific sender —
// mirrors Signal.connect(sender=MyModel).
func WithSender(sender string) ConnectOption {
	return func(r *receiver) { r.sender = sender }
}
