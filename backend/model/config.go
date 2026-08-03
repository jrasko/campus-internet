package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sethvargo/go-envconfig"
)

type Configuration struct {
	HMACSecret    string `env:"HMAC_SECRET,required"`
	LoginFilepath string `env:"USER_FILE_PATH,default=login-users.json"`
	Users         []LoginUser

	DBHost     string `env:"POSTGRES_HOST,default=dhcp_db"`
	DBDatabase string `env:"POSTGRES_DB,default=postgres"`
	DBUser     string `env:"POSTGRES_USER,default=postgres"`
	DBPassword string `env:"POSTGRES_PASSWORD,required"`

	CIDR                 string `env:"CIDR,required"`
	OutputFilepath       string `env:"OUTPUT_FILE,default=whitelist.json"`
	SkipDhcpNotification bool   `env:"SKIP_DHCP_RELOAD,default=false"`
}

func (c Configuration) DSN() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable",
		c.DBHost,
		c.DBUser,
		c.DBPassword,
		c.DBDatabase,
	)
}

func LoadConfig(ctx context.Context) (Configuration, error) {
	var config Configuration
	err := envconfig.Process(ctx, &config)
	if err != nil {
		return Configuration{}, fmt.Errorf("when reading configuration: %w", err)
	}
	config.Users, err = LoadUsers(config.LoginFilepath)
	if err != nil {
		return Configuration{}, fmt.Errorf("when loading users: %w", err)
	}

	return config, nil
}

func LoadUsers(path string) ([]LoginUser, error) {
	file, err := os.OpenFile(filepath.Clean(path), os.O_RDONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("could not open users file: %w", err)
	}
	defer file.Close()

	var users []LoginUser
	err = json.NewDecoder(file).Decode(&users)
	if errors.Is(err, io.EOF) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("decoding json: %w", err)
	}

	for _, loginUser := range users {
		if err = loginUser.Validate(); err != nil {
			return nil, fmt.Errorf("validating user: %w", err)
		}
	}

	return users, nil
}
