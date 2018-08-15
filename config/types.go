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
package config

type Config struct {
	Server   serverInfo
	Agent    agentInfo
	Database DBInfo
	Finance  financeInfo
	LogPath  string
	LangPath string
}

type serverInfo struct {
	Port       string
	APIVersion string
	Mode 	   string
}

type agentInfo struct {
	Server              string `json:"server"`
	AddFlow             string `json:"addFlow"`
	ApplyTx             string `json:"applyTx"`
	Registration        string `json:"registration"`
	CoinList            string `json:"coinList"`
	TokenList           string `json:"tokenList"`
	TokenDepositAddress string `json:"tokenDepositAddress"`
	FlowStatus          string `json:"flowStatus"`
	ApprovaledInfos     string `json:"approvaledInfos"`
	DisallowFlow 		string `json:"disallowflow"`
	FlowOpLog 			string `json:"flowoplog"`
}

type DBInfo struct {
	User     string
	Password string
	DbName   string
	Host     string
	MaxOpen  int
	MaxIdle  int
	Enabled  bool
}

type financeInfo struct {
	Fixed int
}
