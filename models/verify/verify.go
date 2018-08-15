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
package verify

import (
	"database/sql"
	"github.com/boxproject/appServerGo/db"
	"encoding/base64"
	"crypto/sha256"
	"crypto/x509"
	"crypto/rsa"
	"crypto"
	"github.com/boxproject/appServerGo/models"
	log "github.com/alecthomas/log4go"
	"github.com/boxproject/appServerGo/utils"
	"time"
	"strconv"
)

// 是否提交过相同的注册申请
func HasApplyedRegistration(appid string, captainid string) (bool, error) {
	var regid string
	err := db.Conns.QueryRow("SELECT regID FROM tb_registration_history WHERE applyer = ? and captain = ? and isDeleted = 0", appid, captainid).Scan(&regid)

	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// 检测账号是否为私钥APP
func AdminAcc(appid string) (bool, error) {
	var regid string
	err := db.Conns.QueryRow("select id from tb_accounts_info where appAccountID = ?", appid).Scan(&regid)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, err
	}

	return false, nil
}

// 检测用户名是否存在
func AccExist(account string) (bool, error) {
	var accid string
	err := db.Conns.QueryRow("select acc.id from tb_accounts_info acc left join tb_registration_history rh on rh.applyerAcc = acc.account where acc.account = ? and rh.consent = 2 ", account).Scan(&accid)
	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

// 验证签名值
func SignInfo(msg, pubkey, signature string) (bool, error) {
	bSignData, err := base64.StdEncoding.DecodeString(signature)
	hashed := sha256.Sum256([]byte(msg))
	bKey, err := base64.RawStdEncoding.DecodeString(pubkey)
	pub, err := x509.ParsePKCS1PublicKey(bKey)
	err = rsa.VerifyPKCS1v15(pub, crypto.SHA256, hashed[:], bSignData)
	if err != nil {
		return false, err
	}
	return true, nil
}

// 验证转账信息中from和to是否相等
func FromIsTo(to string) (bool, error) {
	var address string
	row := db.Conns.QueryRow("select address from tb_currency where address = ?", to)

	err := row.Scan(&address)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// 验证部门是否存在
func BranchExists(name string, creater_id int64)(bool, error) {
	var bName string
	row := db.Conns.QueryRow("select name from tb_department where name = ? AND createdBy = ?", name, creater_id)

	err := row.Scan(&bName)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return true, err
	}
	return true, nil
}

// 校验账号
func ValidateUser(appid string ) ( models.AccountInfo, int, map[string]string )  {
	accInfo, err := models.GetAccDetailByAppID(appid)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.AccountInfo{}, 1004, nil
		}
		log.Error("校验账号信息", err)
		return models.AccountInfo{}, 1000, nil
	}

	if accInfo.Departured == true {
		return models.AccountInfo{}, 1011, nil
	}
	if accInfo.Frozen == true {
		nowTime := time.Now().Unix()
		timeStep := accInfo.FrozenTo-nowTime
		if timeStep <= 0 {
			// 重置账户状态
			db.Conns.Exec("UPDATE tb_accounts_info SET frozen = ?, attempts = ?, frozenTo = ? WHERE appAccountID = ?", 0, 0, sql.NullString{}, appid)
			accInfo.Frozen = false
			return accInfo, 0, nil
		}else {

		}

		return models.AccountInfo{}, 1018, map[string]string{"frozenTo":strconv.FormatInt(accInfo.FrozenTo, 10)}
	}
	return accInfo, 0, nil
}

// 校验密码
func SignPassword(appid, password string) (bool, error) {
	p := utils.GenHashStr(password)
	var id string
	row := db.Conns.QueryRow("SELECT id FROM tb_accounts_info WHERE appAccountID = ? AND passwords = ?", appid, p)

	err := row.Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

