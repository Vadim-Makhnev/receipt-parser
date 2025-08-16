package config

import (
	"errors"
	"os"
)

type IMAPConfig struct {
	Username string
	Password string
	Server   string
}

func NewIMAPConfig() (*IMAPConfig, error) {
	if os.Getenv("IMAP_YA_USERNAME") == "" || os.Getenv("IMAP_YA_PASSWORD") == "" || os.Getenv("IMAP_SERVER") == "" {
		return nil, errors.New("Ошибка при настройке конфига, некоторые поля отсутствуют")
	}

	return &IMAPConfig{
		Username: os.Getenv("IMAP_YA_USERNAME"),
		Password: os.Getenv("IMAP_YA_PASSWORD"),
		Server:   os.Getenv("IMAP_SERVER"),
	}, nil
}
