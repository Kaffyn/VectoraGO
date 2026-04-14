package auth

// Permission representa uma permissão no sistema
type Permission string

const (
	// Query permissions
	PermQueryRead   Permission = "query:read"
	PermQueryWrite  Permission = "query:write"
	PermQueryDelete Permission = "query:delete"

	// Workspace permissions
	PermWorkspaceRead   Permission = "workspace:read"
	PermWorkspaceWrite  Permission = "workspace:write"
	PermWorkspaceDelete Permission = "workspace:delete"

	// Embedding permissions
	PermEmbedRead   Permission = "embed:read"
	PermEmbedWrite  Permission = "embed:write"
	PermEmbedDelete Permission = "embed:delete"

	// Admin permissions
	PermAdminRead   Permission = "admin:read"
	PermAdminWrite  Permission = "admin:write"
	PermAdminDelete Permission = "admin:delete"

	// Tenant permissions
	PermTenantRead   Permission = "tenant:read"
	PermTenantWrite  Permission = "tenant:write"
	PermTenantDelete Permission = "tenant:delete"
)

// Role representa um papel no sistema
type Role string

const (
	RoleGuest      Role = "guest"
	RoleUser       Role = "user"
	RolePowerUser  Role = "power_user"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

// RBACManager gerencia roles, permissions e suas associações
type RBACManager struct {
	rolePermissions map[Role][]Permission
}

// NewRBACManager cria um novo gerenciador RBAC com permissões padrão
func NewRBACManager() *RBACManager {
	manager := &RBACManager{
		rolePermissions: make(map[Role][]Permission),
	}

	// Configurar permissões padrão para cada role
	manager.setDefaultPermissions()

	return manager
}

// setDefaultPermissions configura as permissões padrão para cada role
func (rm *RBACManager) setDefaultPermissions() {
	// Guest: apenas leitura de queries públicas
	rm.rolePermissions[RoleGuest] = []Permission{
		PermQueryRead,
	}

	// User: CRUD de queries, leitura de workspaces
	rm.rolePermissions[RoleUser] = []Permission{
		PermQueryRead,
		PermQueryWrite,
		PermQueryDelete,
		PermWorkspaceRead,
		PermEmbedRead,
	}

	// PowerUser: tudo que User + write/delete workspace + embed write
	rm.rolePermissions[RolePowerUser] = []Permission{
		PermQueryRead,
		PermQueryWrite,
		PermQueryDelete,
		PermWorkspaceRead,
		PermWorkspaceWrite,
		PermWorkspaceDelete,
		PermEmbedRead,
		PermEmbedWrite,
		PermEmbedDelete,
		PermTenantRead,
	}

	// Admin: tudo menos super_admin
	rm.rolePermissions[RoleAdmin] = []Permission{
		PermQueryRead,
		PermQueryWrite,
		PermQueryDelete,
		PermWorkspaceRead,
		PermWorkspaceWrite,
		PermWorkspaceDelete,
		PermEmbedRead,
		PermEmbedWrite,
		PermEmbedDelete,
		PermAdminRead,
		PermAdminWrite,
		PermTenantRead,
		PermTenantWrite,
	}

	// SuperAdmin: todas as permissões
	rm.rolePermissions[RoleSuperAdmin] = []Permission{
		PermQueryRead,
		PermQueryWrite,
		PermQueryDelete,
		PermWorkspaceRead,
		PermWorkspaceWrite,
		PermWorkspaceDelete,
		PermEmbedRead,
		PermEmbedWrite,
		PermEmbedDelete,
		PermAdminRead,
		PermAdminWrite,
		PermAdminDelete,
		PermTenantRead,
		PermTenantWrite,
		PermTenantDelete,
	}
}

// HasPermission verifica se uma role tem uma permissão específica
func (rm *RBACManager) HasPermission(role Role, permission Permission) bool {
	permissions, exists := rm.rolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// GetPermissions retorna todas as permissões de uma role
func (rm *RBACManager) GetPermissions(role Role) []Permission {
	if permissions, exists := rm.rolePermissions[role]; exists {
		// Retornar cópia para evitar modificações externas
		permsCopy := make([]Permission, len(permissions))
		copy(permsCopy, permissions)
		return permsCopy
	}
	return []Permission{}
}

// AddPermissionToRole adiciona uma permissão a uma role
func (rm *RBACManager) AddPermissionToRole(role Role, permission Permission) {
	if !rm.HasPermission(role, permission) {
		rm.rolePermissions[role] = append(rm.rolePermissions[role], permission)
	}
}

// RemovePermissionFromRole remove uma permissão de uma role
func (rm *RBACManager) RemovePermissionFromRole(role Role, permission Permission) {
	if permissions, exists := rm.rolePermissions[role]; exists {
		for i, p := range permissions {
			if p == permission {
				rm.rolePermissions[role] = append(permissions[:i], permissions[i+1:]...)
				break
			}
		}
	}
}

// SetPermissionsForRole define todas as permissões de uma role (sobrescreve anteriores)
func (rm *RBACManager) SetPermissionsForRole(role Role, permissions []Permission) {
	rm.rolePermissions[role] = append([]Permission{}, permissions...)
}

// ValidateRole verifica se um role é válido
func (rm *RBACManager) ValidateRole(role Role) bool {
	_, exists := rm.rolePermissions[role]
	return exists
}

// CanUserAccessResource verifica se um usuário (com seus roles) pode acessar um recurso
func (rm *RBACManager) CanUserAccessResource(userRoles []string, requiredPermission Permission) bool {
	if len(userRoles) == 0 {
		return false
	}

	for _, roleStr := range userRoles {
		role := Role(roleStr)
		if rm.HasPermission(role, requiredPermission) {
			return true
		}
	}
	return false
}

// AllPermissions retorna todas as permissões conhecidas do sistema
func AllPermissions() []Permission {
	return []Permission{
		PermQueryRead,
		PermQueryWrite,
		PermQueryDelete,
		PermWorkspaceRead,
		PermWorkspaceWrite,
		PermWorkspaceDelete,
		PermEmbedRead,
		PermEmbedWrite,
		PermEmbedDelete,
		PermAdminRead,
		PermAdminWrite,
		PermAdminDelete,
		PermTenantRead,
		PermTenantWrite,
		PermTenantDelete,
	}
}

// AllRoles retorna todos os roles conhecidos do sistema
func AllRoles() []Role {
	return []Role{
		RoleGuest,
		RoleUser,
		RolePowerUser,
		RoleAdmin,
		RoleSuperAdmin,
	}
}
