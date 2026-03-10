package systemd

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/dbus"
)

type Client struct {
	conn *dbus.Conn
}

func New() (*Client, error) {
	conn, err := dbus.NewSystemdConnection()
	if err != nil {
		return nil, fmt.Errorf("connect to systemd: %w", err)
	}
	return &Client{conn: conn}, nil
}

func (c *Client) Close() error {
	c.conn.Close()
	return nil
}

func (c *Client) IsActive(ctx context.Context, unit string) (bool, error) {
	props, err := c.conn.GetUnitPropertiesContext(ctx, unit)
	if err != nil {
		return false, fmt.Errorf("get unit properties: %w", err)
	}
	active, _ := props["ActiveState"].(string)
	return active == "active", nil
}
