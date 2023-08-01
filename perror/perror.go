package perror

import "fmt"

var (
	ErrOverMaxRetryTimes = fmt.Errorf("Over max retry times still error")
	ErrUnauthorized      = fmt.Errorf("Unauthorized")
	ErrNotFound          = fmt.Errorf("Not Found")
)
