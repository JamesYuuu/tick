package app

import (
	"fmt"
	"time"

	"github.com/JamesYuuu/tick/internal/store"
	"github.com/JamesYuuu/tick/internal/timeutil"
)

type Config struct {
	Store    store.Store
	Clock    timeutil.Clock
	Location *time.Location
}

func (c Config) validate() error {
	if c.Store == nil {
		return fmt.Errorf("config: store is required")
	}
	if c.Clock == nil {
		return fmt.Errorf("config: clock is required")
	}
	if c.Location == nil {
		c.Location = time.UTC
	}
	return nil
}
