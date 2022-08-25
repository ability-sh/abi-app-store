package srv

import (
	"fmt"
	"time"

	"github.com/ability-sh/abi-db/client/service"
	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-lib/errors"
	"github.com/ability-sh/abi-lib/json"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/oss"
	"github.com/ability-sh/abi-micro/redis"
)

func (s *Server) getContainerMember(ctx micro.Context, id string, uid string) (*Member, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	u := Member{}

	key_cm := fmt.Sprintf("%scm_%s_%s", config.Prefix, id, uid)
	{
		text, err := redis.Get(key_cm)
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

	text, err := collection.Get(cc, fmt.Sprintf("container/%s/%s", id, uid))

	if err != nil {
		return nil, err
	}

	redis.Set(key_cm, string(text), time.Duration(config.CacheExpires)*time.Second)

	json.Unmarshal(text, &u)

	return &u, nil
}

func (s *Server) addContainerMember(ctx micro.Context, id string, uid string, role string) (*Member, error) {

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

	err = collection.PutObject(cc, fmt.Sprintf("container/%s/%s", id, uid), member)

	if err != nil {
		return nil, err
	}

	key_cm := fmt.Sprintf("%scm_%s_%s", config.Prefix, id, uid)

	redis.Del(key_cm)

	return member, nil
}

func (s *Server) removeContainerMember(ctx micro.Context, id string, uid string) error {

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

	err = collection.Del(cc, fmt.Sprintf("container/%s/%s", id, uid))

	if err != nil {
		return err
	}

	key_cm := fmt.Sprintf("%scm_%s_%s", config.Prefix, id, uid)

	redis.Del(key_cm)

	return nil
}

func (s *Server) getContainer(ctx micro.Context, id string) (*Container, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	u := Container{}

	key_c := fmt.Sprintf("%sc_%s", config.Prefix, id)

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

	text, err := collection.Get(cc, fmt.Sprintf("container/%s/meta.json", id))

	if err != nil {
		return nil, err
	}

	redis.Set(key_c, string(text), time.Duration(config.CacheExpires)*time.Second)

	json.Unmarshal(text, &u)

	return &u, nil
}

func (s *Server) ContainerCreate(ctx micro.Context, task *ContainerCreateTask) (*Container, error) {

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

	container := &Container{Id: config.NewID(ctx), Secret: config.NewSecret(), Info: task.Info, Ver: 1}

	err = collection.PutObject(cc, fmt.Sprintf("container/%s/meta.json", container.Id), container)

	if err != nil {
		return nil, err
	}

	_, err = s.addContainerMember(ctx, container.Id, uid, ROLE_OWNER)

	if err != nil {
		return nil, err
	}

	return container, nil
}

func (s *Server) ContainerSet(ctx micro.Context, task *ContainerSetTask) (*Container, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getContainerMember(ctx, task.Id, uid)

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

	secret := ""

	if task.Secret {
		secret = config.NewSecret()
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	text, err := collection.Exec(cc, `
	(function(){
		var id = ${id};
		var info = ${info};
		var secret = ${secret};
		var k_meta = collection + 'container/' + id + '/meta.json';
		var text = get(k_meta);
		if(!text) {
			throw 'container does not exist'
		}
		var object = JSON.parse(text);
		if(secret) {
			object.secret = secret;
		}
		object.ver = object.ver + 1;
		if(info) {
			object.info = info
		}
		text = JSON.stringify(object);
		put(k_meta,text)
		return text;
	})()
	`, map[string]interface{}{"id": task.Id, "info": task.Info, "secret": secret})

	if err != nil {
		return nil, err
	}

	redis, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	key_c := fmt.Sprintf("%sc_%s", config.Prefix, task.Id)

	redis.Del(key_c)

	container := &Container{}

	json.Unmarshal([]byte(text), &container)

	return container, nil
}

func (s *Server) ContainerGet(ctx micro.Context, task *ContainerGetTask) (*Container, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	uid, err := s.getUid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	member, err := s.getContainerMember(ctx, task.Id, uid)

	if err != nil {
		return nil, err
	}

	if member.Role != ROLE_OWNER && member.Role != ROLE_READ_WRITE && member.Role != ROLE_READ_ONLY {
		return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
	}

	return s.getContainer(ctx, task.Id)
}

func (s *Server) ContainerInfoGet(ctx micro.Context, task *ContainerInfoGetTask) (*ContainerInfoGetResult, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	if task.Timestamp == 0 {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter timestamp is incorrect")
	}

	if task.Sign == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter sign is incorrect")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	container, err := s.getContainer(ctx, task.Id)

	if err != nil {
		return nil, err
	}

	ss := config.Sign(container.Secret, map[string]interface{}{"id": task.Id, "timestamp": task.Timestamp, "ver": task.Ver})

	ctx.Println("sign", ss)

	if ss != task.Sign {
		return nil, errors.Errorf(ERRNO_SIGN, "Signature error")
	}

	if task.Ver < container.Ver {
		return &ContainerInfoGetResult{Ver: container.Ver, Info: container.Info}, nil
	} else {
		return &ContainerInfoGetResult{Ver: container.Ver}, nil
	}

}

func (s *Server) ContainerMemberAdd(ctx micro.Context, task *ContainerMemberAddTask) (*Member, error) {

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

	member, err := s.getContainerMember(ctx, task.Id, uid)

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

	return s.addContainerMember(ctx, task.Id, u.Id, task.Role)
}

func (s *Server) ContainerMemberRemove(ctx micro.Context, task *ContainerMemberAddTask) (interface{}, error) {

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

	member, err := s.getContainerMember(ctx, task.Id, uid)

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

	err = s.removeContainerMember(ctx, task.Id, u.Id)

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Server) ContainerAppGet(ctx micro.Context, task *ContainerAppGetTask) (*ContainerAppGetResult, error) {

	if task.Id == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter id is incorrect")
	}

	if task.Timestamp == 0 {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter timestamp is incorrect")
	}

	if task.Sign == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter sign is incorrect")
	}

	if task.Appid == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter appid is incorrect")
	}

	if !re_ver.MatchString(task.Ver) {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter ver is incorrect")
	}

	if task.Ability == "" {
		return nil, errors.Errorf(ERRNO_INPUT_DATA, "The parameter ability is incorrect")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	container, err := s.getContainer(ctx, task.Id)

	if err != nil {
		return nil, err
	}

	ss := config.Sign(container.Secret, map[string]interface{}{
		"id":        task.Id,
		"timestamp": task.Timestamp,
		"ver":       task.Ver,
		"appid":     task.Appid,
		"ability":   task.Ability,
	})

	ctx.Println("sign", ss)

	if ss != task.Sign {
		return nil, errors.Errorf(ERRNO_SIGN, "Signature error")
	}

	client, err := service.GetClient(ctx, config.Db)

	if err != nil {
		return nil, err
	}

	cc := grpc.NewGRPCContext(ctx)

	collection := client.Collection(config.Collection)

	_, err = collection.Get(cc, fmt.Sprintf("app/%s/approve/%s", task.Appid, task.Id))

	if err != nil {
		if IsErrno(err, ERRNO_NOT_FOUND) {
			return nil, errors.Errorf(ERRNO_NO_PERMISSION, "No permission")
		}
		return nil, err
	}

	info, err := collection.GetObject(cc, fmt.Sprintf("app/%s/%s/info.json", task.Appid, task.Ver))

	if err != nil {
		if IsErrno(err, ERRNO_NOT_FOUND) {
			return nil, errors.Errorf(ERRNO_NOT_FOUND, "App version that doesn't exist")
		}
		return nil, err
	}

	if dynamic.Get(info, task.Ability) == nil {
		return nil, errors.Errorf(ERRNO_NOT_FOUND, "application package %s that does not exist", task.Ability)
	}

	sss, err := oss.GetOSS(ctx, SERVICE_OSS)

	if err != nil {
		return nil, err
	}

	u, err := sss.GetSignURL(fmt.Sprintf("app/%s/%s/%s.zip", task.Appid, task.Ver, task.Ability), time.Duration(config.AppGetExpires)*time.Second)

	if err != nil {
		return nil, err
	}

	return &ContainerAppGetResult{Info: info, Url: u}, nil

}
