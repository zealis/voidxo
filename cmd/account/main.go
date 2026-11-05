package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"log/slog"
	"net/mail"
	"os"
	"regexp"
	"strings"

	"github.com/voidxo/voidxo/internal/modules/auth"
	"github.com/voidxo/voidxo/internal/modules/user"
	"github.com/voidxo/voidxo/internal/platform/config"
	"github.com/voidxo/voidxo/internal/platform/database"
)

var usernamePattern = regexp.MustCompile(`^[\p{Han}a-zA-Z0-9]{2,20}$`)

func main() {
	username := flag.String("username", "", "用户名，2 至 20 个中文、字母或数字")
	email := flag.String("email", "", "邮箱地址，创建账号时必填")
	role := flag.String("role", "user", "角色：user、moderator 或 admin")
	passwordEnv := flag.String("password-env", "VOIDXO_BOOTSTRAP_PASSWORD", "保存密码的环境变量名")
	reset := flag.Bool("reset", false, "重置已存在账号的密码；此时只需 -username 与密码环境变量")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	usernameValue := strings.TrimSpace(*username)
	emailValue := strings.TrimSpace(*email)
	password := os.Getenv(*passwordEnv)

	if !usernamePattern.MatchString(usernameValue) {
		logger.Error("invalid username")
		os.Exit(1)
	}
	if err := auth.ValidateNewPassword(password); err != nil {
		logger.Error("invalid bootstrap password", "error", err)
		os.Exit(1)
	}

	db := openDatabase(logger)
	defer db.Close()
	repo := user.NewMySQLRepository(db)
	hasher := auth.NewPasswordHasher(auth.DefaultArgon2Params)

	if *reset {
		resetPassword(logger, repo, hasher, usernameValue, password)
		return
	}

	if err := validateEmail(emailValue); err != nil {
		logger.Error("invalid email", "error", err)
		os.Exit(1)
	}
	if *role != "user" && *role != "moderator" && *role != "admin" {
		logger.Error("invalid role", "role", *role)
		os.Exit(1)
	}
	createAccount(logger, repo, hasher, usernameValue, emailValue, password, *role)
}

func openDatabase(logger *slog.Logger) *sql.DB {
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	db, err := database.OpenMySQL(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	return db
}

func validateEmail(value string) error {
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || !strings.Contains(value, "@") || len([]rune(value)) > 320 {
		return errors.New("邮箱格式不正确")
	}
	return nil
}

func createAccount(logger *slog.Logger, repo *user.MySQLRepository, hasher *auth.PasswordHasher, username, email, password, role string) {
	passwordHash, err := hasher.Hash(password)
	if err != nil {
		logger.Error("hash password failed", "error", err)
		os.Exit(1)
	}
	account, err := repo.Create(context.Background(), user.CreateParams{
		Username: username, Email: email, PasswordHash: passwordHash, RoleCode: role,
	})
	if errors.Is(err, user.ErrDuplicate) {
		logger.Error("username or email already exists", "hint", "使用 -reset 重置该账号密码")
		os.Exit(1)
	}
	if err != nil {
		logger.Error("create account failed", "error", err)
		os.Exit(1)
	}
	logger.Info("account created", "user_id", account.ID, "username", account.Username, "role", role)
}

// resetPassword 只改密码，不动角色、状态与邮箱；已有会话不受影响，需要时由调用方自行失效。
func resetPassword(logger *slog.Logger, repo *user.MySQLRepository, hasher *auth.PasswordHasher, username, password string) {
	ctx := context.Background()
	account, err := repo.FindByLogin(ctx, username)
	if errors.Is(err, user.ErrNotFound) {
		logger.Error("account not found", "username", username)
		os.Exit(1)
	}
	if err != nil {
		logger.Error("lookup account failed", "error", err)
		os.Exit(1)
	}
	passwordHash, err := hasher.Hash(password)
	if err != nil {
		logger.Error("hash password failed", "error", err)
		os.Exit(1)
	}
	if err := repo.SetPassword(ctx, account.ID, passwordHash); err != nil {
		logger.Error("update password failed", "error", err)
		os.Exit(1)
	}
	logger.Info("password updated", "user_id", account.ID, "username", account.Username)
}
