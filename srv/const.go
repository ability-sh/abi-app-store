package srv

const (
	ERRNO_OK              = 200
	ERRNO_NOT_FOUND       = 404
	ERRNO_INTERNAL_SERVER = 500
	ERRNO_INPUT_DATA      = 400
	ERRNO_INDEX_VALUE     = 600
	ERRNO_LOGIN           = 601
	ERRNO_AGAIN           = 602
	ERRNO_NO_PERMISSION   = 603
	ERRNO_SIGN            = 604
	ERRNO_MEMBER          = 605
	ERRNO_APP_VER         = 606
)

const (
	SERVICE_SMTP  = "smtp"
	SERVICE_REDIS = "redis"
	SERVICE_HTTP  = "http"
	SERVICE_OSS   = "oss"
)
