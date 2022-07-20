package srv

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ability-sh/abi-ac/ac"
	"github.com/ability-sh/abi-lib/dynamic"
	pb_app "github.com/ability-sh/abi-micro-app/pb"
	"github.com/ability-sh/abi-micro-user/pb"
	pb_user "github.com/ability-sh/abi-micro-user/pb"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (s *Server) MyAppCreate(ctx micro.Context, task *MyAppCreateTask) (*App, error) {

	if task.Title == "" {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "title error")
	}

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	conn, err := grpc.GetConn(ctx, SERVICE_APP)

	if err != nil {
		return nil, err
	}

	cli := pb_app.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	rs, err := cli.AppCreate(c, &pb_app.AppCreateTask{Title: task.Title})

	if err != nil {
		return nil, err
	}

	if rs.Errno != ERRNO_OK {
		return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
	}

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return nil, err
	}

	db_my_app := db.Collection(fmt.Sprintf("%smy_app", config.Prefix))

	ctime := int32(time.Now().Unix())

	_, err = db_my_app.InsertOne(c,
		bson.D{bson.E{"appid", rs.Data.Id},
			bson.E{"uid", uid},
			bson.E{"title", task.Title},
			bson.E{"ctime", ctime}})

	if err != nil {
		return nil, err
	}

	return &App{Id: rs.Data.Id, Title: rs.Data.Title}, nil
}

func (s *Server) MyAppQuery(ctx micro.Context, task *MyAppQueryTask) (*AppQueryResult, error) {

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return nil, err
	}

	n := task.N
	p := task.P

	if n < 1 {
		n = 20
	}

	c := context.Background()

	db_my_app := db.Collection(fmt.Sprintf("%smy_app", config.Prefix))

	opts := options.Find().SetSort(bson.D{bson.E{"ctime", -1}}).SetLimit(int64(n))

	filter := bson.D{bson.E{"uid", uid}}

	rs := &AppQueryResult{}

	if p > 0 {

		opts = opts.SetSkip(int64(n * (p - 1)))

		totalCount, err := db_my_app.CountDocuments(c, filter)

		if err != nil {
			return nil, err
		}

		count := int32(totalCount) / n

		if int32(totalCount)%n != 0 {
			count = count + 1
		}

		rs.Page = &Page{P: p, N: n, TotalCount: int32(totalCount), Count: count}

	}

	cursor, err := db_my_app.Find(c, filter, opts)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return nil, err
	}

	for _, item := range items {
		rs.Items = append(rs.Items, &App{Id: dynamic.StringValue(dynamic.Get(item, "appid"), ""), Title: dynamic.StringValue(dynamic.Get(item, "title"), "")})
	}

	return rs, nil
}

func (s *Server) isAllow(ctx micro.Context, uid string, appid string, keys []string) ([]string, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return nil, err
	}

	db_my_app := db.Collection(fmt.Sprintf("%smy_app", config.Prefix))
	db_app_member := db.Collection(fmt.Sprintf("%sapp_member", config.Prefix))

	c := context.Background()

	var rs bson.M

	err = db_my_app.FindOne(c,
		bson.D{bson.E{"appid", appid}, bson.E{"uid", uid}}).Decode(&rs)

	if err != nil {

		if err == mongo.ErrNoDocuments {

			err = db_app_member.FindOne(c,
				bson.D{bson.E{"appid", appid}, bson.E{"uid", uid}}).Decode(&rs)

			if err != nil {
				if err == mongo.ErrNoDocuments {
					return []string{}, nil
				}
				return nil, err
			}

			allow := dynamic.StringValue(dynamic.Get(rs, "allow"), "")

			if allow == "" {
				return []string{}, nil
			}

			allowSet := map[string]bool{}

			for _, v := range strings.Split(allow, "|") {
				allowSet[v] = true
			}

			if allowSet[ALLOW_OWN] {
				return keys, nil
			}

			vs := []string{}

			for _, k := range keys {
				if allowSet[k] {
					vs = append(vs, k)
				}
			}

			return vs, nil
		}
		return nil, err
	}

	return keys, nil
}

func (s *Server) AppMemberQuery(ctx micro.Context, task *AppMemberQueryTask) (*AppMemberQueryResult, error) {

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	keys, err := s.isAllow(ctx, uid, task.Appid, []string{ALLOW_OWN})

	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ac.Errorf(ERRNO_NO_PERMISSIONS, "no permissions")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return nil, err
	}

	n := task.N
	p := task.P

	if n < 1 {
		n = 20
	}

	c := context.Background()

	db_app_member := db.Collection(fmt.Sprintf("%sapp_member", config.Prefix))

	opts := options.Find().SetSort(bson.D{bson.E{"ctime", -1}}).SetLimit(int64(n))

	filter := bson.D{bson.E{"appid", task.Appid}}

	rs := &AppMemberQueryResult{}

	if p > 0 {

		opts = opts.SetSkip(int64(n * (p - 1)))

		totalCount, err := db_app_member.CountDocuments(c, filter)

		if err != nil {
			return nil, err
		}

		count := int32(totalCount) / n

		if int32(totalCount)%n != 0 {
			count = count + 1
		}

		rs.Page = &Page{P: p, N: n, TotalCount: int32(totalCount), Count: count}

	}

	cursor, err := db_app_member.Find(c, filter, opts)

	if err != nil {
		return nil, err
	}

	defer cursor.Close(c)

	var items []bson.M

	err = cursor.All(c, &items)

	if err != nil {
		return nil, err
	}

	uids := []string{}

	for _, item := range items {
		uid := dynamic.StringValue(dynamic.Get(item, "uid"), "")
		uids = append(uids, uid)
		rs.Items = append(rs.Items, &AppMember{Uid: uid, Title: dynamic.StringValue(dynamic.Get(item, "title"), ""), Appid: task.Appid})
	}

	if len(uids) > 0 {

		conn, err := grpc.GetConn(ctx, SERVICE_USER)

		if err != nil {
			return nil, err
		}

		cli := pb_user.NewServiceClient(conn)

		c := grpc.NewGRPCContext(ctx)

		rs_u, err := cli.UserBatchGet(c, &pb_user.UserBatchGetTask{Uid: uids})

		if err != nil {
			return nil, err
		}

		if rs_u.Errno != 200 {
			return nil, ac.Errorf(int(rs_u.Errno), rs_u.Errmsg)
		}

		for i, item := range rs.Items {
			u := rs_u.Items[i]
			if u != nil {
				item.Email = u.Name
			}
		}
	}

	return rs, nil
}

func (s *Server) AppMemberAdd(ctx micro.Context, task *AppMemberAddTask) (*AppMember, error) {

	if !reg_email.MatchString(task.Email) {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "email error")
	}

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	keys, err := s.isAllow(ctx, uid, task.Appid, []string{ALLOW_OWN})

	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ac.Errorf(ERRNO_NO_PERMISSIONS, "no permissions")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	conn, err := grpc.GetConn(ctx, SERVICE_USER)

	if err != nil {
		return nil, err
	}

	cli := pb_user.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	rs_u, err := cli.UserGet(c, &pb_user.UserGetTask{Name: task.Email})

	if err != nil {
		return nil, err
	}

	if rs_u.Errno != 200 {
		return nil, ac.Errorf(int(rs_u.Errno), rs_u.Errmsg)
	}

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return nil, err
	}

	db_app_member := db.Collection(fmt.Sprintf("%sapp_member", config.Prefix))

	opts := options.FindOneAndUpdate().SetUpsert(true)

	var rs bson.M

	set := bson.D{}

	if task.Title != "" {
		set = append(set, bson.E{"title", task.Title})
	}

	if task.Allow != "" {
		set = append(set, bson.E{"allow", task.Allow})
	}

	err = db_app_member.FindOneAndUpdate(c,
		bson.D{bson.E{"appid", task.Appid}, bson.E{"uid", rs_u.Data.Id}}, bson.D{bson.E{"$set", set}}, opts).Decode(&rs)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &AppMember{Appid: task.Appid, Uid: rs_u.Data.Id, Title: task.Title, Allow: task.Allow}, nil
		}
		return nil, err
	}

	if task.Title != "" {
		rs["title"] = task.Title
	}

	if task.Allow != "" {
		rs["allow"] = task.Allow
	}

	return &AppMember{
		Appid: task.Appid,
		Uid:   rs_u.Data.Id,
		Title: dynamic.StringValue(dynamic.Get(rs, "title"), ""),
		Allow: dynamic.StringValue(dynamic.Get(rs, "allow"), ""),
		Email: rs_u.Data.Name,
	}, nil
}

func (s *Server) AppMemberRemove(ctx micro.Context, task *AppMemberRemoveTask) (*AppMember, error) {

	if !reg_email.MatchString(task.Email) {
		return nil, ac.Errorf(ERRNO_INPUT_DATA, "email error")
	}

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	keys, err := s.isAllow(ctx, uid, task.Appid, []string{ALLOW_OWN})

	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ac.Errorf(ERRNO_NO_PERMISSIONS, "no permissions")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	conn, err := grpc.GetConn(ctx, SERVICE_USER)

	if err != nil {
		return nil, err
	}

	cli := pb_user.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	rs_u, err := cli.UserGet(c, &pb.UserGetTask{Name: task.Email})

	if err != nil {
		return nil, err
	}

	if rs_u.Errno != 200 {
		return nil, ac.Errorf(int(rs_u.Errno), rs_u.Errmsg)
	}

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return nil, err
	}

	db_app_member := db.Collection(fmt.Sprintf("%sapp_member", config.Prefix))

	var rs bson.M

	err = db_app_member.FindOneAndDelete(c,
		bson.D{bson.E{"appid", task.Appid}, bson.E{"uid", rs_u.Data.Id}}).Decode(&rs)

	if err != nil {
		return nil, err
	}

	return &AppMember{Appid: task.Appid,
		Uid:   rs_u.Data.Id,
		Title: dynamic.StringValue(dynamic.Get(rs, "title"), ""),
		Allow: dynamic.StringValue(dynamic.Get(rs, "allow"), ""),
		Email: rs_u.Data.Name,
	}, nil

}

func (s *Server) AppVerQuery(ctx micro.Context, task *AppVerQueryTask) (*AppVerQueryResult, error) {

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	keys, err := s.isAllow(ctx, uid, task.Appid, []string{ALLOW_DEV})

	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ac.Errorf(ERRNO_NO_PERMISSIONS, "no permissions")
	}

	conn, err := grpc.GetConn(ctx, SERVICE_APP)

	if err != nil {
		return nil, err
	}

	cli := pb_app.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	rs, err := cli.VerQuery(c, &pb_app.VerQueryTask{Appid: task.Appid, P: task.P, N: task.N})

	if err != nil {
		return nil, err
	}

	if rs.Errno != 200 {
		return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
	}

	ret := &AppVerQueryResult{}

	for _, item := range rs.Items {
		ret.Items = append(ret.Items, &AppVer{Appid: item.Appid, Ver: item.Ver, Title: item.Title})
	}

	if rs.Page != nil {
		ret.Page = &Page{P: rs.Page.P, N: rs.Page.N, TotalCount: rs.Page.TotalCount, Count: rs.Page.Count}
	}

	return ret, nil
}

func (s *Server) AppUp(ctx micro.Context, task *AppUpTask) (*AppUpResult, error) {

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	keys, err := s.isAllow(ctx, uid, task.Appid, []string{ALLOW_DEV})

	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ac.Errorf(ERRNO_NO_PERMISSIONS, "no permissions")
	}

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	conn, err := grpc.GetConn(ctx, SERVICE_APP)

	if err != nil {
		return nil, err
	}

	cli := pb_app.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	{
		rs, err := cli.VerGet(c, &pb_app.VerGetTask{Appid: task.Appid, Ver: task.Ver})
		if err != nil {
			return nil, err
		}
		if rs.Errno == 200 {
			if rs.Data.Status == APP_VER_STATE_OK {
				return nil, ac.Errorf(ERRNO_APP_VER, "The app version already exists")
			}
		} else if rs.Errno != 404 {
			return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
		}
	}

	rs, err := cli.VerUpURL(c, &pb_app.VerUpURLTask{Appid: task.Appid, Ver: task.Ver, Ability: task.Ability, Expires: int32(config.UpExpires)})

	if err != nil {
		return nil, err
	}

	if rs.Errno != 200 {
		return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
	}

	return &AppUpResult{Key: rs.Data.Key, Data: rs.Data.Data, Method: rs.Data.Method, Url: rs.Data.Url}, nil

}

func (s *Server) AppUpDone(ctx micro.Context, task *AppUpDoneTask) (*AppUpDoneResult, error) {

	uid, err := s.Uid(ctx, task.Token)

	if err != nil {
		return nil, err
	}

	keys, err := s.isAllow(ctx, uid, task.Appid, []string{ALLOW_DEV})

	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, ac.Errorf(ERRNO_NO_PERMISSIONS, "no permissions")
	}

	conn, err := grpc.GetConn(ctx, SERVICE_APP)

	if err != nil {
		return nil, err
	}

	cli := pb_app.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	rs, err := cli.VerGet(c, &pb_app.VerGetTask{Appid: task.Appid, Ver: task.Ver})
	if err != nil {
		return nil, err
	}

	b_info, _ := json.Marshal(task.Info)

	if rs.Errno == 200 {
		if rs.Data.Status == APP_VER_STATE_OK {
			return nil, ac.Errorf(ERRNO_APP_VER, "The app version already exists")
		}
		rrs, err := cli.VerSet(c, &pb_app.VerSetTask{
			Appid:  task.Appid,
			Ver:    task.Ver,
			Title:  task.Title,
			Status: fmt.Sprintf("%d", APP_VER_STATE_OK),
			Info:   string(b_info),
		})
		if err != nil {
			return nil, err
		}
		if rrs.Errno == 200 {
			return &AppUpDoneResult{}, nil
		} else {
			return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
		}
	} else if rs.Errno != 404 {
		return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
	} else {
		rrs, err := cli.VerCreate(c, &pb_app.VerCreateTask{
			Appid:  task.Appid,
			Ver:    task.Ver,
			Title:  task.Title,
			Status: APP_VER_STATE_OK,
			Info:   string(b_info),
		})
		if err != nil {
			return nil, err
		}
		if rrs.Errno == 200 {
			return &AppUpDoneResult{}, nil
		} else {
			return nil, ac.Errorf(int(rs.Errno), rs.Errmsg)
		}
	}
}
