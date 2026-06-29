package auth

import (
	"time"

	"github.com/mqnifestkelvin/djanGO/core/models"
	"github.com/mqnifestkelvin/djanGO/client/orm"
)

// User mirrors Django's django.contrib.auth.models.User.
//
// Django fields:
//   username, email, password, first_name, last_name
//   is_staff, is_active, is_superuser
//   last_login, date_joined
//
// Django:
//
//	from django.contrib.auth.models import User
//	u = User.objects.create_user("alice", "alice@example.com", "password")
//	u.check_password("password")  # True
type User struct {
	models.Model
	Id          int       `orm:"auto;pk"`
	Username    string    `orm:"size(150);unique"`
	Email       string    `orm:"size(254)"`
	Password    string    `orm:"size(128)"`
	FirstName   string    `orm:"size(150)"`
	LastName    string    `orm:"size(150)"`
	IsStaff     bool      `orm:"default(false)"`
	IsActive    bool      `orm:"default(true)"`
	IsSuperuser bool      `orm:"default(false)"`
	LastLogin   time.Time `orm:"type(datetime);null"`
	DateJoined  time.Time `orm:"type(datetime);auto_now_add"`
}

func (u *User) TableName() string { return "auth_user" }

// SetPassword hashes and stores the password — mirrors Django's user.set_password().
//
// Django:
//
//	user.set_password("newpassword")
//	user.save()
func (u *User) SetPassword(raw string) error {
	hashed, err := MakePassword(raw)
	if err != nil {
		return err
	}
	u.Password = hashed
	return nil
}

// CheckPassword verifies a plain-text password against the stored hash —
// mirrors Django's user.check_password().
//
// Django:
//
//	user.check_password("mypassword")  # True/False
func (u *User) CheckPassword(raw string) bool {
	return CheckPassword(raw, u.Password)
}

// GetFullName returns "FirstName LastName" — mirrors Django's user.get_full_name().
func (u *User) GetFullName() string {
	if u.FirstName == "" && u.LastName == "" {
		return u.Username
	}
	return u.FirstName + " " + u.LastName
}

// IsAuthenticated always returns true for a real User instance —
// mirrors Django's AbstractBaseUser.is_authenticated (always True for real users).
func (u *User) IsAuthenticated() bool { return true }

// Objects is the default manager — mirrors Django's User.objects.
//
// Django:
//
//	User.objects.all()
//	User.objects.get(username="alice")
var Objects = models.NewManager(func() User { return User{} })

// CreateUser creates and saves a user with a hashed password —
// mirrors Django's User.objects.create_user().
//
// Django:
//
//	User.objects.create_user("alice", "alice@example.com", "password")
func CreateUser(username, email, password string) (*User, error) {
	u := &User{
		Username: username,
		Email:    email,
		IsActive: true,
	}
	if err := u.SetPassword(password); err != nil {
		return nil, err
	}
	o := orm.NewOrm()
	if _, err := o.Insert(u); err != nil {
		return nil, err
	}
	return u, nil
}

// CreateSuperuser creates a superuser — mirrors Django's createsuperuser command.
func CreateSuperuser(username, email, password string) (*User, error) {
	u, err := CreateUser(username, email, password)
	if err != nil {
		return nil, err
	}
	u.IsStaff = true
	u.IsSuperuser = true
	o := orm.NewOrm()
	if _, err := o.Update(u); err != nil {
		return nil, err
	}
	return u, nil
}

func init() {
	orm.RegisterModel(&User{})
}
