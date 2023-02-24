package unrelation

import (
	"OpenIM/pkg/common/config"
	"OpenIM/pkg/common/db/table/unrelation"
	"OpenIM/pkg/utils"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"strings"
	"time"
)

//func NewMongo() *Mongo {
//	mgo := &Mongo{}
//	mgo.InitMongo()
//	return mgo
//}

func NewMongo() (*Mongo, error) {
	uri := "mongodb://sample.host:27017/?maxPoolSize=20&w=majority"
	if config.Config.Mongo.DBUri != "" {
		// example: mongodb://$user:$password@mongo1.mongo:27017,mongo2.mongo:27017,mongo3.mongo:27017/$DBDatabase/?replicaSet=rs0&readPreference=secondary&authSource=admin&maxPoolSize=$DBMaxPoolSize
		uri = config.Config.Mongo.DBUri
	} else {
		//mongodb://mongodb1.example.com:27317,mongodb2.example.com:27017/?replicaSet=mySet&authSource=authDB
		mongodbHosts := ""
		for i, v := range config.Config.Mongo.DBAddress {
			if i == len(config.Config.Mongo.DBAddress)-1 {
				mongodbHosts += v
			} else {
				mongodbHosts += v + ","
			}
		}
		if config.Config.Mongo.DBPassword != "" && config.Config.Mongo.DBUserName != "" {
			uri = fmt.Sprintf("mongodb://%s:%s@%s/%s?maxPoolSize=%d&authSource=admin",
				config.Config.Mongo.DBUserName, config.Config.Mongo.DBPassword, mongodbHosts,
				config.Config.Mongo.DBDatabase, config.Config.Mongo.DBMaxPoolSize)
		} else {
			uri = fmt.Sprintf("mongodb://%s/%s/?maxPoolSize=%d&authSource=admin",
				mongodbHosts, config.Config.Mongo.DBDatabase,
				config.Config.Mongo.DBMaxPoolSize)
		}
	}
	fmt.Println("mongo:", uri)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	return &Mongo{db: mongoClient}, nil
}

type Mongo struct {
	db *mongo.Client
}

func (m *Mongo) InitMongo() {
	uri := "mongodb://sample.host:27017/?maxPoolSize=20&w=majority"
	if config.Config.Mongo.DBUri != "" {
		// example: mongodb://$user:$password@mongo1.mongo:27017,mongo2.mongo:27017,mongo3.mongo:27017/$DBDatabase/?replicaSet=rs0&readPreference=secondary&authSource=admin&maxPoolSize=$DBMaxPoolSize
		uri = config.Config.Mongo.DBUri
	} else {
		//mongodb://mongodb1.example.com:27317,mongodb2.example.com:27017/?replicaSet=mySet&authSource=authDB
		mongodbHosts := ""
		for i, v := range config.Config.Mongo.DBAddress {
			if i == len(config.Config.Mongo.DBAddress)-1 {
				mongodbHosts += v
			} else {
				mongodbHosts += v + ","
			}
		}
		if config.Config.Mongo.DBPassword != "" && config.Config.Mongo.DBUserName != "" {
			// clientOpts := options.Client().ApplyURI("mongodb://localhost:27017,localhost:27018/?replicaSet=replset")
			//mongodb://[username:password@]host1[:port1][,...hostN[:portN]][/[defaultauthdb][?options]]
			//uri = fmt.Sprintf("mongodb://%s:%s@%s/%s?maxPoolSize=%d&authSource=admin&replicaSet=replset",
			uri = fmt.Sprintf("mongodb://%s:%s@%s/%s?maxPoolSize=%d&authSource=admin",
				config.Config.Mongo.DBUserName, config.Config.Mongo.DBPassword, mongodbHosts,
				config.Config.Mongo.DBDatabase, config.Config.Mongo.DBMaxPoolSize)
		} else {
			uri = fmt.Sprintf("mongodb://%s/%s/?maxPoolSize=%d&authSource=admin",
				mongodbHosts, config.Config.Mongo.DBDatabase,
				config.Config.Mongo.DBMaxPoolSize)
		}
	}
	fmt.Println(utils.GetFuncName(1), "start to init mongoDB:", uri)
	mongoClient, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		time.Sleep(time.Duration(30) * time.Second)
		mongoClient, err = mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
		if err != nil {
			panic(err.Error() + " mongo.Connect failed " + uri)
		}
	}
	m.db = mongoClient
}

func (m *Mongo) GetClient() *mongo.Client {
	return m.db
}

func (m *Mongo) CreateMsgIndex() {
	if err := m.createMongoIndex(unrelation.CChat, false, "uid"); err != nil {
		fmt.Println(err.Error() + " index create failed " + unrelation.CChat + " uid, please create index by yourself in field uid")
	}
}

func (m *Mongo) CreateSuperGroupIndex() {
	if err := m.createMongoIndex(unrelation.CSuperGroup, true, "group_id"); err != nil {
		panic(err.Error() + "index create failed " + unrelation.CSuperGroup + " group_id")
	}
	if err := m.createMongoIndex(unrelation.CUserToSuperGroup, true, "user_id"); err != nil {
		panic(err.Error() + "index create failed " + unrelation.CUserToSuperGroup + "user_id")
	}
}

func (m *Mongo) CreateExtendMsgSetIndex() {
	if err := m.createMongoIndex(unrelation.CExtendMsgSet, true, "-create_time", "work_moment_id"); err != nil {
		panic(err.Error() + "index create failed " + unrelation.CExtendMsgSet + " -create_time, work_moment_id")
	}
}

func (m *Mongo) createMongoIndex(collection string, isUnique bool, keys ...string) error {
	db := m.db.Database(config.Config.Mongo.DBDatabase).Collection(collection)
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)
	indexView := db.Indexes()
	keysDoc := bsonx.Doc{}
	// create composite indexes
	for _, key := range keys {
		if strings.HasPrefix(key, "-") {
			keysDoc = keysDoc.Append(strings.TrimLeft(key, "-"), bsonx.Int32(-1))
		} else {
			keysDoc = keysDoc.Append(key, bsonx.Int32(1))
		}
	}
	// create index
	index := mongo.IndexModel{
		Keys: keysDoc,
	}
	if isUnique == true {
		index.Options = options.Index().SetUnique(true)
	}
	result, err := indexView.CreateOne(
		context.Background(),
		index,
		opts,
	)
	if err != nil {
		return utils.Wrap(err, result)
	}
	return nil
}

func MongoTransaction(ctx context.Context, mgo *mongo.Client, fn func(ctx mongo.SessionContext) error) error {
	sess, err := mgo.StartSession()
	if err != nil {
		return err
	}
	sCtx := mongo.NewSessionContext(ctx, sess)
	defer sess.EndSession(sCtx)
	if err := fn(sCtx); err != nil {
		_ = sess.AbortTransaction(sCtx)
		return err
	}
	return utils.Wrap(sess.CommitTransaction(sCtx), "")
}

func getTxCtx(ctx context.Context, tx []any) context.Context {
	if len(tx) > 0 {
		if ctx, ok := tx[0].(mongo.SessionContext); ok {
			return ctx
		}
	}
	return ctx
}
