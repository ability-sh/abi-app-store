package srv

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/google/uuid"
)

const (
	SERVICE_CONFIG = "abi-app-store"
)

type ConfigService struct {
	name   string
	config interface{}

	Db             string `json:"db"`
	Collection     string `json:"collection"`
	Prefix         string `json:"prefix"`
	CodeLength     int    `json:"code-length"`
	EmailSubject   string `json:"email-subject"`
	EmailBody      string `json:"email-body"`
	EmailBodyType  string `json:"email-body-type"`
	EmailExpires   int    `json:"email-expires"`    //邮件超时时间(秒)
	EmailReExpires int    `json:"email-re-expires"` //重发邮件间隔时间(秒)
	TokenExpires   int    `json:"token-expires"`
	UserSvc        string `json:"user-svc"`
	CacheExpires   int    `json:"cache-expires"`
}

func newConfigService(name string, config interface{}) *ConfigService {
	return &ConfigService{name: name, config: config}
}

/**
* 服务名称
**/
func (s *ConfigService) Name() string {
	return s.name
}

/**
* 服务配置
**/
func (s *ConfigService) Config() interface{} {
	return s.config
}

/**
* 初始化服务
**/
func (s *ConfigService) OnInit(ctx micro.Context) error {

	dynamic.SetValue(s, s.config)

	rand.Seed(time.Now().UnixNano())

	if s.CodeLength <= 0 {
		s.CodeLength = 6
	}

	if s.EmailSubject == "" {
		s.EmailSubject = "${code} is your captcha code"
	}

	if s.EmailBody == "" {
		s.EmailBody = "${code} is your captcha code"
	}

	if s.EmailBodyType == "" {
		s.EmailBodyType = "text/plain"
	}

	if s.EmailExpires <= 0 {
		s.EmailExpires = 300
	}

	if s.EmailReExpires <= 0 {
		s.EmailReExpires = 60
	}

	if s.TokenExpires <= 0 {
		s.TokenExpires = 30 * 24 * 3600
	}

	if s.CacheExpires <= 0 {
		s.CacheExpires = 300
	}

	return nil
}

/**
* 校验服务是否可用
**/
func (s *ConfigService) OnValid(ctx micro.Context) error {
	return nil
}

func (s *ConfigService) Recycle() {

}

func (s *ConfigService) NewID(ctx micro.Context) string {
	return strconv.FormatInt(ctx.Runtime().NewID(), 36)
}

func (s *ConfigService) NewSecret() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func (s *ConfigService) NewToken() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func (s *ConfigService) NewCode() string {

	ss := strconv.Itoa(rand.Int())

	for len(ss) < s.CodeLength {
		ss = ss + strconv.Itoa(rand.Int())
	}

	return ss[0:s.CodeLength]
}

func (s *ConfigService) Sign(secret string, data map[string]interface{}) string {

	m := md5.New()

	keys := []string{}

	for key, _ := range data {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	op_and := []byte("&")
	op_eq := []byte("=")

	for i, key := range keys {
		if i != 0 {
			m.Write(op_and)
		}
		m.Write([]byte(key))
		m.Write(op_eq)
		m.Write([]byte(dynamic.StringValue(data[key], "")))
	}

	m.Write(op_and)
	m.Write([]byte(secret))

	return hex.EncodeToString(m.Sum(nil))
}

func GetConfigService(ctx micro.Context, name string) (*ConfigService, error) {
	s, err := ctx.GetService(name)
	if err != nil {
		return nil, err
	}
	ss, ok := s.(*ConfigService)
	if ok {
		return ss, nil
	}
	return nil, fmt.Errorf("service %s not instanceof *ConfigService", name)
}

func init() {
	micro.Reg(SERVICE_CONFIG, func(name string, config interface{}) (micro.Service, error) {
		return newConfigService(name, config), nil
	})
}
