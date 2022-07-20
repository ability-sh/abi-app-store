package srv

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ability-sh/abi-ac/ac"
	pb_user "github.com/ability-sh/abi-micro-user/pb"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/redis"
	"github.com/ability-sh/abi-micro/smtp"
)

var reg_email, _ = regexp.Compile(`^[a-zA-Z0-9\.\-\_]+@[a-zA-Z0-9\.\-\_]+$`)

func (s *Server) LoginCode(ctx micro.Context, task *LoginCodeTask) (*LoginCodeResult, error) {

	if !reg_email.MatchString(task.Email) {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "email error")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	if !config.IsAllow(task.Email) {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "not support email")
	}

	cache, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	sm, err := smtp.GetSMTPService(ctx, SERVICE_SMTP)

	if err != nil {
		return nil, err
	}

	k := fmt.Sprintf("%slogin_code_%s", config.Prefix, task.Email)

	_, err = cache.Get(k)

	if err == nil {
		return nil, ac.Errorf(ERRNO_LOGIN_CODE, "Too many requests, please try again later")
	}

	code := config.NewCode()

	err = sm.Send([]string{task.Email}, "验证码", strings.ToUpper(code), "text/plain")

	if err != nil {
		return nil, err
	}

	err = cache.Set(k, code, time.Second*time.Duration(config.CodeExpires))

	if err != nil {
		return nil, err
	}

	return &LoginCodeResult{}, nil
}

func (s *Server) Login(ctx micro.Context, task *LoginTask) (*LoginResult, error) {

	if !reg_email.MatchString(task.Email) {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "email error")
	}

	if task.Code == "" {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "code error")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	if !config.IsAllow(task.Email) {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "not support email")
	}

	cache, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	k := fmt.Sprintf("%slogin_code_%s", config.Prefix, task.Email)

	code, err := cache.Get(k)

	if err != nil {
		if redis.IsNil(err) {
			return nil, ac.Errorf(ERRNO_404, "not found code")
		}
		return nil, err
	}

	if strings.ToLower(task.Code) != code {
		return nil, ac.Errorf(ERRNO_LOGIN_CODE, "code error")
	}

	err = cache.Del(k)

	if err != nil {
		return nil, err
	}

	conn, err := grpc.GetConn(ctx, SERVICE_USER)

	if err != nil {
		return nil, err
	}

	cli := pb_user.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	rs, err := cli.UserGet(c, &pb_user.UserGetTask{Name: task.Email, AutoCreated: true})

	if err != nil {
		return nil, err
	}

	if rs.Errno != ERRNO_OK {
		return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
	}

	token := config.NewToken()

	k = fmt.Sprintf("%stoken_%s", config.Prefix, token)

	err = cache.Set(k, rs.Data.Id, time.Duration(config.TokenExpires)*time.Second)

	if err != nil {
		return nil, err
	}

	return &LoginResult{User: &User{Id: rs.Data.Id, Email: rs.Data.Name}, Token: token}, nil
}

func (s *Server) Logout(ctx micro.Context, task *LogoutTask) (*LogoutResult, error) {

	if task.Token == "" {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "token error")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	cache, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	k := fmt.Sprintf("%stoken_%s", config.Prefix, task.Token)

	err = cache.Del(k)

	if err != nil && !redis.IsNil(err) {
		return nil, err
	}

	return &LogoutResult{}, nil
}

func (s *Server) Uid(ctx micro.Context, token string) (string, error) {

	if token == "" {
		return "", ac.Errorf(ERRNO_INPUT_DATA, "token error")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return "", err
	}

	cache, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return "", err
	}

	k := fmt.Sprintf("%stoken_%s", config.Prefix, token)

	uid, err := cache.Get(k)

	if err != nil {
		if redis.IsNil(err) {
			return "", ac.Errorf(ERRNO_NOT_LOGIN, "not login")
		}
		return "", err
	}

	return uid, nil
}

func (s *Server) UserGet(ctx micro.Context, task *UserGetTask) (*User, error) {

	if task.Token == "" {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "token error")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	cache, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	k := fmt.Sprintf("%stoken_%s", config.Prefix, task.Token)

	uid, err := cache.Get(k)

	if err != nil {
		if redis.IsNil(err) {
			return nil, ac.Errorf(ERRNO_NOT_LOGIN, "not login")
		}
		return nil, err
	}

	conn, err := grpc.GetConn(ctx, SERVICE_USER)

	if err != nil {
		return nil, err
	}

	cli := pb_user.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	rs, err := cli.UserGet(c, &pb_user.UserGetTask{Uid: uid})

	if err != nil {
		return nil, err
	}

	if rs.Errno != ERRNO_OK {
		return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
	}

	return &User{Id: rs.Data.Id, Email: rs.Data.Name}, nil
}
