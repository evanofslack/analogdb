package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/evanofslack/analogdb"
)

// a day's salt outlives the day so a late request still finds it
const saltTTL = 48 * time.Hour

var _ analogdb.SaltService = (*RDB)(nil)

func saltKey(day string) string {
	return fmt.Sprintf("ui:salt:%s", day)
}

// DailySalt stores salt for the day unless one is already set, then returns the stored salt
func (rdb *RDB) DailySalt(ctx context.Context, day string, salt []byte) ([]byte, error) {
	key := saltKey(day)
	if err := rdb.db.SetNX(ctx, key, salt, saltTTL).Err(); err != nil {
		return nil, err
	}
	return rdb.db.Get(ctx, key).Bytes()
}
