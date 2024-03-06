// MIT License

// Copyright (c) The RAI Authors

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package dbdrivers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/retail-ai-inc/bean/v2/aes"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/gorm"
)

type MongoConfig struct {
	Master *struct {
		Database string
		Username string
		Password string
		Host     string
		Port     string
	}
	ConnectTimeout        time.Duration
	MaxConnectionPoolSize uint64
	MaxConnectionLifeTime time.Duration
	Debug                 bool
}

// Init the mongo database connection map.
func InitMongoTenantConns(config MongoConfig, master *gorm.DB, tenantAlterDbHostParam, tenantDBPassPhraseKey string) (map[uint64]*mongo.Client, map[uint64]string) {

	tenantCfgs := GetAllTenantCfgs(master)

	return getAllMongoTenantDB(config, tenantCfgs, tenantAlterDbHostParam, tenantDBPassPhraseKey)
}

func InitMongoMasterConn(config MongoConfig) (*mongo.Client, string) {

	masterCfg := config.Master
	if masterCfg != nil && masterCfg.Database != "" {
		return connectMongoDB(masterCfg.Username, masterCfg.Password, masterCfg.Host, masterCfg.Port, masterCfg.Database,
			config.MaxConnectionPoolSize, config.ConnectTimeout, config.MaxConnectionLifeTime, config.Debug)
	}

	return nil, ""
}

func getAllMongoTenantDB(config MongoConfig, tenantCfgs []*TenantConnections, tenantAlterDbHostParam, tenantDBPassPhraseKey string) (map[uint64]*mongo.Client, map[uint64]string) {

	mongoConns := make(map[uint64]*mongo.Client, len(tenantCfgs))
	mongoDBNames := make(map[uint64]string, len(tenantCfgs))

	for _, t := range tenantCfgs {

		var cfgsMap map[string]map[string]interface{}
		var err error
		if t.Connections != nil {
			if err = json.Unmarshal(t.Connections, &cfgsMap); err != nil {
				panic(err)
			}
		}

		// IMPORTANT: Check the `mongodb` object exist in the Connections column or not.
		if mongoCfg, ok := cfgsMap["mongodb"]; ok {
			userName := mongoCfg["username"].(string)
			password := mongoCfg["password"].(string)

			// IMPORTANT: If tenant database password is encrypted in master db config.
			if tenantDBPassPhraseKey != "" {
				password, err = aes.BeanAESDecrypt(tenantDBPassPhraseKey, password)
				if err != nil {
					panic(err)
				}
			}

			host := mongoCfg["host"].(string)

			// IMPORTANT - If a command or service wants to use a different `host` parameter for tenant database connection
			// then it's easy to do just by passing that parameter string name using `bean.TenantAlterDbHostParam`.
			// Therfore, `bean` will overwrite all host string in `TenantConnections`.`Connections` JSON.
			if tenantAlterDbHostParam != "" && mongoCfg[tenantAlterDbHostParam] != nil {
				host = mongoCfg[tenantAlterDbHostParam].(string)
			}

			port := mongoCfg["port"].(string)
			dbName := mongoCfg["database"].(string)

			mongoConns[t.TenantID], mongoDBNames[t.TenantID] = connectMongoDB(
				userName, password, host, port, dbName, config.MaxConnectionPoolSize,
				config.ConnectTimeout, config.MaxConnectionLifeTime, config.Debug)

		} else {
			mongoConns[t.TenantID], mongoDBNames[t.TenantID] = nil, ""
		}
	}

	return mongoConns, mongoDBNames
}

func connectMongoDB(userName, password, host, port, dbName string, maxConnectionPoolSize uint64,
	connectTimeout, maxConnectionLifeTime time.Duration, debug bool) (*mongo.Client, string) {

	connStr := "mongodb://" + host + ":" + port

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().
		ApplyURI(connStr).
		SetConnectTimeout(connectTimeout).
		SetMaxPoolSize(maxConnectionPoolSize).
		SetMaxConnIdleTime(maxConnectionLifeTime)

	if userName != "" && password != "" {
		credential := options.Credential{Username: userName, Password: password, AuthSource: dbName}
		opts.SetAuth(credential)
	}

	// log monitor
	var logMonitor = event.CommandMonitor{
		Started: func(ctx context.Context, startedEvent *event.CommandStartedEvent) {
			log.Printf("mongo reqId:%d start on db:%s cmd:%s sql:%+v", startedEvent.RequestID, startedEvent.DatabaseName,
				startedEvent.CommandName, startedEvent.Command)
		},
		Succeeded: func(ctx context.Context, succeededEvent *event.CommandSucceededEvent) {
			log.Printf("mongo reqId:%d exec cmd:%s success duration %d ns", succeededEvent.RequestID,
				succeededEvent.CommandName, succeededEvent.DurationNanos)
		},
		Failed: func(ctx context.Context, failedEvent *event.CommandFailedEvent) {
			log.Printf("mongo reqId:%d exec cmd:%s failed duration %d ns", failedEvent.RequestID,
				failedEvent.CommandName, failedEvent.DurationNanos)
		},
	}
	if debug {
		// cmd monitor set
		opts.SetMonitor(&logMonitor)
	}

	mdb, err := mongo.Connect(ctx, opts)
	if err != nil {
		panic(err)
	}

	// Check the connection
	err = mdb.Ping(ctx, nil)
	if err != nil {
		panic(err)
	}

	return mdb, dbName
}
