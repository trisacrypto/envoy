package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"go.rtnl.ai/ulid"
)

//===========================================================================
// Users Store
//===========================================================================

const (
	listUsersSQL   = "SELECT id, name, email, role_id, last_login, created, modified FROM users ORDER BY created DESC"
	filterUsersSQL = "SELECT u.id, u.name, u.email, u.role_id, u.last_login, u.created, u.modified FROM users u JOIN roles r ON role_id=r.id WHERE r.title=:role COLLATE NOCASE ORDER BY u.created DESC"
)

func (s *Store) ListUsers(ctx context.Context, page *models.UserPageInfo) (out *models.UserPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO: handle pagination
	out = &models.UserPage{
		Users: make([]*models.User, 0),
		Page:  &models.UserPageInfo{PageInfo: *models.PageInfoFrom(&page.PageInfo), Role: page.Role},
	}

	// Fetch roles to map onto user information
	// Since there are less than 10 roles, it's easier to do this in memory than in db
	var roles map[int64]*models.Role
	if roles, err = s.fetchRoles(tx); err != nil {
		return nil, err
	}

	var rows *sql.Rows
	if page.Role != "" {
		if rows, err = tx.Query(filterUsersSQL, sql.Named("role", page.Role)); err != nil {
			return nil, dbe(err)
		}
	} else {
		if rows, err = tx.Query(listUsersSQL); err != nil {
			return nil, dbe(err)
		}
	}
	defer rows.Close()

	for rows.Next() {
		// Scan counterparty into memory
		user := &models.User{}
		if err = user.ScanSummary(rows); err != nil {
			return nil, err
		}

		// Assign the role pointer to the user
		if role, ok := roles[user.RoleID]; ok {
			user.SetRole(role)
		}

		out.Users = append(out.Users, user)
	}

	tx.Commit()
	return out, nil
}

const (
	createUserSQL  = "INSERT INTO users (id, name, email, password, role_id, last_login, created, modified) VALUES (:id, :name, :email, :password, :roleID, :lastLogin, :created, :modified)"
	defaultRoleSQL = "SELECT id FROM roles WHERE is_default='t' LIMIT 1"
)

func (s *Store) CreateUser(ctx context.Context, user *models.User) (err error) {
	if !user.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	user.ID = ulid.MakeSecure()
	user.Created = time.Now()
	user.Modified = user.Created

	// If no roleID is assigned, use the default role
	if user.RoleID == 0 {
		if err = tx.QueryRow(defaultRoleSQL).Scan(&user.RoleID); err != nil {
			return err
		}
	}

	if _, err = tx.Exec(createUserSQL, user.Params()...); err != nil {
		return dbe(err)
	}

	return tx.Commit()
}

const (
	retrieveUserByIDSQL    = "SELECT * FROM users WHERE id=:id"
	retrieveUserByEmailSQL = "SELECT * FROM users WHERE email=:email"
)

func (s *Store) RetrieveUser(ctx context.Context, emailOrUserID any) (user *models.User, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var (
		query string
		param sql.NamedArg
	)

	switch t := emailOrUserID.(type) {
	case ulid.ULID:
		query = retrieveUserByIDSQL
		param = sql.Named("id", t)
	case string:
		query = retrieveUserByEmailSQL
		param = sql.Named("email", t)
	default:
		return nil, fmt.Errorf("unknown type %T for email or user id", t)
	}

	// Fetch user details
	user = &models.User{}
	if err = user.Scan(tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	// Fetch user role information
	var role *models.Role
	if role, err = s.fetchRole(tx, user.RoleID); err != nil {
		return nil, err
	}
	user.SetRole(role)

	// Fetch user permissions
	var permissions []string
	if permissions, err = s.fetchUserPermissions(tx, user.ID); err != nil {
		return nil, err
	}
	user.SetPermissions(permissions)

	tx.Commit()
	return user, nil
}

const (
	updateUserSQL = "UPDATE users SET name=:name, email=:email, role_id=:roleID, modified=:modified WHERE id=:id"
)

func (s *Store) UpdateUser(ctx context.Context, user *models.User) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	user.Modified = time.Now()

	var result sql.Result
	if result, err = tx.Exec(updateUserSQL, user.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const setUserPasswordSQL = "UPDATE users SET password=:password, modified=:modified WHERE id=:id"

func (s *Store) SetUserPassword(ctx context.Context, userID ulid.ULID, password string) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	params := []any{
		sql.Named("password", password),
		sql.Named("modified", time.Now()),
		sql.Named("id", userID),
	}

	var result sql.Result
	if result, err = tx.Exec(setUserPasswordSQL, params...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const setUserLastLoginSQL = "UPDATE users SET last_login=:lastLogin, modified=:modified WHERE id=:id"

func (s *Store) SetUserLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	params := []any{
		sql.Named("id", userID),
		sql.Named("lastLogin", sql.NullTime{Time: lastLogin, Valid: !lastLogin.IsZero()}),
		sql.Named("modified", time.Now()),
	}

	var result sql.Result
	if result, err = tx.Exec(setUserLastLoginSQL, params...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const deleteUserSQL = "DELETE FROM users WHERE id=:id"

func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteUserSQL, sql.Named("id", userID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const lookupRoleSQL = "SELECT * FROM roles WHERE title like :role LIMIT 1"

func (s *Store) LookupRole(ctx context.Context, role string) (model *models.Role, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Normalize the role
	role = strings.TrimSpace(role)

	// Fetch role details
	model = &models.Role{}
	if err = model.Scan(tx.QueryRow(lookupRoleSQL, sql.Named("role", role))); err != nil {
		return nil, dbe(err)
	}

	tx.Commit()
	return model, nil
}

const fetchRolesQL = "SELECT * FROM roles"

func (s *Store) fetchRoles(tx *sql.Tx) (roles map[int64]*models.Role, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(fetchRolesQL); err != nil {
		return nil, err
	}
	defer rows.Close()

	roles = make(map[int64]*models.Role, 3)
	for rows.Next() {
		role := &models.Role{}
		if err = role.Scan(rows); err != nil {
			return nil, err
		}
		roles[role.ID] = role
	}

	return roles, dbe(rows.Err())
}

const fetchRoleSQL = "SELECT * FROM roles WHERE id=:roleID"

func (s *Store) fetchRole(tx *sql.Tx, roleID int64) (role *models.Role, err error) {
	role = &models.Role{}
	if err = role.Scan(tx.QueryRow(fetchRoleSQL, sql.Named("roleID", roleID))); err != nil {
		return nil, dbe(err)
	}
	return role, nil
}

const userPermissionsSQL = "SELECT permission FROM user_permissions WHERE user_id=:userID"

func (s *Store) fetchUserPermissions(tx *sql.Tx, userID ulid.ULID) (permissions []string, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(userPermissionsSQL, sql.Named("userID", userID)); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	permissions = make([]string, 0)
	for rows.Next() {
		var permission string
		if err = rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, dbe(rows.Err())
}

//===========================================================================
// APIKeys Store
//===========================================================================

const listAPIKeysSQL = "SELECT id, description, client_id, last_seen, created, modified FROM api_keys"

func (s *Store) ListAPIKeys(ctx context.Context, page *models.PageInfo) (out *models.APIKeyPage, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// TODO: handle pagination
	out = &models.APIKeyPage{
		APIKeys: make([]*models.APIKey, 0),
	}

	var rows *sql.Rows
	if rows, err = tx.Query(listAPIKeysSQL); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		key := &models.APIKey{}
		if err = key.ScanSummary(rows); err != nil {
			return nil, err
		}
		out.APIKeys = append(out.APIKeys, key)
	}

	tx.Commit()
	return out, nil
}

const (
	createKeySQL     = "INSERT INTO api_keys (id, description, client_id, secret, last_seen, created, modified) VALUES (:id, :description, :clientID, :secret, :lastSeen, :created, :modified)"
	createKeyPermSQL = "INSERT INTO api_key_permissions (api_key_id, permission_id, created, modified) VALUES (:keyID, (SELECT id FROM permissions WHERE title=:permission), :created, :modified)"
)

func (s *Store) CreateAPIKey(ctx context.Context, key *models.APIKey) (err error) {
	if !key.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	key.ID = ulid.MakeSecure()
	key.Created = time.Now()
	key.Modified = key.Created

	if _, err = tx.Exec(createKeySQL, key.Params()...); err != nil {
		return dbe(err)
	}

	// Add permissions to API key (can only be set on create)
	permissions := key.Permissions()
	for _, permission := range permissions {
		params := []any{
			sql.Named("keyID", key.ID),
			sql.Named("permission", permission),
			sql.Named("created", key.Created),
			sql.Named("modified", key.Modified),
		}

		if _, err = tx.Exec(createKeyPermSQL, params...); err != nil {
			return dbe(err)
		}
	}

	return tx.Commit()
}

const (
	retrieveKeyByIDSQL     = "SELECT * FROM api_keys WHERE id=:id"
	retrieveKeyByClientSQL = "SELECT * FROM api_keys WHERE client_id=:clientID"
)

func (s *Store) RetrieveAPIKey(ctx context.Context, clientIDOrKeyID any) (key *models.APIKey, err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var (
		query string
		param sql.NamedArg
	)

	switch t := clientIDOrKeyID.(type) {
	case ulid.ULID:
		query = retrieveKeyByIDSQL
		param = sql.Named("id", t)
	case string:
		query = retrieveKeyByClientSQL
		param = sql.Named("clientID", t)
	default:
		return nil, fmt.Errorf("unkown type %T for client id or api key id", t)
	}

	// Fetch api key details
	key = &models.APIKey{}
	if err = key.Scan(tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	// Fetch api key permissions
	var permissions []string
	if permissions, err = s.fetchAPIKeyPermissions(tx, key.ID); err != nil {
		return nil, err
	}
	key.SetPermissions(permissions)

	tx.Commit()
	return key, nil
}

const updateKeySQL = "UPDATE api_keys SET description=:description, last_seen=:lastSeen, modified=:modified WHERE id=:id"

// NOTE: the only thing that can be updated on an api key right now is last_seen
func (s *Store) UpdateAPIKey(ctx context.Context, key *models.APIKey) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	key.Modified = time.Now()
	var result sql.Result
	if result, err = tx.Exec(updateKeySQL, key.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const deleteKeySQL = "DELETE FROM api_keys WHERE id=:id"

func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) (err error) {
	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	var result sql.Result
	if result, err = tx.Exec(deleteKeySQL, sql.Named("id", keyID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return tx.Commit()
}

const keyPermissionsSQL = "SELECT permission FROM api_key_permission_list WHERE api_key_id=:keyID"

func (s *Store) fetchAPIKeyPermissions(tx *sql.Tx, keyID ulid.ULID) (permissions []string, err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(keyPermissionsSQL, sql.Named("keyID", keyID)); err != nil {
		return nil, err
	}
	defer rows.Close()

	permissions = make([]string, 0)
	for rows.Next() {
		var permission string
		if err = rows.Scan(&permission); err != nil {
			return nil, err
		}
		permissions = append(permissions, permission)
	}

	return permissions, dbe(rows.Err())
}
