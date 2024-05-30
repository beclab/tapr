package infisical

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"bytetrade.io/web3os/tapr/pkg/constants"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

type tokenClaims struct {
	jwt.StandardClaims
	UserId string `json:"userId"`
}

type tokenIssuer struct {
	kubeconfig    *rest.Config
	getUserAndPwd func(ctx context.Context) (user string, pwd string, err error)
}

func NewTokenIssuer(kubeconfig *rest.Config) *tokenIssuer {
	return &tokenIssuer{kubeconfig: kubeconfig}
}

func (t *tokenIssuer) WithUserAndPwd(f func(ctx context.Context) (user string, pwd string, err error)) *tokenIssuer {
	t.getUserAndPwd = f
	return t
}

func (t *tokenIssuer) IssueInfisicalToken(next func(c *fiber.Ctx) error) func(c *fiber.Ctx) error {
	return func(c *fiber.Ctx) error {

		// get user email from ctx
		email, ok := c.Context().UserValueBytes([]byte(constants.UserEmailCtxKey)).(string)
		if !ok {
			return c.JSON(fiber.Map{
				"code":    http.StatusUnauthorized,
				"message": "auth user email is invalid",
				"data":    nil,
			})
		}

		ctx := c.UserContext()
		user, err := t.getUserFromInfisicalDB(ctx, email)
		if err != nil {
			return c.JSON(fiber.Map{
				"code":    http.StatusUnauthorized,
				"message": fmt.Sprintf("get user from infisical error, %s, %s", err.Error(), email),
				"data":    nil,
			})
		}
		c.Context().SetUserValueBytes(constants.UserCtxKey, user)
		uid := user.UserID
		klog.Info("get user id, ", uid)

		authKey, refreshKey, err := t.getJwtSecret(ctx)
		if err != nil {
			return c.JSON(fiber.Map{
				"code":    http.StatusUnauthorized,
				"message": fmt.Sprintf("get user jwt key error, %s", err.Error()),
				"data":    nil,
			})
		}

		authToken, err := t.issueToken(uid, authKey, 10*24*time.Hour)
		if err != nil {
			return c.JSON(fiber.Map{
				"code":    http.StatusUnauthorized,
				"message": fmt.Sprintf("unable to sign auth token, %s", err.Error()),
				"data":    nil,
			})
		}
		c.Context().SetUserValueBytes(constants.UserAuthTokenCtxKey, authToken)
		klog.Info("get user token, ", authToken)

		refreshToken, err := t.issueToken(uid, refreshKey, 10*24*time.Hour)
		if err != nil {
			return c.JSON(fiber.Map{
				"code":    http.StatusUnauthorized,
				"message": fmt.Sprintf("unable to sign refresh token, %s", err.Error()),
				"data":    nil,
			})
		}
		c.Context().SetUserValueBytes(constants.UserRefreshTokenCtxKey, refreshToken)

		return next(c)
	}
}

func (t *tokenIssuer) getUserFromInfisicalDB(ctx context.Context, email string) (*UserEncryptionKeysPG, error) {
	return t.getUserFromInfisicalPostgres(ctx, email)
}

func (t *tokenIssuer) getUserFromInfisicalMongoDB(ctx context.Context, email string) (*UserMDB, error) {
	user, password, err := t.getUserAndPwd(ctx)
	if err != nil {
		return nil, err
	}
	mongo := MongoClient{
		User:     user,
		Password: password,
		Database: InfisicalDBName,
		Addr:     InfisicalDBAddr,
	}

	return mongo.GetUser(ctx, email)
}

func (t *tokenIssuer) getUserFromInfisicalPostgres(ctx context.Context, email string) (*UserEncryptionKeysPG, error) {
	user, password, err := t.getUserAndPwd(ctx)
	if err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, password, InfisicalDBAddr, InfisicalDBName)

	pg, err := NewClient(dsn)
	if err != nil {
		return nil, err
	}

	defer pg.Close()

	return pg.GetUser(ctx, email)
}

func (t *tokenIssuer) getJwtSecret(ctx context.Context) (authKey string, refreshKey string, err error) {
	client, err := kubernetes.NewForConfig(t.kubeconfig)
	if err != nil {
		return "", "", err
	}

	backendSecret, err := client.CoreV1().Secrets(InfisicalNamespace).Get(ctx, "infisical-backend", metav1.GetOptions{})
	if err != nil {
		return "", "", err
	}

	authKey = string(backendSecret.Data["JWT_AUTH_SECRET"])
	refreshKey = string(backendSecret.Data["JWT_REFRESH_SECRET"])

	return authKey, refreshKey, nil
}

func (t *tokenIssuer) issueToken(userId string, key string, expireIn time.Duration) (string, error) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims{
		UserId: userId,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(expireIn).Unix(),
		},
	}).SignedString([]byte(key))
	if err != nil {
		return "", err
	}

	return token, nil
}
