package auth

import (
	"testing"
)

func TestNewRBACManager(t *testing.T) {
	manager := NewRBACManager()

	if manager == nil {
		t.Fatal("Expected NewRBACManager to return non-nil manager")
	}

	if len(manager.rolePermissions) == 0 {
		t.Error("Expected role permissions to be initialized")
	}
}

func TestHasPermission(t *testing.T) {
	manager := NewRBACManager()

	tests := []struct {
		role       Role
		permission Permission
		expected   bool
	}{
		{RoleUser, PermQueryRead, true},
		{RoleUser, PermQueryWrite, true},
		{RoleUser, PermAdminRead, false},
		{RoleAdmin, PermAdminRead, true},
		{RoleAdmin, PermAdminDelete, false},
		{RoleSuperAdmin, PermAdminDelete, true},
		{RoleGuest, PermQueryWrite, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role)+":"+string(tt.permission), func(t *testing.T) {
			result := manager.HasPermission(tt.role, tt.permission)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetPermissions(t *testing.T) {
	manager := NewRBACManager()

	userPerms := manager.GetPermissions(RoleUser)

	if len(userPerms) == 0 {
		t.Error("Expected RoleUser to have permissions")
	}

	// Check specific permission exists
	hasQueryRead := false
	for _, p := range userPerms {
		if p == PermQueryRead {
			hasQueryRead = true
			break
		}
	}

	if !hasQueryRead {
		t.Error("Expected PermQueryRead in RoleUser permissions")
	}
}

func TestAddPermissionToRole(t *testing.T) {
	manager := NewRBACManager()

	// Guest doesn't have write permission by default
	if manager.HasPermission(RoleGuest, PermQueryWrite) {
		t.Error("Expected RoleGuest to not have PermQueryWrite initially")
	}

	// Add permission
	manager.AddPermissionToRole(RoleGuest, PermQueryWrite)

	if !manager.HasPermission(RoleGuest, PermQueryWrite) {
		t.Error("Expected RoleGuest to have PermQueryWrite after adding")
	}

	// Adding same permission again shouldn't duplicate
	manager.AddPermissionToRole(RoleGuest, PermQueryWrite)
	perms := manager.GetPermissions(RoleGuest)

	count := 0
	for _, p := range perms {
		if p == PermQueryWrite {
			count++
		}
	}

	if count != 1 {
		t.Errorf("Expected PermQueryWrite to appear once, got %d times", count)
	}
}

func TestRemovePermissionFromRole(t *testing.T) {
	manager := NewRBACManager()

	if !manager.HasPermission(RoleUser, PermQueryWrite) {
		t.Error("Expected RoleUser to have PermQueryWrite initially")
	}

	manager.RemovePermissionFromRole(RoleUser, PermQueryWrite)

	if manager.HasPermission(RoleUser, PermQueryWrite) {
		t.Error("Expected RoleUser to not have PermQueryWrite after removal")
	}
}

func TestSetPermissionsForRole(t *testing.T) {
	manager := NewRBACManager()

	newPermissions := []Permission{PermQueryRead, PermQueryWrite}
	manager.SetPermissionsForRole(RoleGuest, newPermissions)

	perms := manager.GetPermissions(RoleGuest)

	if len(perms) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(perms))
	}

	if !manager.HasPermission(RoleGuest, PermQueryRead) {
		t.Error("Expected PermQueryRead")
	}

	if !manager.HasPermission(RoleGuest, PermQueryWrite) {
		t.Error("Expected PermQueryWrite")
	}
}

func TestValidateRole(t *testing.T) {
	manager := NewRBACManager()

	if !manager.ValidateRole(RoleAdmin) {
		t.Error("Expected RoleAdmin to be valid")
	}

	if manager.ValidateRole(Role("invalid_role")) {
		t.Error("Expected invalid role to be invalid")
	}
}

func TestCanUserAccessResource(t *testing.T) {
	manager := NewRBACManager()

	tests := []struct {
		name             string
		userRoles        []string
		requiredPerm     Permission
		expected         bool
	}{
		{"User with permission", []string{"user"}, PermQueryRead, true},
		{"User without permission", []string{"user"}, PermAdminRead, false},
		{"Admin with permission", []string{"admin"}, PermAdminRead, true},
		{"Multiple roles with permission", []string{"guest", "user"}, PermQueryWrite, true},
		{"Multiple roles without permission", []string{"guest"}, PermAdminRead, false},
		{"Empty roles", []string{}, PermQueryRead, false},
		{"Super admin has everything", []string{"super_admin"}, PermAdminDelete, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := manager.CanUserAccessResource(tt.userRoles, tt.requiredPerm)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRoleHierarchy(t *testing.T) {
	manager := NewRBACManager()

	// Verify role hierarchy: SuperAdmin >= Admin >= PowerUser >= User >= Guest
	superAdminPerms := len(manager.GetPermissions(RoleSuperAdmin))
	adminPerms := len(manager.GetPermissions(RoleAdmin))
	powerUserPerms := len(manager.GetPermissions(RolePowerUser))
	userPerms := len(manager.GetPermissions(RoleUser))
	guestPerms := len(manager.GetPermissions(RoleGuest))

	if !(superAdminPerms >= adminPerms && adminPerms >= powerUserPerms && powerUserPerms >= userPerms && userPerms >= guestPerms) {
		t.Logf("SuperAdmin: %d, Admin: %d, PowerUser: %d, User: %d, Guest: %d",
			superAdminPerms, adminPerms, powerUserPerms, userPerms, guestPerms)
		t.Error("Role hierarchy is not maintained")
	}
}

func TestAllPermissions(t *testing.T) {
	perms := AllPermissions()

	if len(perms) == 0 {
		t.Error("Expected AllPermissions to return non-empty list")
	}

	// Verify specific permissions exist
	expectedPerms := []Permission{
		PermQueryRead,
		PermAdminRead,
		PermEmbedWrite,
	}

	for _, ep := range expectedPerms {
		found := false
		for _, p := range perms {
			if p == ep {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected permission %s not found", ep)
		}
	}
}

func TestAllRoles(t *testing.T) {
	roles := AllRoles()

	if len(roles) != 5 {
		t.Errorf("Expected 5 roles, got %d", len(roles))
	}

	expectedRoles := []Role{RoleGuest, RoleUser, RolePowerUser, RoleAdmin, RoleSuperAdmin}

	for _, er := range expectedRoles {
		found := false
		for _, r := range roles {
			if r == er {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected role %s not found", er)
		}
	}
}

func TestPermissionImmutability(t *testing.T) {
	manager := NewRBACManager()

	perms := manager.GetPermissions(RoleUser)
	originalLen := len(perms)

	// Try to modify returned permissions
	perms = append(perms, PermAdminRead)

	// Original should be unchanged
	permsAgain := manager.GetPermissions(RoleUser)
	if len(permsAgain) != originalLen {
		t.Error("Expected GetPermissions to return copies, not references")
	}
}

func TestCustomRolePermissions(t *testing.T) {
	manager := NewRBACManager()

	// Create custom role by setting permissions
	customRole := Role("custom_analyst")
	customPerms := []Permission{
		PermQueryRead,
		PermWorkspaceRead,
		PermEmbedRead,
	}

	manager.SetPermissionsForRole(customRole, customPerms)

	if !manager.ValidateRole(customRole) {
		t.Error("Expected custom role to be valid after setting permissions")
	}

	if !manager.HasPermission(customRole, PermQueryRead) {
		t.Error("Expected custom role to have assigned permission")
	}

	if manager.HasPermission(customRole, PermQueryWrite) {
		t.Error("Expected custom role to not have unassigned permission")
	}
}
