package serverlist

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/apolloconfig/agollo/v4/env"
	"github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/perror"
	"github.com/apolloconfig/agollo/v4/protocol/http"
)

// CheckSecretOK 检查秘钥是否正确
func CheckSecretOK(ctx context.Context, appConfigFunc func() config.AppConfig) (err error) {
	if appConfigFunc == nil {
		return fmt.Errorf("AppConfig is not found, please check")
	}

	appConfig := appConfigFunc()
	c := &env.ConnectConfig{
		AppID:  appConfig.AppID,
		Secret: appConfig.Secret,
	}
	if appConfigFunc().SyncServerTimeout > 0 {
		duration, err := time.ParseDuration(strconv.Itoa(appConfigFunc().SyncServerTimeout) + "s")
		if err != nil {
			return err
		}
		c.Timeout = duration
	}
	if _, err = http.Request(ctx, appConfig.GetCheckSecretURL(), c, nil); err != nil {
		switch err {
		case perror.ErrOverMaxRetryTimes:
			return fmt.Errorf("failed to check Apollo-Secret. err: %v", err)
		case perror.ErrUnauthorized:
			return fmt.Errorf("invalid Apollo-Secret, please check. AppID: %s, Cluster: %s", appConfig.AppID, appConfig.Cluster)
		default:
			return
		}
	}
	return
}
