package port

import "goyt/internal/domain/model"

type ConfigPort interface {
	LoadCookie() (string, error)
	LoadTheme() (*model.Theme, error)
	LoadNotificationsEnabled() (bool, error)
}
