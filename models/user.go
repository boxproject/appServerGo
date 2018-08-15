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
package models

import (
	"github.com/boxproject/appServerGo/db"
	"fmt"
	"database/sql"
	"math"
	log "github.com/alecthomas/log4go"
	"time"
	"github.com/boxproject/appServerGo/utils"
)

// 根据appID获取账号信息
func GetAccountInfoByAppID(appid string) (accountinfo AccountInfo, err error) {
	res := db.Conns.QueryRow(`SELECT acc.id, acc.account, acc.pubKey, acc.appAccountID, acc.regID, acc.lft, acc.rgt, acc.depth, ifnull(acc.cipherText, ""), acc.isDepartured, acc.branchID, dep.name FROM tb_accounts_info acc left join tb_registration_history rh on rh.applyer = acc.appAccountID left join tb_department as dep on dep.id = acc.branchID WHERE acc.appAccountID = ? and rh.consent = 2`, appid)
	err = res.Scan(&accountinfo.ID, &accountinfo.Account, &accountinfo.Pubkey, &accountinfo.AppID, &accountinfo.RegID, &accountinfo.AccLft, &accountinfo.AccRgt, &accountinfo.Depth, &accountinfo.CipherText, &accountinfo.Departured, &accountinfo.BranchID, &accountinfo.BranchName)
	return
}

// 获取用户账号详情
func GetAccDetailByAppID(appid string) (accountinfo AccountInfo, err error) {
	res := db.Conns.QueryRow(`SELECT acc.id, acc.account, acc.pubKey, acc.appAccountID, acc.regID, acc.lft, acc.rgt, acc.depth, ifnull(acc.cipherText, ""), acc.isDepartured, acc.frozen, acc.attempts, ifnull(UNIX_TIMESTAMP(frozenTo), 0), acc.branchID, dep.name FROM tb_accounts_info acc left join tb_registration_history rh on rh.applyer = acc.appAccountID left join tb_department as dep on dep.id = acc.branchID WHERE acc.appAccountID = ? and rh.consent = 2`, appid)
	err = res.Scan(&accountinfo.ID, &accountinfo.Account, &accountinfo.Pubkey, &accountinfo.AppID, &accountinfo.RegID, &accountinfo.AccLft, &accountinfo.AccRgt, &accountinfo.Depth, &accountinfo.CipherText, &accountinfo.Departured, &accountinfo.Frozen, &accountinfo.Attempts, &accountinfo.FrozenTo, &accountinfo.BranchID, &accountinfo.BranchName)
	return
}

// 记录注册申请
func AddRegistration(regid, applyer, captain, msg, applicant_account, password string) (err error) {
	_, err = db.Conns.Exec("INSERT INTO tb_registration_history (regID, applyer, captain, msg, applyerAcc, passwords) VALUES (?, ?, ?, ?, ?, ?)", regid, applyer, captain, msg, applicant_account, password)
	return
}

// 获取注册申请
func Registration(captain string) ([]RegistrationInfo, error) {
	reglist := []RegistrationInfo{}
	rows, err := db.Conns.Query("select regID, applyer, captain, msg, consent, UNIX_TIMESTAMP(createdAt) as apply_at, applyerAcc from tb_registration_history where captain = ? and isDeleted = 0 order by apply_at DESC ", captain)

	defer rows.Close()

	if err != nil {
		return nil, err
	}

	for rows.Next() {

		if rows.Err() != nil {
			return nil, rows.Err()
		}

		reginfo := RegistrationInfo{}
		err = rows.Scan(&reginfo.RegID, &reginfo.ApplicantID, &reginfo.CaptainID, &reginfo.Msg, &reginfo.Consent, &reginfo.ApplyAt, &reginfo.ApplicantAccount)

		if err != nil {
			return nil, err
		}

		reglist = append(reglist, reginfo)
	}
	return reglist, nil
}

// 删除多余的注册申请记录
func DelRegistrationInfoByDateTime(start, end int64) error {
	res, err := db.Conns.Prepare("update tb_registration_history set consent = ?, isDeleted = ? where UNIX_TIMESTAMP(createdAt) between ? and ?")
	defer res.Close()

	if err != nil {
		return err
	}

	_, err = res.Exec(1, 1, start, end)
	if err != nil {
		return err
	}
	return nil
}

// 根据registration_id获取注册申请
func GetRegistrationByRegID(regid string, is_deleted int) (reginfo RegistrationInfo, err error) {
	var row *sql.Row
	if is_deleted == 0 || is_deleted == 1 {
		row = db.Conns.QueryRow("SELECT id, regID, applyer, captain, msg, consent, applyerAcc FROM tb_registration_history WHERE regID = ? and isDeleted = ?", regid, is_deleted)
	} else {
		row = db.Conns.QueryRow("SELECT id, regID, applyer, captain, msg, consent, applyerAcc FROM tb_registration_history WHERE regID = ? " , regid)
	}

	err = row.Scan(&reginfo.ID, &reginfo.RegID, &reginfo.ApplicantID, &reginfo.CaptainID, &reginfo.Msg, &reginfo.Consent, &reginfo.ApplicantAccount)

	return
}

// 账户对应的注册信息
func GetRegistrationByRegIDWithAcc(regid string) (regInfoWithAcc, error) {
	result := regInfoWithAcc{}
	res := db.Conns.QueryRow(`SELECT rh.id, rh.regID, rh.applyer, rh.captain, rh.msg, rh.consent, ifnull(acc.depth, -1) as depth, rh.applyerAcc, ifnull(acc.cipherText, "") FROM tb_registration_history rh LEFT JOIN tb_accounts_info acc ON acc.regID = rh.regID WHERE rh.regID = ?`, regid)
	err := res.Scan(&result.ID, &result.RegID, &result.ApplicantID, &result.LeaderID, &result.Msg, &result.Consent, &result.Depth, &result.ApplicantAccount, &result.CipherText)
	return result, err
}

// 生成账号
func GenAccount(accountInfo AccountInfo, is_uploaded int) error {
	var max_rgt int64
	var password string
	// 获取用户注册时设置的密码
	db.Conns.QueryRow("SELECT passwords FROM tb_registration_history WHERE regID = ?", accountInfo.RegID).Scan(&password)

	row := db.Conns.QueryRow("select ifnull(max(rgt), 0) as max_rgt from tb_accounts_info")

	if err := row.Scan(&max_rgt); err != nil {
		return err
	}

	if accountInfo.AccRgt == 0 {
		accountInfo.AccRgt = max_rgt + 1
	}
	conn, err := db.Conns.Begin()
	if err != nil {
		return err
	}

	_, err = conn.Exec("update tb_accounts_info set rgt = rgt + 2 where rgt >= ?",accountInfo.AccRgt)
	if err != nil {
		conn.Rollback()
		return err
	}

	_, err = conn.Exec("update tb_accounts_info set lft = lft + 2 where lft > ?", accountInfo.AccRgt)
	if err != nil {
		conn.Rollback()
		return err
	}

	_, err = conn.Exec("insert into tb_accounts_info set account = ?, appAccountID = ?, regID = ?, pubKey = ?, enPubKey = ?, cipherText = ?, lft = ?, rgt = ?, isUploaded = ?, depth = ?, passwords = ?", accountInfo.Account, accountInfo.AppID, accountInfo.RegID, accountInfo.Pubkey, accountInfo.EnPubkey, accountInfo.CipherText, accountInfo.AccRgt, accountInfo.AccRgt+1, is_uploaded, accountInfo.Depth, password)
	if err != nil {
		conn.Rollback()
		return err
	}

	conn.Commit()
	return nil
}

// 记录上级审批注册结果
func CaptainApprovalRegInfo(regid, consent string) error {
	_, err := db.Conns.Exec("update tb_registration_history set consent = ?, isDeleted = ? where regID = ?", consent, 1, regid)
	return err
}

// 删除||替换员工
func ChangeEmployee(appid string, underEmployee []AccountInfo) error {
	elderLeader, err := GetAccountInfoByAppID(appid)

	if err != nil {
		return err
	}

	conn, err := db.Conns.Begin()
	if err != nil {
		return err
	}

	_, err = conn.Exec("update tb_accounts_info set isDepartured = ? where appAccountID = ?", 1, appid)
	if err != nil {
		conn.Rollback()
		return err
	}

	_, err = conn.Exec("update tb_accounts_info set lft = lft -1, rgt = rgt - 1, depth = ? where lft between ? and ?", elderLeader.Depth, elderLeader.AccLft+1, elderLeader.AccRgt-1)
	if err != nil {
		conn.Rollback()
		return err
	}
	_, err = conn.Exec("update tb_accounts_info set rgt = rgt -2 where rgt > ?", elderLeader.AccRgt)
	if err != nil {
		conn.Rollback()
		return err
	}
	_, err = conn.Exec("update tb_accounts_info set lft = lft - 2 where lft > ?", elderLeader.AccRgt)
	if err != nil {
		conn.Rollback()
		return err
	}

	query, _ := conn.Prepare("UPDATE tb_accounts_info SET cipherText = ?  WHERE appAccountID = ?")

	defer query.Close()

	for i:=0; i<len(underEmployee);i++ {
		_, err = query.Exec(underEmployee[i].CipherText, underEmployee[i].AppID)
		if err != nil {
			conn.Rollback()
			return err
		}
	}


	conn.Commit()
	return nil
}

// 搜索获取下属员工账号列表
func SearchAccountInfoByAccount(keywords string, page, limit int64) (data EmployeeList, err error) {
	var employeeInfo = AccountInfo{}
	list := []AccountInfo{}
	start := (page - 1) * limit
	rsCount := db.Conns.QueryRow("select count(*) as count from tb_accounts_info where account like ?", "%"+keywords+"%")
	err = rsCount.Scan(&data.Count)
	if err != nil {
		return
	}

	if data.Count == 0 {
		data.CurrentPage = page
		data.TotalPage = 1
		data.List = list
		return
	}

	rs, err := db.Conns.Query(`select acc.account, acc.isUploaded, acc.cipherText, acc.appAccountID, rh.captain
    from tb_accounts_info as acc
      left join tb_registration_history as rh
        on rh.id = acc.regID
    where rh.consent = 2 and acc.account like ?
    limit ?, ?`, "%"+keywords+"%", start, limit)

	defer rs.Close()

	if err != nil {
		return
	}
	for rs.Next() {
		if rs.Err() != nil {
			err = rs.Err()
			return
		}
		rs.Scan(&employeeInfo.Account, &employeeInfo.Uploaded, &employeeInfo.CipherText, &employeeInfo.AppID, &employeeInfo.ManagerAppID)
		if employeeInfo.AppID != "" {
			ChildAccInfo, err := GetAccountInfoByAppID(employeeInfo.AppID)
			if err != sql.ErrNoRows {
				childCountInfo := db.Conns.QueryRow("SELECT count(*) as count FROM tb_accounts_info WHERE depth = ? AND isDepartured = 0 AND lft BETWEEN ? AND ?", ChildAccInfo.Depth+1, ChildAccInfo.AccLft, ChildAccInfo.AccRgt)
				err = childCountInfo.Scan(&employeeInfo.EmployeeNum)
			}
		}
		list = append(list, employeeInfo)
	}
	data.CurrentPage = page
	data.List = list
	data.TotalPage = int64(math.Ceil(float64(data.Count) / float64(limit)))
	return

}

// 根据直属上级appid获取下属账号信息
func GetEmployeeAccInfoByCaptainID(depth, c_lft, c_rgt, page, limit int64) (EmployeeList, error) {
	var employeeInfo = AccountInfo{}
	data := EmployeeList{}
	list := []AccountInfo{}
	start := (page - 1) * limit
	rsCount := db.Conns.QueryRow("SELECT count(*) as count FROM tb_accounts_info WHERE depth = ? AND isDepartured = 0 AND lft BETWEEN ? AND ?", depth, c_lft, c_rgt)
	err := rsCount.Scan(&data.Count)

	if err != nil {
		return EmployeeList{}, err
	}

	if data.Count == 0 {
		data.CurrentPage = page
		data.TotalPage = 1
		data.List = []AccountInfo{}
		return data, nil
	}

	rs, err := db.Conns.Query(`
	SELECT acc.account, acc.isUploaded, acc.cipherText ,acc.appAccountID, rh.captain, acc.lft, acc.rgt, acc.depth
  FROM tb_accounts_info as acc
    left join tb_registration_history as rh
      on rh.regID = acc.regID
  WHERE acc.depth = ? AND acc.isDepartured = 0 AND acc.lft BETWEEN ? AND ?
  ORDER BY acc.lft
  limit ?, ?`, depth, c_lft, c_rgt, start, limit)

	defer rs.Close()

	if err != nil {
		return EmployeeList{}, err
	}

	for rs.Next() {
		if rs.Err() != nil {
			err = rs.Err()
			return EmployeeList{}, err
		}

		err = rs.Scan(&employeeInfo.Account, &employeeInfo.Uploaded, &employeeInfo.CipherText, &employeeInfo.AppID, &employeeInfo.ManagerAppID, &employeeInfo.AccLft, &employeeInfo.AccRgt, &employeeInfo.Depth)

		if err != nil {
			return EmployeeList{}, err
		}


		err = db.Conns.QueryRow("SELECT count(*) as count FROM tb_accounts_info WHERE depth = ? AND isDepartured = 0 AND lft BETWEEN ? AND ?", employeeInfo.Depth+1, employeeInfo.AccLft, employeeInfo.AccRgt).Scan(&employeeInfo.EmployeeNum)

		if err != nil {
			return EmployeeList{}, err
		}
		//if employeeInfo.AppID != "" {
		//	//ChildAccInfo, err := GetAccountInfoByAppID(employeeInfo.AppID)
		//	if err != sql.ErrNoRows {
		//		childCountInfo := db.Conns.QueryRow("SELECT count(*) as count FROM tb_accounts_info WHERE depth = ? AND isDepartured = 0 AND lft BETWEEN ? AND ?", ChildAccInfo.Depth+1, ChildAccInfo.AccLft, ChildAccInfo.AccRgt)
		//		err = childCountInfo.Scan(&employeeInfo.EmployeeNum)
		//	}
		//}

		list = append(list, employeeInfo)
	}
	data.CurrentPage = page
	data.List = list
	data.TotalPage = int64(math.Ceil(float64(data.Count) / float64(limit)))
	return data, nil
}

// 记录上级审批记录结果
func UpdateCaptainApprovalInfo(regid, consent string) error {
	_, err := db.Conns.Exec("update tb_registration_history set consent = ?, isDeleted = ? where regID = ?", consent, 1, regid)
	if err != nil {
		return err
	}
	return nil
}

// 上级获取下属账号信息
func GetUnderlingInfoByManagerAccountID(depth, lft, rgt int64) ([]AccountInfo, error) {
	accInfo := AccountInfo{}
	list := []AccountInfo{}
	res, err := db.Conns.Query("select appAccountID, account, cipherText from tb_accounts_info where depth = ? and lft between ? and ?", depth, lft, rgt)

	defer res.Close()

	if err != nil {
		return nil, err
	}

	for res.Next() {
		if res.Err() != nil {
			return nil, res.Err()
		}

		err = res.Scan(&accInfo.AppID, &accInfo.Account, &accInfo.CipherText)
		if err != nil {
			return nil, err
		}
		list = append(list, accInfo)
	}
	return list, nil
}

// 更新摘要信息
func ChangeCipherInfo(employeeAccInfo []AccountInfo, cipherTexts []AccountInfo) []AccountInfo {
	data := []AccountInfo{}
	for _, r := range employeeAccInfo {
		for _, c := range cipherTexts {
			if r.AppID == c.AppID {
				eInfo := AccountInfo{}
				eInfo.AppID = r.AppID
				eInfo.CipherText = r.CipherText
				data = append(data, eInfo)
			}
		}
	}
	return data
}

// 替换员工
func ReplaceEmployee(oAppid, dAppid string) error {
	leaderAccInfo, err := GetAccountInfoByAppID(dAppid)
	if err != nil {
		return err
	}
	memberAccInfo, err := GetAccountInfoByAppID(oAppid)

	if err != nil {
		return err
	}

	conn, err := db.Conns.Begin()
	if err != nil {
		return err
	}

	if leaderAccInfo.AccLft > memberAccInfo.AccLft {
		_, err = conn.Exec("update tb_accounts_info set lft = lft - 2 where isDepartured = 0 and lft > ?", memberAccInfo.AccRgt)
		if err != nil {
			conn.Rollback()
			return err
		}

		_, err = conn.Exec("update tb_accounts_info set rgt = rgt - 2 where isDepartured = 0 and rgt > ?", memberAccInfo.AccRgt)
		if err != nil {
			conn.Rollback()
			return err
		}

		leaderAccInfo, err = GetAccountInfoByAppID(dAppid)
		if err != nil {
			return err
		}
		memberAccInfo, err = GetAccountInfoByAppID(oAppid)

		if err != nil {
			return err
		}
	}

	_, err = conn.Exec("update tb_accounts_info set rgt = rgt + 2 where rgt >= ? and isDepartured = 0", leaderAccInfo.AccRgt)
	if err != nil {
		conn.Rollback()
		return err
	}

	_, err = conn.Exec("update tb_accounts_info set lft = lft + 2 where lft > ? and isDepartured = 0", leaderAccInfo.AccRgt)
	if err != nil {
		conn.Rollback()
		return err
	}

	_, err = conn.Exec("update tb_accounts_info set lft = ?, rgt = ?, depth = ? where appAccountID = ?", leaderAccInfo.AccRgt, leaderAccInfo.AccRgt+1, memberAccInfo.AppID)
	if err != nil {
		conn.Rollback()
		return err
	}

	conn.Commit()
	return nil
}

func GetEmployeeEnPubKeyInfo(appid string) (AccountInfo, error) {
	data := AccountInfo{}
	row := db.Conns.QueryRow(`select acc.appAccountID, acc.pubKey, rh.captain, acc.enPubKey, acc.cipherText,
		UNIX_TIMESTAMP(acc.createdAt) as apply_at, acc.account
	from tb_accounts_info as acc
	left join tb_registration_history as rh
	on rh.regID = acc.regID
	where acc.appAccountID = ? and rh.consent = 2`, appid)

	err := row.Scan(&data.AppID, &data.Pubkey, &data.ManagerAppID, &data.EnPubkey, &data.CipherText, &data.ApplyAt, &data.Account)
	return data, err
}

func UpdateAccountsPubkeyUploadInfo(accInfos []AccountInfo) error {
	var query string
	if len(accInfos) > 0 {
		if len(accInfos) > 1 {
			query = fmt.Sprintf("update tb_accounts_info set isUploaded = 1 where appAccountID in (%v, ", accInfos[0].AppID)
			for i := 1; i < len(accInfos)-1; i++ {
				query = query + fmt.Sprintf(" %v, ", accInfos[i].AppID)
			}
			query = query + fmt.Sprintf(" %v)", accInfos[len(accInfos)-1].AppID)
		} else {
			query = fmt.Sprintf("update tb_accounts_info set isUploaded = 1 where appAccountID = %v", accInfos[0].AppID)
		}
		log.Info("query ", query)
		_, err := db.Conns.Exec(query)
		if err != nil {
			return err
		}
	}
	return nil
}


func GetEmployeeEnPubKeyInfoList(captainID string) ([]AccountInfo, error) {
	result := []AccountInfo{}
	res, err := db.Conns.Query(`
	SELECT t.applyer, acc.pubKey, rh.captain, acc.enPubKey, acc.cipherText as cipher_text,
		UNIX_TIMESTAMP(acc.createdAt) as apply_at, acc.account
	FROM(
		SELECT node.appAccountID as applyer
	FROM tb_accounts_info AS node,
		tb_accounts_info AS parent
	left join tb_registration_history as rh
	on rh.applyer = parent.appAccountID
	WHERE node.lft BETWEEN parent.lft AND parent.rgt
	AND rh.captain = ?
	ORDER BY node.lft) AS t
	LEFT JOIN tb_accounts_info AS acc
	ON acc.appAccountID = t.applyer
	LEFT JOIN tb_registration_history AS rh
	ON rh.regID = acc.regID
	WHERE acc.isUploaded = 0 AND acc.isDepartured = 0 and rh.consent = 2`, captainID)

	defer res.Close()

	if err != nil {
		return nil, err
	}

	for res.Next() {
		emAccInfo := AccountInfo{}
		if res.Err() != nil {
			return nil, res.Err()
		}
		err = res.Scan(&emAccInfo.AppID, &emAccInfo.Pubkey, &emAccInfo.ManagerAppID, &emAccInfo.EnPubkey, &emAccInfo.CipherText, &emAccInfo.ApplyAt, &emAccInfo.Account)

		if err != nil {
			return nil, err
		}

		result = append(result, emAccInfo)
	}
	return result, nil
}

// 获取账号对应的根节点账号
func GetRootAccountByUnderlingAcc(lft, rgt int64) (AccountInfo, error) {
	accInfo := AccountInfo{}
	row := db.Conns.QueryRow("select id, lft, rgt from tb_accounts_info where lft <= ? and rgt >= ? and depth = 0", lft, rgt)
	err := row.Scan(&accInfo.ID, &accInfo.AccLft, &accInfo.AccRgt)
	return accInfo, err
}

// 修改账号所属部门信息
func ChangeAccBranch(appid, bid string ) error {
	_, err := db.Conns.Exec("UPDATE tb_accounts_info SET branchID = ? WHERE appAccountID = ?", bid, appid)
	return err
}

func GetAccountsListByBid(bid string, lft, rgt int64)([]AccBriefInfo, error) {
	list := []AccBriefInfo{}
	employees := AccBriefInfo{}

	res, err := db.Conns.Query("SELECT id, appAccountID, account, depth, branchID, lft, rgt FROM tb_accounts_info WHERE branchID = ? AND isDepartured = 0 AND lft >= ? AND rgt <= ?", bid, lft, rgt)

	if err != nil {
		return []AccBriefInfo{}, err
	}

	for res.Next() {
		if res.Err() != nil {
			return []AccBriefInfo{}, res.Err()
		}
		var accLft, accRgt int
		err = res.Scan(&employees.ID, &employees.AppID, &employees.Account, &employees.Depth, &employees.BranchID, &accLft, &accRgt)

		if err != nil {
			return []AccBriefInfo{}, err
		}

		_ = db.Conns.QueryRow("SELECT count(b.id) FROM tb_accounts_info as a left join tb_accounts_info as b on b.depth = a.depth + 1 where a.id = ? and b.isDepartured = 0 and b.lft between a.lft and a.rgt", employees.ID).Scan(&employees.EmployeeNum)

		list = append(list, employees)
	}

	return list, nil
}

// 用户输入密码错误，记录尝试次数
func AttemptFrozen(appid string) (bool, int, error) {
	var attempts int
	// 查询用户尝试次数
	res := db.Conns.QueryRow("SELECT attempts FROM tb_accounts_info WHERE appAccountID = ?", appid)
	err := res.Scan(&attempts)
	if err != nil {
		return false, 0, err
	}
	if attempts == MAX_ATTEMPTS - 1 {
		// 冻结账户，设置解锁时间
		var frozenTo = time.Now().Add(FROZEN_HOUR*time.Hour).Unix()
		db.Conns.Exec("UPDATE tb_accounts_info set attempts = 0, frozen = 1, frozenTo = FROM_UNIXTIME(?) WHERE appAccountID = ?", frozenTo, appid)
		return true, 5, nil

	} else {
		// 尝试次数+1
		db.Conns.Exec("UPDATE tb_accounts_info SET attempts = attempts + 1 WHERE appAccountID = ?", appid)
		return false, attempts+1, nil
	}
}

// 重置用户密码尝试次数
func ResetAttempts(appid string) error {
	_, err := db.Conns.Exec("UPDATE tb_accounts_info SET attempts = 0 WHERE appAccountID = ?", appid)
	return err
}

// 修改密码
func ModifyPwd(appid, newpwd string) error {
	pwd := utils.GenHashStr(newpwd)

	_, err := db.Conns.Exec("UPDATE tb_accounts_info SET passwords = ? WHERE appAccountID = ?", pwd, appid)
	return err
}