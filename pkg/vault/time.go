package vault

import "time"

func timeNow() int64 {
	return time.Now().UTC().Unix()
}
