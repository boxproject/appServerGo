// Copyright 2018. box.la authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package db

import (
	"database/sql"
	"github.com/boxproject/appServerGo/config"
	"github.com/go-sql-driver/mysql"
)

var Conns *sql.DB
var cfg = config.GetConfig()

func Start(config config.Config) error {
	var err error
	dbOpts := mysql.Config{
		User:      config.Database.User,
		Passwd:    config.Database.Password,
		DBName:    config.Database.DbName,
		Net:       "tcp",
		Addr:      config.Database.Host,
		ParseTime: true,
		Params:    map[string]string{"charset": "utf8"},
	}
	Conns, err = sql.Open("mysql", dbOpts.FormatDSN())

	if err != nil {
		return err
	}

	Conns.SetMaxOpenConns(config.Database.MaxOpen)
	Conns.SetMaxIdleConns(config.Database.MaxIdle)
	//db.SetConnMaxLifetime(-1)
	//if err = Conns.Ping(); err != nil {
	//	dbOpts.DBName = ""
	//	Conns, err = sql.Open("mysql", dbOpts.FormatDSN())
	//	if err != nil {
	//		Conns.Close()
	//		return err
	//	}
	//	InitDBStruct()
	//}

	return nil
}

func Stop() error {
	if Conns != nil {
		return Conns.Close()
	} else {
		//logger.Info("DBManager unstarted yet!")
		return nil
	}
}
