package main

import (
	"context"
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
	email := flag.String("email", "", "邮箱地址")
	role := flag.String("role", "user", "角色：user、moderator 或 admin")
	passwordEnv := flag.String("password-env", "VOIDXO_BOOTSTRAP_PASSWORD", "保存初始密码的环境变量名")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	usernameValue := strings.TrimSpace(*username)
	emailValue := strings.TrimSpace(*email)
	password := os.Getenv(*passwordEnv)
	if !usernamePattern.MatchString(usernameValue) {
		logger.Error("invalid username")
		os.Exit(1)
	}
	parsed, err := mail.ParseAddress(emailValue)
	if err != nil || parsed.Address != emailValue || !strings.Contains(emailValue, "@") || len([]rune(emailValue)) > 320 {
		logger.Error("invalid email")
		os.Exit(1)
	}
	if err := auth.ValidateNewPassword(password); err != nil {
		logger.Error("invalid bootstrap password", "error", err)
		os.Exit(1)
	}
	if *role != "user" && *role != "moderator" && *role != "admin" {
		logger.Error("invalid role", "role", *role)
		os.Exit(1)
	}

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
	defer db.Close()

	hasher := auth.NewPasswordHasher(auth.DefaultArgon2Params)
	passwordHash, err := hasher.Hash(password)
	if err != nil {
		logger.Error("hash password failed", "error", err)
		os.Exit(1)
	}
	account, err := user.NewMySQLRepository(db).Create(context.Background(), user.CreateParams{
		Username: usernameValue, Email: emailValue, PasswordHash: passwordHash, RoleCode: *role,
	})
	if errors.Is(err, user.ErrDuplicate) {
		logger.Error("username or email already exists")
		os.Exit(1)
	}
	if err != nil {
		logger.Error("create account failed", "error", err)
		os.Exit(1)
	}
	logger.Info("account created", "user_id", account.ID, "username", account.Username, "role", *role)
}
