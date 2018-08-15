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
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/boxproject/appServerGo/routers"
	"github.com/boxproject/appServerGo/config"
	"gopkg.in/urfave/cli.v1"
	"fmt"
	"os"
	"os/exec"
	"github.com/boxproject/appServerGo/db"
)

func main() {
	app := newApp()
	app.Run(os.Args)
}

func newApp() *cli.App {
	app := cli.NewApp()
	//app.Version = PrintVersion(gitCommit, stage, version)
	app.Name = "Blockchain agent"
	app.Usage = "The blockchain monitor command line interface"
	app.Author = "2SE Group"
	app.Copyright = "Copyright 2017-2018 The exchange Authors"
	app.Email = "support@2se.com"
	app.Description = "blockchain agent"

	app.Commands = []cli.Command{
		// 启动
		{
			Name:   "start",
			Usage:  "start the monitor",
			Action: StartCmd,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "config,c",
					Usage: "Path of the config.json file",
					Value: "",
				},
				cli.StringFlag{
					Name:  "block-file,b",
					Usage: "Check point block number",
					Value: "",
				},
			},
		},
		// 停止
		{
			Name:   "stop",
			Usage:  "stop the monitor",
			Action: StopCmd,
			Flags:  []cli.Flag{},
		},
	}

	return app
}

func PrintVersion(gitCommit, stage, version string) string {
	if gitCommit != "" {
		return fmt.Sprintf("%s-%s-%s", stage, version, gitCommit)
	}
	return fmt.Sprintf("%s-%s", stage, version)
}

func StopCmd(_ *cli.Context) error {
	_, err := exec.Command("sh", "-c", "pkill -SIGINT agent").Output()
	return err
}

func StartCmd(c *cli.Context) error {
	// 读取配置文件
	cfg, err := config.LoadConfig(c.String("c"), "config/config.toml")

	// init logger
	config.InitLogger()

	if err != nil {
		return err
	}
	// 初始化数据库
	if err = db.Start(*cfg); err != nil {
		return err
	}

	// 初始化路由
	gin.SetMode(cfg.Server.Mode)
	router := routers.InitRouter()
	router.Run(cfg.Server.Port)

	return nil
}


