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
	"fmt"
	"github.com/boxproject/appServerGo/db"
	"database/sql"
	"encoding/json"
	"github.com/boxproject/appServerGo/utils"
	"github.com/go-errors/errors"
	"math"
	"github.com/boxproject/appServerGo/config"
	log "github.com/alecthomas/log4go"
)

var cfg = config.GetConfig()

func GetBusinessFlowInfoByFlowID(flowid string, paramType int64)(FlowInfo, error) {
	var flow_content_str, where_str string
	flowInfo := FlowInfo{}
	flowContent := FlowInfo{}
	if paramType == 0 {
		where_str = " where id = ? "
	} else {
		where_str = " where flowID = ? "
	}
	row := db.Conns.QueryRow("select id, flowID, flowHash, flowName, progress, founderID, UNIX_TIMESTAMP(createdAt), content, singleLimit, ifnull(UNIX_TIMESTAMP(updatedAt), 0) from tb_business_flow " + where_str, flowid)
	err := row.Scan(&flowInfo.ID, &flowInfo.FlowID, &flowInfo.Hash, &flowInfo.Name, &flowInfo.Progress, &flowInfo.FounderID, &flowInfo.CreatedAt, &flow_content_str, &flowInfo.SingleLimit, &flowInfo.UpdatedAt)
	//log.Debug("content_str", flow_content_str)
	json.Unmarshal([]byte(flow_content_str), &flowContent)
	if flowInfo.Progress == FLOW_APPROVALED {
		flowInfo.ApprovalAt = flowInfo.UpdatedAt
	}
	flowInfo.ApprovalInfo = flowContent.ApprovalInfo
	flowInfo.Period = flowContent.Period
	flowInfo.ApprovalInfo = flowContent.ApprovalInfo
	flowInfo.FlowLimit = flowContent.FlowLimit
	return flowInfo, err
}

//func GetTransferFlowInfoByFlowID(flowid string, transID int64)(FlowInfo, error) {
//	var flow_content_str string
//	flowInfo := FlowInfo{}
//	flowContent := FlowInfo{}
//
//	row := db.Conns.QueryRow("select id, flowID, flowHash, flowName, progress, UNIX_TIMESTAMP(createdAt), content, singleLimit, UNIX_TIMESTAMP(updatedAt) from tb_business_flow where id = ?", flowid)
//
//	err := row.Scan(&flowInfo.ID, &flowInfo.FlowID, &flowInfo.Hash, &flowInfo.Name, &flowInfo.Progress, &flowInfo.CreatedAt, &flow_content_str, &flowInfo.SingleLimit, &flowInfo.UpdatedAt)
//
//	json.Unmarshal([]byte(flow_content_str), &flowContent)
//
//	if flowInfo.Progress == 2 {
//		flowInfo.ApprovalAt = flowInfo.UpdatedAt
//	}
//
//	flowInfo.ApprovalInfo = flowContent.ApprovalInfo
//
//	for i:=0; i<len(flowInfo.ApprovalInfo); i++ {
//		approvers := flowInfo.ApprovalInfo[i].Approvers
//		for j:=0; j<len(approvers);j++ {
//			err = db.Conns.QueryRow("SELECT sign FROM tb_review_transfer WHERE managerAccID = ? AND transID = ?", approvers[j].AppID, transID).Scan(&approvers[j].Sign)
//			if err == sql.ErrNoRows {
//				err = nil
//				break
//			}
//		}
//	}
//	return flowInfo, err
//}


// 更新审批流上链状态
func UpdateFlowStatus(id string, status int) error {
	res, err := db.Conns.Prepare("update tb_business_flow set progress = ? where id = ?")
	defer res.Close()
	if err != nil {
		return err
	}
	_, err = res.Exec(status, id)
	if err != nil {
		return err
	}
	return nil
}

// 获取指定订单的审批详情
func GetApprovalInfoByTxID(flowInfo FlowInfo, txID int64)(approvalInfo []TxApprovalInfo, err error) {
	approval_info :=  flowInfo.ApprovalInfo
	for i:=0; i<len(approval_info);i++ {
		data := TxApprovalInfo{}
		total_approvers := 0
		total_rejects := 0
		final_progress := 0
		data.Require = approval_info[i].Require
		data.Total = int64(len(approval_info[i].Approvers))
		for j:=0; j<len(approval_info[i].Approvers);j++ {
			approval_info[i].Approvers[j].Progress = 0
			theApprovalInfo, err := GetTxApprovalInfoByAccID(approval_info[i].Approvers[j].AppID, txID)
			if err != sql.ErrNoRows {
				approval_info[i].Approvers[j].Progress = theApprovalInfo.Progress
				approval_info[i].Approvers[j].Sign = theApprovalInfo.Sign
			}

			if theApprovalInfo.Progress == TX_APPROVALED {
				total_approvers++
			} else if theApprovalInfo.Progress == TX_REJECTED {
				total_rejects++
			}
		}
		if total_approvers >= approval_info[i].Require {
			final_progress = TX_APPROVALED
		} else if total_rejects > len(approval_info[i].Approvers) - approval_info[i].Require {
			final_progress = TX_REJECTED
		} else if total_rejects == 0 && total_approvers == 0 {
			final_progress = TX_WAITING
		} else {
			final_progress = TX_DOING
		}
		data.CurrentProgress = final_progress
		data.Approvers = approval_info[i].Approvers
		approvalInfo = append(approvalInfo, data)
	}
	return
}

// 获取用户转账申请的审批信息
func GetTxApprovalInfoByAccID(appid string, tx_id int64)(txSignInfo Approver, err error) {
	row := db.Conns.QueryRow("select rt.sign, rt.comments from tb_accounts_info as acc left join tb_review_transfer as rt on rt.managerAccID = acc.id where acc.appAccountID = ? and rt.transID = ?", appid, tx_id)
	err = row.Scan(&txSignInfo.Sign, &txSignInfo.Progress)
	return
}

// 业务流哈希是否存在
func FlowHashExist(hash string) (bool, error) {
	var id string
	res := db.Conns.QueryRow("select id from tb_business_flow where flowHash = ?", hash)
	err := res.Scan(&id)
	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return true, err
	}
	return true, nil
}

// 生成审批流
func GenBusinessFlow(flowID, flowHash, flowName, flow, sign, singleLimit string, founderID int64, flowLimit []flowLimit, period string) (int,error) {
	conn, err := db.Conns.Begin()

	if err != nil {
		return 1000, err
	}

	// 生成审批流
	res, err := db.Conns.Exec("insert into tb_business_flow set flowID = ?, flowHash = ?, flowName = ?, founderID = ?, content = ?, founderSign = ?, singleLimit = ?", flowID, flowHash, flowName, founderID, flow, sign, singleLimit)

	if err != nil {
		return 1000, err
		conn.Rollback()
	}
	flow_id, _ := res.LastInsertId()

	for i:=0; i<len(flowLimit); i++ {
		// 获取对应币种信息
		var currencyID int
		err = conn.QueryRow("SELECT id FROM tb_currency WHERE currency = ? AND available = 1", flowLimit[i].CurrencyName).Scan(&currencyID)

		if err != nil {
			if err == sql.ErrNoRows {
				return 2002, nil
			}
			return 1000, err
		}

		// 初始化对应币种额度
		_, err = conn.Exec("INSERT INTO tb_flow_limit (flowID, currencyID, amountLeft, amount, period) VALUES (?, ?, ?, ?, ?)", flow_id, currencyID, flowLimit[i].Limit, flowLimit[i].Limit, period)

		if err != nil {
			return 1000, err
			conn.Rollback()
		}
	}
	conn.Commit()
	return 0, nil
}

// 根据根节点获取审批流内容
func GetFlowInfoByID(flowID string, mAccid int64)(FlowInfo, error) {
	flow := FlowInfo{}
	var flow_content_str string
	row := db.Conns.QueryRow("select f.id, f.flowName, f.content, f.progress, acc.appAccountID from tb_business_flow f left join tb_accounts_info acc on acc.id = f.founderID where f.flowID = ? and f.founderID = ?", flowID, mAccid)
	err := row.Scan(&flow.ID, &flow.Name, &flow_content_str, &flow.Progress, &flow.CreatedBy)

	err = db.Conns.QueryRow("SELECT count(*) FROM tb_transfer WHERE flowID = ? AND progress < ?", flow.ID, TX_REJECTED).Scan(&flow.PendingTxNum)

	json.Unmarshal([]byte(flow_content_str), &flow)
	if flow.Progress == FLOW_APPROVALED {
		flow.ApprovalAt = flow.UpdatedAt
	}
	return flow, err
}

func DisableFlows() error {
	_, err := db.Conns.Exec("update tb_business_flow set progress = 2 where progress <> 0")

	if err != nil {
		return err
	}
	return nil
}


// 从代理服务器获取已上链的审批流列表
type approvaledFlows struct {
	RspNo 	string
	ApprovalInfos []RPCFlowInfo
}
func UpdateFlowStatusByRPC()(bool, error) {
	cfg := config.GetConfig()
	urls := cfg.Agent.Server + cfg.Agent.ApprovaledInfos + "?type=3"
	res := approvaledFlows{}

	data, err := utils.HttpRequest("GET", urls, "")

	if err != nil {
		return false, err
	}

	err = json.Unmarshal(data, &res)
	if err != nil {
		return false, err
	}

	if res.RspNo != "0" {
		return false, errors.New("从代理服务器获取已上链的审批流列表错误: " + res.RspNo)
	}

	// 更新本地数据库中对应flow的状态
	if len(res.ApprovalInfos)>0 {
		for i:=0;i<len(res.ApprovalInfos);i++ {
			var flow_statu int
			if res.ApprovalInfos[i].Status == "7" {
				flow_statu = FLOW_APPROVALED

			} else if res.ApprovalInfos[i].Status == "2" || res.ApprovalInfos[i].Status == "5" {
				flow_statu = FLOW_REJECTED
			} else if res.ApprovalInfos[i].Status == "9" {
				flow_statu = FLOW_INVALID
			}

			_, err := db.Conns.Exec("UPDATE tb_business_flow SET progress = ? WHERE flowHash = ?", flow_statu, res.ApprovalInfos[i].Hash)

			if err != nil {
				log.Error("获取审批流列表，更新审批流审批状态", err)
				return false, err
			}

		}
	}

	return true, nil
}

// 更新审批流的额度信息
func UpdateFlowAmount(timestamp int64) error {
	_, err := db.Conns.Exec("UPDATE tb_flow_limit SET amountLeft = amount, frozenTo = ?, frozen = 0 WHERE amountLeft != amount and frozenTo is NULL OR UNIX_TIMESTAMP(frozenTo) < ? ", sql.NullString{}, timestamp)
	return err
}

// 搜索获取审批流列表

func SearchFlowByName(keywords string, founderID, page, limit, statu int64, currencyName, currencyID string)(FlowListInfo, error) {
	result := FlowListInfo{}
	start := (page-1)*limit
	query := "select id, flowID, flowName, content, flowHash, progress, singleLimit, ifnull(UNIX_TIMESTAMP(fl.frozenTo), 0) from tb_business_flow where founderID = ? "
	where_str := "and flowName like ? order by createdAt desc limit ?, ?"
	progress_query := ""
	if statu == 1 {
		progress_query = "and progress = 3"
	}

	row := db.Conns.QueryRow("select count(*) as count from tb_business_flow where founderID = ? and flowName like ? " + progress_query, founderID, "%"+keywords+"%")
	if currencyName != "" {
		row = db.Conns.QueryRow("SELECT count(f.id) FROM tb_business_flow f left join tb_flow_limit fl on fl.flowID = f.id WHERE f.founderID = ? AND f.flowName LIKE ? AND fl.currencyID = ? AND f.progress = ?", founderID, "%"+keywords+"%", currencyID, FLOW_APPROVALED)
	}

	err := row.Scan(&result.Count)

	if err != nil {
		return FlowListInfo{}, err
	}

	result.TotalPage = int64(math.Ceil(float64(result.Count)/float64(limit)))
	result.CurrentPage = page

	rows, err := db.Conns.Query(query+progress_query+where_str, founderID, "%"+keywords+"%", start, limit)

	if currencyName != "" {
		rows, err = db.Conns.Query("select f.id, f.flowID, f.flowName, f.content, f.flowHash, f.progress, f.singleLimit, fl.amountLeft from tb_business_flow f left join tb_flow_limit fl on fl.flowID = f.id where f.founderID = ? and f.progress = ? and fl.currencyID = ? AND fl.frozen = 0 and f.flowName like ? order by f.createdAt desc limit ?, ?", founderID, FLOW_APPROVALED, currencyID, "%"+keywords+"%", start, limit)
	}
	defer rows.Close()

	for rows.Next() {
		if rows.Err() != nil {
			return FlowListInfo{}, rows.Err()
		}
		data := FlowInfo{}
		var flowContent, currencyAmountLeft string
		var frozenTo int64

		if currencyName != "" {
			rows.Scan(&data.ID, &data.FlowID, &data.Name, &flowContent, &data.Hash, &data.Progress, &data.SingleLimit, &currencyAmountLeft)
			data.FlowLimit = append(data.FlowLimit, flowLimit{CurrencyName:currencyName, Limit:currencyAmountLeft})
		} else {
			rows.Scan(&data.ID, &data.FlowID, &data.Name, &flowContent, &data.Hash, &data.Progress, &data.SingleLimit, &frozenTo)
		}

		if flowContent != "" {
			json.Unmarshal([]byte(flowContent), &data.ApprovalInfo)
		}

		result.List = append(result.List, data)
	}

	return result, nil
}

func GetFlowList(founderID, page, limit, statu int64, currencyName, currencyID string)(FlowListInfo, error) {
	start := (page-1)*limit
	where_str := ""
	result := FlowListInfo{}
	list := []FlowInfo{}
	if statu == 1 {
		where_str = " and progress = 3 "
	}

	rowCount := db.Conns.QueryRow("select count(*) as count from tb_business_flow where founderID = ? "+where_str, founderID)
	if currencyName != "" {
		rowCount = db.Conns.QueryRow("SELECT count(f.id) FROM tb_business_flow f LEFT JOIN tb_flow_limit fl on fl.flowID = f.id WHERE f.founderID = ? and f.progress = ? and fl.currencyID = ?", founderID, FLOW_APPROVALED, currencyID)
	}
	err := rowCount.Scan(&result.Count)

	if err != nil {
		return FlowListInfo{}, err
	}

	result.TotalPage = int64(math.Ceil(float64(result.Count)/float64(limit)))
	result.CurrentPage = page

	query := "select id, flowID, flowName, content, flowHash, progress, singleLimit, founderID from tb_business_flow where founderID = ? "

	res, err := db.Conns.Query(query+where_str+ " order by createdAt desc limit ?, ? ", founderID, start, limit)

	if currencyName != "" {
		res, err = db.Conns.Query(`
SELECT f.id, f.flowID, f.flowName, f.content, f.flowHash, f.progress, f.singleLimit, fl.amountLeft, f.founderID
FROM tb_business_flow f
	LEFT JOIN tb_flow_limit fl
		ON fl.flowID = f.id
WHERE f.founderID = ? AND fl.currencyID = ? AND f.progress = ? ORDER BY f.createdAt DESC limit ?, ?`, founderID, currencyID, FLOW_APPROVALED, start, limit)
	}
	defer res.Close()

	if err != nil {
		log.Error("FlowList", err)
		return FlowListInfo{}, err
	}


	for res.Next() {
		if res.Err() != nil {
			return FlowListInfo{}, res.Err()
		}

		data := FlowInfo{}

		var flowContent, currencyAmountLeft string

		if currencyID != "" {
			res.Scan(&data.ID, &data.FlowID, &data.Name, &flowContent, &data.Hash, &data.Progress, &data.SingleLimit, &currencyAmountLeft, &data.FounderID)
			flow_limit_info := flowLimit{CurrencyName:currencyName, Limit:currencyAmountLeft}
			data.FlowLimit = append(data.FlowLimit, flow_limit_info)

		} else {
			res.Scan(&data.ID, &data.FlowID, &data.Name, &flowContent, &data.Hash, &data.Progress, &data.SingleLimit, &data.FounderID)
		}
		if flowContent != "" {
			json.Unmarshal([]byte(flowContent), &data.ApprovalInfo)
		}
		list = append(list, data)
	}
	result.List = list
	return result, nil
}

// 获取审批者在审批流中的位置
func GetManagerLocation(flowContent []Approvalinfo, appID string) Location {
	location := Location{}
	for i :=0; i<len(flowContent);i++ {
		approvers := flowContent[i].Approvers
		for j:=0; j<len(approvers); j++ {
			if approvers[j].AppID == appID {
				location.Level = i
				location.Number = j
				location.Require = flowContent[i].Require
				break
			}
		}
	}
	return location
}

func InitManagerComments(flowContent []Approvalinfo, transID int64, location Location) error {
	approversInfo := flowContent[location.Level]
	var pass, rej int
	for i:=0;i<len(approversInfo.Approvers);i++  {
		comments, err := GetTxInfoByApprover(approversInfo.Approvers[i].AppID, transID)

		if err != nil {
			return err
		}

		if comments == 2 {
			rej++
		}

		if comments == 3 {
			pass++
		}
	}

	if pass >= approversInfo.Require && len(flowContent) >location.Level+1 {
		// 某一层approvers审批通过
		newApprovers := flowContent[location.Level+1].Approvers

		mAccInfo, err := GetAccountInfoByAppID(newApprovers[0].AppID)
		if err != nil {
			return err
		}
		query_s := fmt.Sprintf("select transID from tb_review_transfer where transID = %v and managerAccID in ( %v ", transID, mAccInfo.ID)
		query := fmt.Sprintf("insert into tb_review_transfer (transID, managerAccID) values (%v, %v) ", transID, mAccInfo.ID)

		if len(newApprovers) >1 {
			for i:=1; i<len(newApprovers); i++ {
				query_s = query_s + fmt.Sprintf(" ,%v ", newApprovers[i].ID)
				newManagerAcc, _ := GetAccountInfoByAppID(newApprovers[i].AppID)
				query = query + fmt.Sprintf(", (%v, %v) ", transID, newManagerAcc.ID)
			}
		}
		query_s = query_s + ")"

		rowsLength := 0
		res, err := db.Conns.Query(query_s)

		if err != nil {
			log.Error("通知上级待审批", err)
			res.Close()
			return err
		}

		for res.Next() {

			rowsLength++

			var transid string

			err = res.Scan(&transid)
			if err != nil {
				return err
				res.Close()
			}
		}

		res.Close()

		conn, err := db.Conns.Begin()

		if err != nil {
			return err
		}

		if rowsLength == 0  {

			_, err = conn.Exec("update tb_review_transfer set comments = -1 where transID = ? and sign IS NULL", transID)

			if err != nil {
				conn.Rollback()
				return err
			}

			_, err = conn.Exec(query)

			if err != nil {
				conn.Rollback()
				return err
			}
		}
		conn.Commit()
	}
	return nil
}

// 获取各级人员审批信息
type TxSignInfo struct {
	AppID string `json:"appid"`
	Signature string `json:"sign"`
}
func GetTxApproversSign(approvalInfo []Approvalinfo, transID int64) []TxSignInfo {
	result := []TxSignInfo{}
	data := TxSignInfo{}
	for i:=0; i<len(approvalInfo); i++ {
		approvers := approvalInfo[i].Approvers
		for j:=0; j<len(approvers); j++ {
			var sign string
			db.Conns.QueryRow("SELECT rt.sign FROM tb_review_transfer rt left join tb_accounts_info acc on acc.id = rt.managerAccID WHERE rt.transID = ? AND acc.appAccountID = ? ", transID, approvers[j].AppID).Scan(&sign)


			if sign != ""  {
				data.AppID = approvers[j].AppID
				data.Signature = sign
				result = append(result, data)
			}
		}
	}
	return result
}

// 获取部门信息
func GetBranchInfoByID(bid string ) (Branch, error) {
	branchInfo := Branch{}
	res := db.Conns.QueryRow("SELECT id, `name`, bIndex, UNIX_TIMESTAMP(createdAt), ifnull(UNIX_TIMESTAMP(updatedAt), 0) from tb_department WHERE id = ?", bid)
	err := res.Scan(&branchInfo.ID, &branchInfo.Name, &branchInfo.Index, &branchInfo.CreatedAt, &branchInfo.UpdatedAt)
	return branchInfo, err
}

// 添加部门
func AddBranch(name string, createdBy int64) error {
	cfg := config.GetConfig()
	var index, other_index int
	index_sql := `SELECT auto_increment as t FROM information_schema.`+"`TABLES`" + " WHERE TABLE_SCHEMA= " + "'" + cfg.Database.DbName + "'" + ` AND TABLE_NAME='tb_department'`
	err := db.Conns.QueryRow(index_sql).Scan(&index)
	db.Conns.QueryRow("SELECT bIndex FROM tb_department WHERE id = 1 ").Scan(&other_index)
	conn, err := db.Conns.Begin()

	if err != nil {
		return err
	}

	_, err =conn.Exec("insert into tb_department (bIndex, `name`, createdBy) values (?, ?, ?)", index, name, createdBy)

	if err != nil {
		conn.Rollback()
		return err
	}

	_, err = conn.Exec(`
	UPDATE tb_department SET bIndex = CASE
	WHEN id = 1 THEN ?
	ELSE bIndex - 1
	END
	WHERE bIndex >= ?`, index, other_index)

	if err != nil {
		conn.Rollback()
		return err
	}
	conn.Commit()

	return nil
}

// 删除部门
func DelBranch(bid string) error {
	cfg := config.GetConfig()
	conn, err := db.Conns.Begin()

	if err != nil {
		return err
	}

	// 删除部门
	_, err = conn.Exec("delete from tb_department WHERE id = ?", bid)

	if err != nil {
		conn.Rollback()
		return err
	}

	_, err = conn.Exec("UPDATE tb_department SET bIndex = bIndex -1 WHERE id > ?", bid)

	if err != nil {
		conn.Rollback()
		return err
	}

	// 重设auto_increment值
	index_sql := `SELECT auto_increment as t FROM information_schema.`+"`TABLES`" + " WHERE TABLE_SCHEMA= " + "'" + cfg.Database.DbName + "'" + ` AND TABLE_NAME='tb_department'`

	next_auto_increment := 0
	err = conn.QueryRow(index_sql).Scan(&next_auto_increment)

	if err != nil {
		conn.Rollback()
		return err
	}

	sql_alter := fmt.Sprintf("alter table " + cfg.Database.DbName +  ".tb_department AUTO_INCREMENT = %v", next_auto_increment-1)
	_, err = conn.Exec(sql_alter)

	if err != nil {
		conn.Rollback()
		return err
	}

	// 将对应员工部门ID置为1
	_, err = conn.Exec("update tb_accounts_info set branchID = 1 where branchID = ?", bid)

	if err != nil {
		conn.Rollback()
		return err
	}

	conn.Commit()

	return nil
}

// 修改部门名称
func ChangeBranchName(bid, name string) error {
	_, err := db.Conns.Exec("UPDATE tb_department SET `name` = ? WHERE id = ?", name, bid)
	return err
}

// 更改部门index
func ChangeBranchIndex(index int64, branchInfo Branch) error {
	//var start, end int64

	conn, err := db.Conns.Begin()

	if err != nil {
		return err
	}

	_, err = conn.Exec("UPDATE tb_department SET bIndex = ? WHERE id = ?", index, branchInfo.ID)

	if err != nil {
		conn.Rollback()
		return err
	}

	if index < branchInfo.Index {
		_, err = conn.Exec("UPDATE tb_department SET bIndex = bIndex + 1 WHERE id <> ? AND bIndex BETWEEN ? AND ?", branchInfo.ID, index, branchInfo.Index)

		if err != nil {
			conn.Rollback()
			return err
		}
	} else {
		_, err = conn.Exec("UPDATE tb_department SET bIndex = bIndex - 1 WHERE id <> ? AND bIndex BETWEEN ? AND ?", branchInfo.ID, branchInfo.Index, index)

		if err != nil {
			conn.Rollback()
			return err
		}
	}



	conn.Commit()

	return nil
}

// 获取部门列表
func GetBranchList(accID, lft, rgt int64) ([]Branch, error) {
	list := []Branch{}

	rows, err := db.Conns.Query("select dep.id, dep.`name`, dep.bIndex, UNIX_TIMESTAMP(dep.createdAt), ifnull(UNIX_TIMESTAMP(dep.updatedAt), 0), count(acc.id) from tb_department dep left join tb_accounts_info as acc on acc.lft between ? and ? and acc.branchID = dep.id where dep.createdBy in (0, ?) GROUP BY dep.id ORDER BY dep.bIndex", lft, rgt, accID)

	defer rows.Close()

	if err != nil {
		return list, err
	}

	for rows.Next() {
		branchInfo := Branch{}
		rows.Scan(&branchInfo.ID, &branchInfo.Name, &branchInfo.Index, &branchInfo.CreatedAt, &branchInfo.UpdatedAt, &branchInfo.Employees)

		list = append(list, branchInfo)
	}

	return list, nil
}

// 作废审批流
func DisuseFlow(flowID string) error {
	_, err := db.Conns.Exec("UPDATE tb_business_flow SET progress = ? WHERE flowID = ?", FLOW_INVALID, flowID)
	return err
}
