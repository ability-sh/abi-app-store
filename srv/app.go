package srv

import (
	"fmt"
	"regexp"
	"time"

	"github.com/ability-sh/abi-db/client/service"
	"github.com/ability-sh/abi-db/source"
	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/errors"
	"github.com/ability-sh/abi-lib/json"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/oss"
	"github.com/ability-sh/abi-micro/redis"
)

var re_ver, _ = regexp.Compile(`^[0-9]+\.[0-9]+(\.[0-9]+)?(\-[0-9]+)?$`)

func (s *Server) getAppMember(ctx micro.Context, id string, uid string) (*Member, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	u := Member{}

	key_am := fmt.Sprintf("%sam_%s_%s", config.Prefix, id, uid)
	{
		text, err := redis.Get(key_am)
		if err == nil && text != "" {
			err = json.Unmarshal([]byte(text), &u)
			if err == nil {
				return &u, nil
			}
		}
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	text, err := collection.Get(cc, fmt.Sprintf("app/%s/member/%s", id, uid))

	if err != nil {
		return nil, err
	}

	redis.Set(key_am, string(text), time.Duration(config.CacheExpires)*time.Second)

	json.Unmarshal(text, &u)

	return &u, nil
}

func (s *Server) addAppMember(ctx micro.Context, id string, uid string, role string) (*Member, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	member := &Member{Id: uid, Role: role}

	err = collection.PutObject(cc, fmt.Sprintf("app/%s/member/%s", id, uid), member)

	if err != nil {
		return nil, err
	}

	key_cm := fmt.Sprintf("%sam_%s_%s", config.Prefix, id, uid)

	redis.Del(key_cm)

	return member, nil
}

func (s *Server) removeAppMember(ctx micro.Context, id string, uid string) error {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	err = collection.Del(cc, fmt.Sprintf("app/%s/member/%s", id, uid))

	if err != nil {
		return err
	}

	key_cm := fmt.Sprintf("%sam_%s_%s", config.Prefix, id, uid)

	redis.Del(key_cm)

	return nil
}

func (s *Server) getApp(ctx micro.Context, id string) (*App, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	u := App{}

	key_c := fmt.Sprintf("%sa_%s", config.Prefix, id)

	{
		text, err := redis.Get(key_c)
		if err == nil && text != "" {
			err = json.Unmarshal([]byte(text), &u)
			if err == nil {
				return &u, nil
			}
		}
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	text, err := collection.Get(cc, fmt.Sprintf("app/%s/info.json", id))

	if err != nil {
		return nil, err
	}

	redis.Set(key_c, string(text), time.Duration(config.CacheExpires)*time.Second)

	json.Unmarshal(text, &u)

	return &u, nil
}

func (s *Server) AppCreate(ctx micro.Context, task *AppCreateTask) (*App, error) {

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	app := &App{Id: config.NewID(ctx), Info: task.Info}

	err = collection.PutObject(cc, fmt.Sprintf("app/%s/info.json", app.Id), app)

	if err != nil {
		return nil, err
	}

	_, err = s.addAppMember(ctx, app.Id, uid, ROLE_OWNER)

	if err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Server) AppSet(ctx micro.Context, task *AppSetTask) (*App, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER && member.Role != ROLE_READ_WRITE {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	text, err := collection.Exec(cc, `
	(function(){
		var id = ${id};
		var info = ${info};
		var k_info = collection + 'app/' + id + '/info.json';
		var text = get(k_info);
		if(!text) {
			throw 'app does not exist'
		}
		var object = JSON.parse(text);
		if(info) {
			object.info = info
			text = JSON.stringify(object);
			put(k_info,text)
		}
		return text;
	})()
	`, map[string]interface{}{"id": task.Id, "info": task.Info})

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	key_c := fmt.Sprintf("%sa_%s", config.Prefix, task.Id)

	redis.Del(key_c)

	app := &App{}

	json.Unmarshal([]byte(text), &app)

	return app, nil
}

func (s *Server) AppGet(ctx micro.Context, task *AppGetTask) (*App, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER && member.Role != ROLE_READ_WRITE && member.Role != ROLE_READ_ONLY {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	return s.getApp(ctx, task.Id)
}

func (s *Server) AppMemberAdd(ctx micro.Context, task *AppMemberAddTask) (*Member, error) {

	if !re_email.MatchString(task.Email) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter email is incorrect")
	}

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	u, err := s.getUser(ctx, task.Email)

	if err != nil {
		return nil, err
	}

	if u.Id == uid {
		return &Member{Id: uid, Role: ROLE_OWNER}, nil
	}

	return s.addAppMember(ctx, task.Id, u.Id, task.Role)
}

func (s *Server) AppMemberRemove(ctx micro.Context, task *AppMemberAddTask) (interface{}, error) {

	if !re_email.MatchString(task.Email) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter email is incorrect")
	}

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	u, err := s.getUser(ctx, task.Email)

	if err != nil {
		return nil, err
	}

	if u.Id == uid {
		return nil, errors.Errorf(ERRNO_MEMBER, "cannot delete owner member")
	}

	err = s.removeAppMember(ctx, task.Id, u.Id)

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Server) AppVerUp(ctx micro.Context, task *AppVerUpTask) (*AppVerUpResult, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	if task.Ability == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter ability is incorrect")
	}

	if !re_ver.MatchString(task.Ver) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter ver is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER && member.Role != ROLE_READ_WRITE {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	_, err = collection.Get(cc, fmt.Sprintf("app/%s/%s/info.json", task.Id, task.Ver))

	if !IsErrno(err, ERRNO_NOT_FOUND) {
		return nil, errors.Errorf(ERRNO_APP_VER, "The app version already exists and cannot be uploaded")
	}

	ss, err := oss.GetOSS(ctx, SERVICE_OSS)

	if err != nil {
		return nil, err
	}

	u, err := ss.PutSignURL(fmt.Sprintf("app/%s/%s/%s.zip", task.Id, task.Ver, task.Ability), time.Duration(config.AppUpExpires)*time.Second)

	if err != nil {
		return nil, err
	}

	return &AppVerUpResult{Url: u}, nil
}

func (s *Server) AppVerDone(ctx micro.Context, task *AppVerDoneTask) (interface{}, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	if !re_ver.MatchString(task.Ver) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter ver is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER && member.Role != ROLE_READ_WRITE {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	info := map[string]interface{}{}

	dynamic.Each(task.Info, func(key interface{}, value interface{}) bool {
		info[dynamic.StringValue(key, "")] = value
		return true
	})

	info["appid"] = task.Id
	info["ver"] = task.Ver

	_, err = collection.Exec(cc, `
	(function(){
		var id = ${id};
		var info = ${info};
		var ver = ${ver};
		var k_info = collection + 'app/' + id + '/' + ver + '/info.json';
		var text = get(k_info);
		if(text) {
			throw 'The app version already exists and cannot be uploaded'
		}
		text = JSON.stringify(info);
		put(k_info,text)
	})()
	`, map[string]interface{}{"id": task.Id, "info": info, "ver": task.Ver})

	if err != nil {
		return nil, err
	}

	return info, nil
}

func (s *Server) AppVerInfoGet(ctx micro.Context, task *AppVerInfoGetTask) (interface{}, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	if !re_ver.MatchString(task.Ver) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter ver is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER && member.Role != ROLE_READ_WRITE && member.Role != ROLE_READ_ONLY {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	var info interface{} = nil

	key_av := fmt.Sprintf("%sav_%s_%s", config.Prefix, task.Id, task.Ver)

	{
		text, err := redis.Get(key_av)
		if err == nil && text != "" {
			err = json.Unmarshal([]byte(text), &info)
			if err == nil {
				return info, nil
			}
		}
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	text, err := collection.Get(cc, fmt.Sprintf("app/%s/%s/info.json", task.Id, task.Ver))

	if err != nil {
		if err == source.ErrNoSuchKey {
			return nil, errors.Errorf(ERRNO_APP_VER, "App version that doesn't exist")
		}
		return nil, err
	}

	redis.Set(key_av, string(text), time.Duration(config.CacheExpires)*time.Second)

	json.Unmarshal(text, &info)

	return info, nil
}

func (s *Server) AppApprove(ctx micro.Context, task *AppApproveTask) (interface{}, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	if task.ContainerId == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter containerId is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	err = collection.Put(cc, fmt.Sprintf("app/%s/approve/%s", task.Id, task.ContainerId), []byte("{}"))

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{}, nil
}

func (s *Server) AppUnapprove(ctx micro.Context, task *AppUnapproveTask) (interface{}, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	if task.ContainerId == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter containerId is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getAppMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	err = collection.Del(cc, fmt.Sprintf("app/%s/approve/%s", task.Id, task.ContainerId))

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{}, nil
}
