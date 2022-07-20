package srv

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	SERVICE_CONFIG = "uv-app-store"
)

type ConfigService struct {
	name   string
	config interface{}

	allows []string

	Prefix       string `json:"prefix"`
	Db           string `json:"db"`
	CodeLength   int    `json:"codeLength"`
	CodeExpires  int    `json:"codeEpxires"`
	TokenExpires int    `json:"tokenEpxires"`
	UpExpires    int    `json:"upExpires"` // 上传超时秒数
	Allow        string `json:"allow"`
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

	if s.CodeLength == 0 {
		s.CodeLength = 4
	}

	if s.Allow == "" {
		s.allows = []string{"*"}
	} else {
		s.allows = strings.Split(s.Allow, ",")
	}

	ctx.Printf("db init ...")

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return err
	}

	c := context.Background()

	db_my_app := db.Collection(fmt.Sprintf("%smy_app", s.Prefix))
	db_app_member := db.Collection(fmt.Sprintf("%sapp_member", s.Prefix))

	{
		indexes := db_my_app.Indexes()
		_, err = indexes.CreateMany(c, []mongo.IndexModel{
			{
				Keys: bson.D{bson.E{"ctime", -1}},
			},
			{
				Keys:    bson.D{bson.E{"appid", -1}, bson.E{"uid", -1}},
				Options: options.Index().SetUnique(true),
			},
		})
		if err != nil {
			return err
		}
	}

	{
		indexes := db_app_member.Indexes()
		_, err = indexes.CreateMany(c, []mongo.IndexModel{
			{
				Keys:    bson.D{bson.E{"appid", -1}, bson.E{"uid", -1}},
				Options: options.Index().SetUnique(true),
			},
			{
				Keys: bson.D{bson.E{"ctime", -1}},
			},
		})
		if err != nil {
			return err
		}
	}

	ctx.Printf("db init done")

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

func (s *ConfigService) NewCode() string {
	v := strconv.FormatInt(rand.Int63(), 36)
	for len(v) < s.CodeLength {
		v = strings.Repeat(v, s.CodeLength)
	}
	return v[0:s.CodeLength]
}

func (s *ConfigService) NewToken() string {
	return micro.NewTrace()
}

func (s *ConfigService) IsAllow(email string) bool {

	for _, v := range s.allows {
		if v == "*" {
			return true
		}
		if strings.HasSuffix(email, v) {
			return true
		}
	}

	return false
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
	micro.Reg("uv-app-store", func(name string, config interface{}) (micro.Service, error) {
		return newConfigService(name, config), nil
	})
}
