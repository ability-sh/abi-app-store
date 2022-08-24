package srv

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/ability-sh/abi-db/client/service"
	"github.com/ability-sh/abi-lib/errors"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/micro"
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
	var id = ${id};
	var info = ${info};
	var secret = ${secret};
	var k_meta = collection + 'container/' + id + '/meta.json';
	var text = get(k_meta);
	if(!text) {
		throw 'container does not exist'
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
	}
	return text;
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

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	container, err := s.getContainer(ctx, task.Id)

	if err != nil {
		return nil, err
	}

	ss := config.Sign(container.Secret, map[string]interface{}{"id": task.Id, "timestamp": task.Timestamp, "ver": task.Ver})

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
