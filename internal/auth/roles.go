package auth

// Role represents a user privilege level in AlertLens.
//
// The four roles form a strict hierarchy: each higher role implies all
// capabilities of every lower role.
//
//	viewer        — read-only: view alerts, silences, routing tree.
//	silencer      — viewer + create / update / expire silences & visual acks.
//	config-editor — silencer + read and write alertmanager configuration.
//	admin         — config-editor + full control (future admin-only operations).
type Role string

const (
	RoleViewer       Role = "viewer"
	RoleSilencer     Role = "silencer"
	RoleConfigEditor Role = "config-editor"
	RoleAdmin        Role = "admin"
)

// roleRank assigns a numeric level to each role.
// A request carrying a role at level N satisfies any RequireRole check whose
// threshold is ≤ N (i.e. higher rank ⇒ more privileges).
var roleRank = map[Role]int{
	RoleViewer:       1,
	RoleSilencer:     2,
	RoleConfigEditor: 3,
	RoleAdmin:        4,
}

// HasAtLeast returns true when r has at least the privilege level of required.
// Unknown roles (rank 0) never satisfy any requirement.
func (r Role) HasAtLeast(required Role) bool {
	return roleRank[r] >= roleRank[required]
}

// IsValid returns true when r is a recognised role.
func (r Role) IsValid() bool {
	_, ok := roleRank[r]
	return ok
}
