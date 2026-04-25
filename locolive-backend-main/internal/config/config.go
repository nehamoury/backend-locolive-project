package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	DBDriver             string        `mapstructure:"DB_DRIVER"`
	DBSource             string        `mapstructure:"DB_SOURCE"`
	ServerAddress        string        `mapstructure:"SERVER_ADDRESS"`
	RedisAddress         string        `mapstructure:"REDIS_ADDRESS"`
	TokenSymmetricKey    string        `mapstructure:"JWT_SECRET"`
	AccessTokenDuration  time.Duration `mapstructure:"ACCESS_TOKEN_DURATION"`
	RefreshTokenDuration time.Duration `mapstructure:"REFRESH_TOKEN_DURATION"`
	GoogleClientID       string        `mapstructure:"GOOGLE_CLIENT_ID"`
	GoogleClientSecret   string        `mapstructure:"GOOGLE_CLIENT_SECRET"`
	R2AccountID          string        `mapstructure:"R2_ACCOUNT_ID"`
	R2AccessKey          string        `mapstructure:"R2_ACCESS_KEY"`
	R2SecretKey          string        `mapstructure:"R2_SECRET_KEY"`
	R2BucketName         string        `mapstructure:"R2_BUCKET_NAME"`
	ExpoRedirectURL      string        `mapstructure:"EXPO_REDIRECT_URL"`
	FrontendURL          string        `mapstructure:"FRONTEND_URL"`
	Environment          string        `mapstructure:"ENVIRONMENT"`
	EmailSenderName      string        `mapstructure:"EMAIL_SENDER_NAME"`
	EmailSenderAddress   string        `mapstructure:"EMAIL_SENDER_ADDRESS"`
	EmailSenderPassword  string        `mapstructure:"EMAIL_SENDER_PASSWORD"`
	SMTPHost             string        `mapstructure:"SMTP_HOST"`
	SMTPPort             string        `mapstructure:"SMTP_PORT"`
	// TLS Configuration (optional - for HTTPS)
	TLSCertFile          string        `mapstructure:"TLS_CERT_FILE"`
	TLSKeyFile           string        `mapstructure:"TLS_KEY_FILE"`
	ForceHTTPS           bool          `mapstructure:"FORCE_HTTPS"`
	FirebaseCredentialsPath string     `mapstructure:"FIREBASE_CREDENTIALS_PATH"`
}

func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
