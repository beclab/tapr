package infisical

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bytetrade.io/web3os/tapr/pkg/postgres"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/mitchellh/mapstructure"
	"k8s.io/klog/v2"
)

type PostgresClient struct {
	DB *postgres.DBLogger
}

/*
	export const UsersSchema = z.object({
	  id: z.string().uuid(),
	  email: z.string().nullable().optional(),
	  authMethods: z.string().array().nullable().optional(),
	  superAdmin: z.boolean().default(false).nullable().optional(),
	  firstName: z.string().nullable().optional(),
	  lastName: z.string().nullable().optional(),
	  isAccepted: z.boolean().default(false).nullable().optional(),
	  isMfaEnabled: z.boolean().default(false).nullable().optional(),
	  mfaMethods: z.string().array().nullable().optional(),
	  devices: z.unknown().nullable().optional(),
	  createdAt: z.date(),
	  updatedAt: z.date(),
	  isGhost: z.boolean().default(false),
	  username: z.string(),
	  isEmailVerified: z.boolean().default(false).nullable().optional()
	});
*/
type UserPG struct {
	ID           string   `db:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	Email        string   `db:"email" json:"email" mapstructure:"email"`
	AuthMethods  []string `db:"authMethods" json:"authMethods" mapstructure:"authMethods"`
	SuperAdmin   bool     `db:"superAdmin" json:"superAdmin" mapstructure:"superAdmin"`
	FirstName    string   `db:"firstName" json:"firstName" mapstructure:"firstName"`
	LastName     string   `db:"lastName" json:"lastName" mapstructure:"lastName"`
	IsAccepted   bool     `db:"isAccepted" json:"isAccepted" mapstructure:"isAccepted"`
	IsMfaEnabled bool     `db:"isMfaEnabled" json:"isMfaEnabled" mapstructure:"isMfaEnabled"`
	MfaMethods   []string `db:"mfaMethods" json:"mfaMethods" mapstructure:"mfaMethods"`
	IsGhost      bool     `db:"IsGhost" json:"IsGhost" mapstructure:"IsGhost"`
	Username     string   `db:"Username" json:"Username" mapstructure:"Username"`
}

/*
	export const UserEncryptionKeysSchema = z.object({
	  id: z.string().uuid(),
	  clientPublicKey: z.string().nullable().optional(),
	  serverPrivateKey: z.string().nullable().optional(),
	  encryptionVersion: z.number().default(2).nullable().optional(),
	  protectedKey: z.string().nullable().optional(),
	  protectedKeyIV: z.string().nullable().optional(),
	  protectedKeyTag: z.string().nullable().optional(),
	  publicKey: z.string(),
	  encryptedPrivateKey: z.string(),
	  iv: z.string(),
	  tag: z.string(),
	  salt: z.string(),
	  verifier: z.string(),
	  userId: z.string().uuid()
	});
*/
type UserEncryptionKeysPG struct {
	ID                  string `db:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	ClientPublicKey     string `db:"clientPublicKey" json:"clientPublicKey" mapstructure:"clientPublicKey"`
	ServerPrivateKey    string `db:"serverPrivateKey" json:"serverPrivateKey" mapstructure:"serverPrivateKey"`
	EncryptionVersion   int32  `db:"encryptionVersion" json:"encryptionVersion" mapstructure:"encryptionVersion"`
	ProtectedKey        string `db:"protectedKey" json:"protectedKey" mapstructure:"protectedKey"`
	ProtectedKeyIV      string `db:"protectedKeyIV" json:"protectedKeyIV" mapstructure:"protectedKeyIV"`
	ProtectedKeyTag     string `db:"protectedKeyTag" json:"protectedKeyTag" mapstructure:"protectedKeyTag"`
	PublicKey           string `db:"publicKey" json:"publicKey" mapstructure:"publicKey"`
	EncryptedPrivateKey string `db:"encryptedPrivateKey" json:"encryptedPrivateKey" mapstructure:"encryptedPrivateKey"`
	IV                  string `db:"iv" json:"iv" mapstructure:"iv"`
	Tag                 string `db:"tag" json:"tag" mapstructure:"tag"`
	Salt                string `db:"salt" json:"salt" mapstructure:"salt"`
	Verifier            string `db:"verifier" json:"verifier" mapstructure:"verifier"`
	UserID              string `db:"userId" json:"userId" mapstructure:"userId"`
}

/*
	export const OrganizationsSchema = z.object({
	  id: z.string().uuid(),
	  name: z.string(),
	  customerId: z.string().nullable().optional(),
	  slug: z.string(),
	  createdAt: z.date(),
	  updatedAt: z.date(),
	  authEnforced: z.boolean().default(false).nullable().optional(),
	  scimEnabled: z.boolean().default(false).nullable().optional()
	});
*/
type OrganizationsPG struct {
	ID           string `db:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	Name         string `db:"name" json:"name" mapstructure:"name"`
	CustomerId   string `db:"customerId" json:"customerId" mapstructure:"customerId"`
	Slug         string `db:"slug" json:"slug" mapstructure:"slug"`
	AuthEnforced bool   `db:"authEnforced" json:"authEnforced" mapstructure:"authEnforced"`
	ScimEnabled  bool   `db:"scimEnabled" json:"scimEnabled" mapstructure:"scimEnabled"`
}

/*
	export const OrgMembershipsSchema = z.object({
	  id: z.string().uuid(),
	  role: z.string(),
	  status: z.string().default("invited"),
	  inviteEmail: z.string().nullable().optional(),
	  createdAt: z.date(),
	  updatedAt: z.date(),
	  userId: z.string().uuid().nullable().optional(),
	  orgId: z.string().uuid(),
	  roleId: z.string().uuid().nullable().optional()
	});
*/
type OrgMembershipsPG struct {
	ID          string `db:"id,omitempty" json:"_id,omitempty" mapstructure:"_id,omitempty"`
	Role        string `db:"role" json:"role" mapstructure:"role"`
	Status      string `db:"status" json:"status" mapstructure:"status"`
	InviteEmail string `db:"inviteEmail" json:"inviteEmail" mapstructure:"inviteEmail"`
	UserId      string `db:"userId" json:"userId" mapstructure:"userId"`
	OrgId       string `db:"orgId" json:"orgId" mapstructure:"orgId"`
	RoleId      string `db:"roleId" json:"roleId" mapstructure:"roleId"`
}

func (c *PostgresClient) Close() {
	err := c.DB.Close()
	if err != nil {
		klog.Error("close db error, ", err)
	}
}

func NewClient(dsn string) (*PostgresClient, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, err
	}

	dbProxy := postgres.DBLogger{DB: db}

	dbProxy.Debug()

	return &PostgresClient{DB: &dbProxy}, nil
}

func (c *PostgresClient) GetUser(basectx context.Context, email string) (*UserEncryptionKeysPG, error) {
	if email == "" {
		return nil, errors.New("email is empty")
	}

	ctx, cancel := context.WithTimeout(basectx, 10*time.Second)
	defer cancel()

	sql := "select b.* from users a, user_encryption_keys b  where a.email=:email and a.id = b.\"userId\""
	res, err := c.DB.NamedQueryContext(ctx, sql, map[string]interface{}{
		"email": email,
	})

	if err != nil {
		klog.Error("fetch user error, ", err)

		return nil, err
	}

	var user UserEncryptionKeysPG
	if res.Next() {
		err = res.StructScan(&user)
		if err != nil {
			klog.Error("scan user data error, ", err)
			return nil, err
		}
		return &user, nil
	}

	return nil, nil
}

func (c *PostgresClient) SaveUser(basectx context.Context, user *UserPG, userEnc *UserEncryptionKeysPG) (string, error) {
	if user == nil {
		return "", errors.New("user is empty")
	}

	uid, err := user.Create(basectx, c)
	if err != nil {
		return "", err
	}

	if userEnc != nil {
		userEnc.UserID = uid
		_, err := userEnc.Create(basectx, c)
		if err != nil {
			return "", err
		}
	}

	org := OrganizationsPG{
		Name: "Terminus",
	}

	orgId, err := org.Create(basectx, c)
	if err != nil {
		return "", err
	}

	member := OrgMembershipsPG{
		OrgId:  orgId,
		UserId: uid,
		Role:   "owner",
		Status: "accepted",
	}

	_, err = member.Create(basectx, c)
	if err != nil {
		return "", err
	}

	return uid, nil
}

func ValueMapper[T interface{}](obj T) (fields, namedKeys []string, err error) {
	values := make(map[string]interface{})
	err = mapstructure.Decode(obj, &values)
	if err != nil {
		klog.Error("decode object value error, ", err)
		return
	}

	fields = make([]string, 0, len(values))
	namedKeys = make([]string, 0, len(values))
	for k := range values {
		fields = append(fields, "\""+k+"\"")
		namedKeys = append(namedKeys, ":"+k)
	}

	return
}

func insert[T interface{}](basectx context.Context, client *PostgresClient, table string, obj T, setId func(T, string) T) (id string, err error) {
	id = uuid.New().String()

	obj = setId(obj, id)

	fields, keys, err := ValueMapper(obj)
	if err != nil {
		return
	}

	sql := fmt.Sprintf("insert into %s(%s) values(%s)", table, strings.Join(fields, ","), strings.Join(keys, ","))
	ctx, cancel := context.WithTimeout(basectx, 10*time.Second)
	defer cancel()

	_, err = client.DB.NamedExecContext(ctx, sql, obj)
	if err != nil {
		klog.Error("create error, ", err, ", ", table)
		return
	}

	return

}
func (u *UserPG) Create(basectx context.Context, client *PostgresClient) (id string, err error) {
	return insert(basectx, client, "users", u, func(obj *UserPG, id string) *UserPG {
		obj.ID = id
		return obj
	})
}

func (u *UserEncryptionKeysPG) Create(basectx context.Context, client *PostgresClient) (id string, err error) {
	return insert(basectx, client, "user_encryption_keys", u, func(obj *UserEncryptionKeysPG, id string) *UserEncryptionKeysPG {
		obj.ID = id
		return obj
	})
}

func (o *OrganizationsPG) Create(basectx context.Context, client *PostgresClient) (id string, err error) {
	return insert(basectx, client, "organizations", o, func(obj *OrganizationsPG, id string) *OrganizationsPG {
		obj.ID = id
		return obj
	})
}

func (o *OrgMembershipsPG) Create(basectx context.Context, client *PostgresClient) (id string, err error) {
	return insert(basectx, client, "org_memberships", o, func(obj *OrgMembershipsPG, id string) *OrgMembershipsPG {
		obj.ID = id
		return obj
	})
}
