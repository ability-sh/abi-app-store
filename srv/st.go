package srv

import (
	"context"
	"fmt"

	pb_app "github.com/ability-sh/abi-micro-app/pb"
	"github.com/ability-sh/abi-micro/grpc"
	"github.com/ability-sh/abi-micro/micro"
	"github.com/ability-sh/abi-micro/mongodb"
	"github.com/ability-sh/abi-micro/redis"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type St struct {
}

func (s *Server) StNil(ctx micro.Context, task *St) (*St, error) {
	return &St{}, nil
}

func (s *Server) StRedis(ctx micro.Context, task *St) (*St, error) {

	cache, err := redis.GetRedis(ctx, SERVICE_REDIS)

	if err != nil {
		return nil, err
	}

	_, err = cache.Get("test")

	if !redis.IsNil(err) {
		return nil, err
	}

	return &St{}, nil
}

func (s *Server) StMongo(ctx micro.Context, task *St) (*St, error) {

	config, err := GetConfigService(ctx, SERVICE_CONFIG)

	if err != nil {
		return nil, err
	}

	db, err := mongodb.GetDB(ctx, SERVICE_MONGODB)

	if err != nil {
		return nil, err
	}

	db_my_app := db.Collection(fmt.Sprintf("%smy_app", config.Prefix))

	c := context.Background()

	var rs bson.M

	err = db_my_app.FindOne(c,
		bson.D{bson.E{"appid", '1'}, bson.E{"uid", '1'}}).Decode(&rs)

	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	return &St{}, nil
}

func (s *Server) StGrpc(ctx micro.Context, task *St) (*St, error) {

	conn, err := grpc.GetConn(ctx, SERVICE_APP)

	if err != nil {
		return nil, err
	}

	cli := pb_app.NewServiceClient(conn)

	c := grpc.NewGRPCContext(ctx)

	_, err = cli.VerGetURL(c, &pb_app.VerGetURLTask{})

	if err != nil {
		return nil, err
	}

	return &St{}, nil
}
