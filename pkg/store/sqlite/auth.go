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
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListUsers(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (t *Tx) ListUsers(page *models.UserPageInfo) (out *models.UserPage, err error) {
	// TODO: handle pagination
	out = &models.UserPage{
		Users: make([]*models.User, 0),
		Page:  &models.UserPageInfo{PageInfo: *models.PageInfoFrom(&page.PageInfo), Role: page.Role},
	}

	// Fetch roles to map onto user information
	// Since there are less than 10 roles, it's easier to do this in memory than in db
	var roles map[int64]*models.Role
	if roles, err = t.fetchRoles(); err != nil {
		return nil, err
	}

	var rows *sql.Rows
	if page.Role != "" {
		if rows, err = t.tx.Query(filterUsersSQL, sql.Named("role", page.Role)); err != nil {
			return nil, dbe(err)
		}
	} else {
		if rows, err = t.tx.Query(listUsersSQL); err != nil {
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

	return out, nil
}

const (
	createUserSQL  = "INSERT INTO users (id, name, email, password, role_id, last_login, created, modified) VALUES (:id, :name, :email, :password, :roleID, :lastLogin, :created, :modified)"
	defaultRoleSQL = "SELECT id FROM roles WHERE is_default='t' LIMIT 1"
)

func (s *Store) CreateUser(ctx context.Context, user *models.User) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateUser(user); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateUser(user *models.User) (err error) {
	if !user.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	user.ID = ulid.MakeSecure()
	user.Created = time.Now()
	user.Modified = user.Created

	// If no roleID is assigned, use the default role
	if user.RoleID == 0 {
		if err = t.tx.QueryRow(defaultRoleSQL).Scan(&user.RoleID); err != nil {
			return err
		}
	}

	if _, err = t.tx.Exec(createUserSQL, user.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}

const (
	retrieveUserByIDSQL    = "SELECT * FROM users WHERE id=:id"
	retrieveUserByEmailSQL = "SELECT * FROM users WHERE email=:email"
)

func (s *Store) RetrieveUser(ctx context.Context, emailOrUserID any) (user *models.User, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if user, err = tx.RetrieveUser(emailOrUserID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return user, nil
}

func (t *Tx) RetrieveUser(emailOrUserID any) (user *models.User, err error) {
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
	if err = user.Scan(t.tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	// Fetch user role information
	var role *models.Role
	if role, err = t.fetchRole(user.RoleID); err != nil {
		return nil, err
	}
	user.SetRole(role)

	// Fetch user permissions
	var permissions []string
	if permissions, err = t.fetchUserPermissions(user.ID); err != nil {
		return nil, err
	}
	user.SetPermissions(permissions)

	return user, nil
}

const (
	updateUserSQL = "UPDATE users SET name=:name, email=:email, role_id=:roleID, modified=:modified WHERE id=:id"
)

func (s *Store) UpdateUser(ctx context.Context, user *models.User) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateUser(user); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) UpdateUser(user *models.User) (err error) {
	user.Modified = time.Now()

	var result sql.Result
	if result, err = t.tx.Exec(updateUserSQL, user.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const setUserPasswordSQL = "UPDATE users SET password=:password, modified=:modified WHERE id=:id"

func (s *Store) SetUserPassword(ctx context.Context, userID ulid.ULID, password string) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.SetUserPassword(userID, password); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) SetUserPassword(userID ulid.ULID, password string) (err error) {
	params := []any{
		sql.Named("password", password),
		sql.Named("modified", time.Now()),
		sql.Named("id", userID),
	}

	var result sql.Result
	if result, err = t.tx.Exec(setUserPasswordSQL, params...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const setUserLastLoginSQL = "UPDATE users SET last_login=:lastLogin, modified=:modified WHERE id=:id"

func (s *Store) SetUserLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.SetUserLastLogin(userID, lastLogin); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) SetUserLastLogin(userID ulid.ULID, lastLogin time.Time) (err error) {
	params := []any{
		sql.Named("id", userID),
		sql.Named("lastLogin", sql.NullTime{Time: lastLogin, Valid: !lastLogin.IsZero()}),
		sql.Named("modified", time.Now()),
	}

	var result sql.Result
	if result, err = t.tx.Exec(setUserLastLoginSQL, params...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const deleteUserSQL = "DELETE FROM users WHERE id=:id"

func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteUser(userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteUser(userID ulid.ULID) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteUserSQL, sql.Named("id", userID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

const lookupRoleSQL = "SELECT * FROM roles WHERE title like :role LIMIT 1"

func (s *Store) LookupRole(ctx context.Context, role string) (model *models.Role, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if model, err = tx.LookupRole(role); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return model, nil
}

func (t *Tx) LookupRole(role string) (model *models.Role, err error) {
	// Normalize the role
	role = strings.TrimSpace(role)

	// Fetch role details
	model = &models.Role{}
	if err = model.Scan(t.tx.QueryRow(lookupRoleSQL, sql.Named("role", role))); err != nil {
		return nil, dbe(err)
	}

	return model, nil
}

const fetchRolesQL = "SELECT * FROM roles"

func (t *Tx) fetchRoles() (roles map[int64]*models.Role, err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(fetchRolesQL); err != nil {
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

func (t *Tx) fetchRole(roleID int64) (role *models.Role, err error) {
	role = &models.Role{}
	if err = role.Scan(t.tx.QueryRow(fetchRoleSQL, sql.Named("roleID", roleID))); err != nil {
		return nil, dbe(err)
	}
	return role, nil
}

const userPermissionsSQL = "SELECT permission FROM user_permissions WHERE user_id=:userID"

func (t *Tx) fetchUserPermissions(userID ulid.ULID) (permissions []string, err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(userPermissionsSQL, sql.Named("userID", userID)); err != nil {
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
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListAPIKeys(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return out, nil
}

func (t *Tx) ListAPIKeys(page *models.PageInfo) (out *models.APIKeyPage, err error) {
	// TODO: handle pagination
	out = &models.APIKeyPage{
		APIKeys: make([]*models.APIKey, 0),
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(listAPIKeysSQL); err != nil {
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

	return out, nil
}

const (
	createKeySQL     = "INSERT INTO api_keys (id, description, client_id, secret, last_seen, created, modified) VALUES (:id, :description, :clientID, :secret, :lastSeen, :created, :modified)"
	createKeyPermSQL = "INSERT INTO api_key_permissions (api_key_id, permission_id, created, modified) VALUES (:keyID, (SELECT id FROM permissions WHERE title=:permission), :created, :modified)"
)

func (s *Store) CreateAPIKey(ctx context.Context, key *models.APIKey) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateAPIKey(key); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) CreateAPIKey(key *models.APIKey) (err error) {
	if !key.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	key.ID = ulid.MakeSecure()
	key.Created = time.Now()
	key.Modified = key.Created

	if _, err = t.tx.Exec(createKeySQL, key.Params()...); err != nil {
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

		if _, err = t.tx.Exec(createKeyPermSQL, params...); err != nil {
			return dbe(err)
		}
	}

	return nil
}

const (
	retrieveKeyByIDSQL     = "SELECT * FROM api_keys WHERE id=:id"
	retrieveKeyByClientSQL = "SELECT * FROM api_keys WHERE client_id=:clientID"
)

func (s *Store) RetrieveAPIKey(ctx context.Context, clientIDOrKeyID any) (key *models.APIKey, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if key, err = tx.RetrieveAPIKey(clientIDOrKeyID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return key, nil
}

func (t *Tx) RetrieveAPIKey(clientIDOrKeyID any) (key *models.APIKey, err error) {
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
	if err = key.Scan(t.tx.QueryRow(query, param)); err != nil {
		return nil, dbe(err)
	}

	// Fetch api key permissions
	var permissions []string
	if permissions, err = t.fetchAPIKeyPermissions(key.ID); err != nil {
		return nil, err
	}
	key.SetPermissions(permissions)

	return key, nil
}

const updateKeySQL = "UPDATE api_keys SET description=:description, last_seen=:lastSeen, modified=:modified WHERE id=:id"

// NOTE: the only thing that can be updated on an api key right now is last_seen
func (s *Store) UpdateAPIKey(ctx context.Context, key *models.APIKey) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.UpdateAPIKey(key); err != nil {
		return err
	}

	return tx.Commit()
}

// NOTE: the only thing that can be updated on an api key right now is last_seen
func (t *Tx) UpdateAPIKey(key *models.APIKey) (err error) {
	key.Modified = time.Now()
	var result sql.Result
	if result, err = t.tx.Exec(updateKeySQL, key.Params()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

const deleteKeySQL = "DELETE FROM api_keys WHERE id=:id"

func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.DeleteAPIKey(keyID); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteAPIKey(keyID ulid.ULID) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteKeySQL, sql.Named("id", keyID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}
	return nil
}

const keyPermissionsSQL = "SELECT permission FROM api_key_permission_list WHERE api_key_id=:keyID"

func (t *Tx) fetchAPIKeyPermissions(keyID ulid.ULID) (permissions []string, err error) {
	var rows *sql.Rows
	if rows, err = t.tx.Query(keyPermissionsSQL, sql.Named("keyID", keyID)); err != nil {
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

//===========================================================================
// ResetPasswordLink Store
//===========================================================================

const listResetPasswordLinkSQL = "SELECT * FROM reset_password_link"

func (s *Store) ListResetPasswordLinks(ctx context.Context, page *models.PageInfo) (out *models.ResetPasswordLinkPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListResetPasswordLinks(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (t *Tx) ListResetPasswordLinks(page *models.PageInfo) (out *models.ResetPasswordLinkPage, err error) {
	// TODO: handle pagination
	out = &models.ResetPasswordLinkPage{
		Links: make([]*models.ResetPasswordLink, 0),
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(listResetPasswordLinkSQL); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		link := &models.ResetPasswordLink{}
		if err = link.Scan(rows); err != nil {
			return nil, err
		}
		out.Links = append(out.Links, link)
	}

	return out, nil
}

const (
	checkExistingLinkSQL       = "SELECT * FROM reset_password_link WHERE user_id=:userID"
	createResetPasswordLinkSQL = "INSERT INTO reset_password_link (id, user_id, email, expiration, signature, sent_on, created, modified) VALUES (:id, :userId, :email, :expiration, :signature, :sentOn, :created, :modified)"
)

// Create a ResetPasswordLink record in the database. This method checks to see if there
// is an existing link for the user and if so, it will return ErrTooSoon if that link
// is not expired. If the link is expired, it will be deleted and a new one created.
func (s *Store) CreateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateResetPasswordLink(link); err != nil {
		return err
	}

	return tx.Commit()
}

// Create a ResetPasswordLink record in the database. This method checks to see if there
// is an existing link for the user and if so, it will return ErrTooSoon if that link
// is not expired. If the link is expired, it will be deleted and a new one created.
func (t *Tx) CreateResetPasswordLink(link *models.ResetPasswordLink) (err error) {
	if !link.ID.IsZero() {
		return dberr.ErrNoIDOnCreate
	}

	link.ID = ulid.MakeSecure()
	link.Created = time.Now()
	link.Modified = link.Created

	existing := &models.ResetPasswordLink{}
	if err = existing.Scan(t.tx.QueryRow(checkExistingLinkSQL, sql.Named("userID", link.UserID))); err != nil {
		if err != sql.ErrNoRows {
			return dbe(err)
		}
	}

	if !existing.Expiration.IsZero() {
		// If the existing link is not expired, then return ErrTooSoon
		if !existing.IsExpired() {
			return dberr.ErrTooSoon
		}

		// Delete the existing link if it is expired
		if _, err = t.tx.Exec(deleteResetPasswordLinkSQL, sql.Named("id", existing.ID)); err != nil {
			return dbe(err)
		}
	}

	if _, err = t.tx.Exec(createResetPasswordLinkSQL, link.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}

const retrieveResetPasswordLinkByIDSQL = "SELECT * FROM reset_password_link WHERE id=:id"

// Retrieve a ResetPasswordLink in the database by its ID.
func (s *Store) RetrieveResetPasswordLink(ctx context.Context, linkID ulid.ULID) (link *models.ResetPasswordLink, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if link, err = tx.RetrieveResetPasswordLink(linkID); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return link, nil
}

// Retrieve a ResetPasswordLink in the database by its ID.
func (t *Tx) RetrieveResetPasswordLink(linkID ulid.ULID) (link *models.ResetPasswordLink, err error) {
	link = &models.ResetPasswordLink{}
	if err = link.Scan(t.tx.QueryRow(retrieveResetPasswordLinkByIDSQL, sql.Named("id", linkID))); err != nil {
		return nil, dbe(err)
	}
	return link, nil
}

const updateResetPasswordLinkSQL = "UPDATE reset_password_link SET signature=:signature, sent_on=:sentOn, modified=:modified  WHERE id=:id"

// Update a ResetPasswordLink record. Only updates the Signature, SentOn,
// VerifiedOn, and Modified fields.
func (s *Store) UpdateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.UpdateResetPasswordLink(link); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// Update a ResetPasswordLink record. Only updates the Signature, SentOn,
// VerifiedOn, and Modified fields.
func (t *Tx) UpdateResetPasswordLink(link *models.ResetPasswordLink) (err error) {
	link.Modified = time.Now()

	var result sql.Result
	if result, err = t.tx.Exec(updateResetPasswordLinkSQL, link.UpdateParams()...); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}

	return nil
}

const deleteResetPasswordLinkSQL = "DELETE FROM reset_password_link WHERE id=:id"

func (s *Store) DeleteResetPasswordLink(ctx context.Context, linkID ulid.ULID) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, nil); err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.DeleteResetPasswordLink(linkID); err != nil {
		return err
	}

	return tx.Commit()
}

func (t *Tx) DeleteResetPasswordLink(linkID ulid.ULID) (err error) {
	var result sql.Result
	if result, err = t.tx.Exec(deleteResetPasswordLinkSQL, sql.Named("id", linkID)); err != nil {
		return dbe(err)
	} else if nRows, _ := result.RowsAffected(); nRows == 0 {
		return dberr.ErrNotFound
	}
	return nil
}
