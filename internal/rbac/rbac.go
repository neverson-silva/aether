package rbac

import (
	"errors"

	"aether/internal/domain"
)

var ErrForbidden = errors.New("forbidden")

var perms = map[domain.Role][]string{
	domain.RoleOwner: {
		"app.read", "app.write", "app.deploy", "app.env", "app.domain",
		"org.read", "org.write", "member.read", "member.write",
		"backup.read", "backup.write", "key.write", "cert.write",
	},
	domain.RoleAdmin: {
		"app.read", "app.write", "app.deploy", "app.env", "app.domain",
		"org.read", "member.read", "member.write",
		"backup.read", "backup.write", "key.write", "cert.write",
	},
	domain.RoleDeveloper: {
		"app.read", "app.write", "app.deploy", "app.env", "app.domain",
		"org.read", "backup.read",
	},
	domain.RoleMember: {
		"app.read", "app.write", "app.deploy", "app.env", "app.domain",
		"org.read", "backup.read",
	},
	domain.RoleViewer: {
		"app.read", "org.read", "backup.read",
	},
}

func Can(role domain.Role, perm string) bool {
	for _, p := range perms[role] {
		if p == perm {
			return true
		}
	}
	return false
}
