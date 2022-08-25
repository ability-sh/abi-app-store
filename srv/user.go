package srv

import (
	"fmt"
	"regexp"
	"time"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/errors"
	"github.com/ability-sh/abi-lib/eval"
	"github.com/ability-sh/abi-micro/http"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/redis"
	"github.com/ability-sh/abi-micro/smtp"
)

var re_email, _ = regexp.Compile(`^[a-zA-Z\.\-\_0-9]+@[a-zA-Z0-9\-\.]+$`)

func (s *Server) getUid(ctx micro.Context, token string) (string, error) {

	if token == "" {
		return "", errors.Errorf(ERRNO_LOGIN, "Retry after logging in")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return "", err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return "", err
	}

	id, err := redis.Get(fmt.Sprintf("%st_%s", config.Prefix, token))

	if err != nil || id == "" {
		return "", errors.Errorf(ERRNO_LOGIN, "Retry after logging in")
	}

	return id, nil
}

func (s *Server) getUser(ctx micro.Context, email string) (*User, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	HTTP, err := http.GetHTTPService(ctx, SERVICE_HTTP)

	if err != nil {
		return nil, err
	}

	res, err := HTTP.Request(ctx, "GET").
		SetURL(fmt.Sprintf("%s/get.json", config.UserSvc), map[string]string{"name": email}).
		Send()

	if err != nil {
		return nil, err
	}

	data, err := res.PraseBody()

	if err != nil {
		return nil, err
	}

	errno := dynamic.IntValue(dynamic.Get(data, "errno"), 0)

	if errno != 200 {
		return nil, errors.Errorf(int32(errno), dynamic.StringValue(dynamic.Get(data, "errmsg"), "Internal service error"))
	}

	uid := dynamic.StringValue(dynamic.GetWithKeys(data, []string{"data", "id"}), "")

	return &User{Email: email, Id: uid}, nil
}

func (s *Server) MailSend(ctx micro.Context, task *SendMailTask) (interface{}, error) {

	if !re_email.MatchString(task.Email) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter email is incorrect")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	key_sr := fmt.Sprintf("%ssr_%s", config.Prefix, task.Email)
	key_s := fmt.Sprintf("%ss_%s", config.Prefix, task.Email)

	_, err = redis.Get(key_sr)

	if err == nil {
		return nil, errors.Errorf(ERRNO_AGAIN, "The operation is too frequent, try again later")
	}

	mail, err := smtp.GetSMTPService(ctx, SERVICE_SMTP)

	if err != nil {
		return nil, err
	}

	code := config.NewCode()

	getValue := func(key string) string {
		if key == "code" {
			return code
		}
		return ""
	}

	if config.EmailEnabled {

		err = mail.Send([]string{task.Email}, eval.ParseEval(config.EmailSubject, getValue), eval.ParseEval(config.EmailBody, getValue), config.EmailBodyType)

		if err != nil {
			return nil, err
		}

	}

	err = redis.Set(key_sr, code, time.Duration(config.EmailReExpires)*time.Second)

	if err != nil {
		return nil, err
	}

	redis.Set(key_s, code, time.Duration(config.EmailExpires)*time.Second)

	if err != nil {
		return nil, err
	}

	if config.EmailEnabled {
		return map[string]interface{}{}, nil
	}

	return map[string]interface{}{"code": code}, nil
}

func (s *Server) Login(ctx micro.Context, task *LoginTask) (*LoginResult, error) {

	if !re_email.MatchString(task.Email) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter email is incorrect")
	}

	if task.Code == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter code is incorrect")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	key_sr := fmt.Sprintf("%ssr_%s", config.Prefix, task.Email)
	key_s := fmt.Sprintf("%ss_%s", config.Prefix, task.Email)

	code, err := redis.Get(key_s)

	if err != nil || code == "" || code != task.Code {
		return nil, errors.Errorf(ERRNO_AGAIN, "wrong captcha")
	}

	HTTP, err := http.GetHTTPService(ctx, SERVICE_HTTP)

	if err != nil {
		return nil, err
	}

	res, err := HTTP.Request(ctx, "GET").
		SetURL(fmt.Sprintf("%s/get.json", config.UserSvc), map[string]string{"name": task.Email, "autoCreated": "true"}).
		Send()

	if err != nil {
		return nil, err
	}

	data, err := res.PraseBody()

	if err != nil {
		return nil, err
	}

	errno := dynamic.IntValue(dynamic.Get(data, "errno"), 0)

	if errno != 200 {
		return nil, errors.Errorf(int32(errno), dynamic.StringValue(dynamic.Get(data, "errmsg"), "Internal service error"))
	}

	uid := dynamic.StringValue(dynamic.GetWithKeys(data, []string{"data", "id"}), "")

	u := &User{Email: task.Email, Id: uid}

	token := config.NewToken()

	err = redis.Set(fmt.Sprintf("%st_%s", config.Prefix, token), uid, time.Duration(config.TokenExpires)*time.Second)

	if err != nil {
		return nil, err
	}

	err = redis.Del(key_s)

	if err != nil {
		return nil, err
	}

	err = redis.Del(key_sr)

	if err != nil {
		return nil, err
	}

	return &LoginResult{Token: token, User: u}, nil
}

func (s *Server) UserGet(ctx micro.Context, task *UserGetTask) (*User, error) {

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	HTTP, err := http.GetHTTPService(ctx, SERVICE_HTTP)

	if err != nil {
		return nil, err
	}

	res, err := HTTP.Request(ctx, "GET").
		SetURL(fmt.Sprintf("%s/get.json", config.UserSvc), map[string]string{"id": uid}).
		Send()

	if err != nil {
		return nil, err
	}

	data, err := res.PraseBody()

	if err != nil {
		return nil, err
	}

	errno := dynamic.IntValue(dynamic.Get(data, "errno"), 0)

	if errno != 200 {
		return nil, errors.Errorf(int32(errno), dynamic.StringValue(dynamic.Get(data, "errmsg"), "Internal service error"))
	}

	name := dynamic.StringValue(dynamic.GetWithKeys(data, []string{"data", "name"}), "")

	return &User{Email: name, Id: uid}, nil
}
