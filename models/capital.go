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
	"github.com/boxproject/appServerGo/utils"
	"github.com/boxproject/appServerGo/db"
	"fmt"
	"database/sql"
	"strconv"
	"math"
	"encoding/json"
	"github.com/boxproject/appServerGo/config"
	log "github.com/alecthomas/log4go"
	"time"
)

// 提交转账申请
func ApplyTransfer(orderNumber, applyInfo, sign string, approversInfo FlowInfo, transferContent TransferContent, applyerAccInfo AccountInfo, currencyInfo CurrencyInfo) error {
	transHash := utils.GenHashStr(applyInfo)
	captainsInfo := approversInfo.ApprovalInfo[0].Approvers

	conn, err := db.Conns.Begin()
	if err != nil {
		return err
	}

	rs, err := conn.Exec("insert into tb_transfer set orderNum = ?, txInfo = ?, transBoxHash = ?, applyerID = ?, currencyID = ?, amount = ?, flowID = ?, applyContent = ?, applyerSign = ?", orderNumber, transferContent.TxInfo, transHash, applyerAccInfo.ID, currencyInfo.CurrencyID, transferContent.Amount, approversInfo.ID, applyInfo, sign)
	if err != nil {
		conn.Rollback()
		return err
	}

	tx_id, err := rs.LastInsertId()
	if err != nil {
		conn.Rollback()
		return err
	}

	if len(captainsInfo) > 0 {
		for i := 0; i < len(captainsInfo); i++ {
			accid := 0
			_ = conn.QueryRow("SELECT id FROM tb_accounts_info WHERE appAccountID = ?", captainsInfo[i].AppID).Scan(&accid)
			_, err = conn.Exec("insert into tb_review_transfer (transID, managerAccID) values (?, ?)", tx_id, accid)
			if err != nil {
				conn.Rollback()
				return err
			}
		}
	}

	conn.Commit()
	return nil
}

// 获取币种信息
func GetCurrencyInfoByName(currency string) (currencyInfo CurrencyInfo, err error) {
	currencyInfo = CurrencyInfo{}
	row := db.Conns.QueryRow("select id, factor, currency, balance from tb_currency where currency = ? and available = 1", currency)
	err = row.Scan(&currencyInfo.CurrencyID, &currencyInfo.Factor, &currencyInfo.Currency, &currencyInfo.Balance)
	//if err != nil {
	//	currencyInfo.Currency = strings.Replace(currencyInfo.Currency, " ", "", -1)
	//}
	return
}

// 根据申请者获取转账记录
func GetTransferRecordsListByApplyerID(accid, progress int64, page, limit string) (result TransferRecordInfo, err error) {
	end, _ := strconv.ParseInt(limit, 10, 64)
	p, _ := strconv.ParseInt(page, 10, 64)
	start := (p - 1) * end
	var data = TransferRecordInfo{}
	var data_info = TxRecordInfo{}
	data.List = []TxRecordInfo{}
	var rowTotal *sql.Row
	if progress != TX_ALL {
		rowTotal = db.Conns.QueryRow("select count(*) from tb_transfer where applyerID = ? and progress = ?", accid, progress)
	} else {
		rowTotal = db.Conns.QueryRow("select count(*) from tb_transfer where applyerID = ? ", accid)
	}
	err = rowTotal.Scan(&data.Count)
	if err == sql.ErrNoRows {
		err = nil
		return
	}

	data.CurrentPage = p
	data.TotalPage = int64(math.Ceil(float64(data.Count) / float64(end)))
	rows, err := db.Conns.Query("select t.orderNum, t.txInfo, t.progress, t.amount, c.currency, UNIX_TIMESTAMP(t.createdAt) as apply_at, t.arrived from tb_transfer as t left join tb_currency as c on c.id = t.currencyID where applyerID = ? order by apply_at desc limit ?, ?", accid, start, end)

	defer rows.Close()

	if err != nil {
		return
	}
	for rows.Next() {
		if rows.Err() != nil {
			err = rows.Err()
			return
		}
		err = rows.Scan(&data_info.OrderNum, &data_info.Tag, &data_info.Progress, &data_info.Amount, &data_info.Currency, &data_info.ApplyAt, &data_info.Arrived)

		if err != nil {
			return
		}
		data.List = append(data.List, data_info)
	}
	result = data
	return
}

// 根据审批者获取转账记录
func GetTransferRecordListByCaptainID(accid, progress int64, page, limit string) (result TransferRecordInfo, err error) {
	end, _ := strconv.ParseInt(limit, 10, 64)
	p, _ := strconv.ParseInt(page, 10, 64)
	start := (p - 1) * end
	var data = TransferRecordInfo{}
	var data_info = TxRecordInfo{}
	var comments_str, order_str string
	var row *sql.Row
	var rows *sql.Rows
	data.List = []TxRecordInfo{}

	order_str = " order by apply_at desc limit ?, ? "
	query_total := "select count(*) as total from tb_transfer as t left join tb_review_transfer as rt on rt.transID = t.id where rt.managerAccID = ? "

	if progress == TX_WAITING {
		comments_str = " and rt.comments = ? and t.progress < 2 "
	} else if progress == TX_ALL {
		comments_str = ""
	} else {
		comments_str = " and t.progress = ? and rt.comments <> 0 "
	}

	query_total = query_total + comments_str

	query := `
	select t.orderNum, t.txInfo, UNIX_TIMESTAMP(t.createdAt) as apply_at, t.amount, c.currency,
      (case t.progress when 0 then 1 else t.progress end) as progress, t.arrived
      from tb_transfer as t
        left join tb_review_transfer as rt
          on rt.transID = t.id
        left join tb_currency as c
          on c.id = t.currencyID
      where rt.managerAccID = ? `
	query = query + comments_str + order_str

	if comments_str != "" {
		row = db.Conns.QueryRow(query_total, accid, progress)
		rows, err = db.Conns.Query(query, accid, progress, start, end)

	} else {
		row = db.Conns.QueryRow(query_total, accid)
		rows, err = db.Conns.Query(query, accid, start, end)
	}

	defer rows.Close()
	if err != nil {
		return
	}

	err = row.Scan(&data.Count)
	if err == sql.ErrNoRows {
		err = nil
		return
	}

	data.CurrentPage = p
	data.TotalPage = int64(math.Ceil(float64(data.Count) / float64(end)))

	for rows.Next() {
		if rows.Err() != nil {
			err = rows.Err()
			return
		}
		err = rows.Scan(&data_info.OrderNum, &data_info.Tag, &data_info.ApplyAt, &data_info.Amount, &data_info.Currency, &data_info.Progress, &data_info.Arrived)

		if err != nil {
			return
		}
		data.List = append(data.List, data_info)
	}
	result = data
	return
}

func GetTransferInfo(orderNum string , orderType int) (flow_id string, txInfo TransferInfo, err error) {
	//txInfo = TransferInfo{}
	var updated_at int64
	var where_str string
	if orderType == TXTYPE_ID {
		// 根据tb_transfer.id获取
		where_str = fmt.Sprintf(" where t.id = ? ")
	} else if orderType == TXTYPE_ORDERNUM {
		where_str = fmt.Sprintf(" where t.orderNum = ? ")
	}
	query := fmt.Sprintf("select t.id, t.orderNum, t.transBoxHash, a.account, t.progress, t.applyContent, a.appAccountID, t.flowID, t.currencyID, t.amount, UNIX_TIMESTAMP(t.createdAt) as apply_at, t.arrived, ifnull(UNIX_TIMESTAMP(t.updatedAt), 0) as updated_at, f.flowName from tb_transfer as t left join tb_accounts_info as a on a.id = t.applyerID left join tb_business_flow f on f.id = t.flowID " + where_str)
	row := db.Conns.QueryRow(query, orderNum)
	err = row.Scan(&txInfo.TxID, &txInfo.OrderNum, &txInfo.TxHash, &txInfo.Applyer, &txInfo.Progress, &txInfo.ApplyInfo, &txInfo.ApplyerAppID, &flow_id, &txInfo.CurrencyID, &txInfo.Amount, &txInfo.ApplyAt, &txInfo.Arrived, &updated_at, &txInfo.FlowName)
	if err == sql.ErrNoRows {
		err = nil
		return
	}
	if err != nil {
		return
	}
	txContent := TransferContent{}
	json.Unmarshal([]byte(txInfo.ApplyInfo), &txContent)
	txInfo.Tag = txContent.TxInfo
	if err != nil {
		return
	}
	if txInfo.Progress == 2 {
		txInfo.RejectAt = updated_at
	} else if txInfo.Progress == 3 {
		txInfo.ApprovalAt = updated_at
	}
	return
}

// 获取资产列表
type Asset struct {
	CurrencyName string `json:"currency"`
	Balance      string `json:"balance"`
}

func GetAssets(page, limit int64) (data []Asset, err error) {
	result := []Asset{}
	start := (page - 1) * limit
	assetInfo := Asset{}
	rows, err := db.Conns.Query("select currency, balance from tb_currency where available = 1 limit ?, ?", start, limit)
	defer rows.Close()

	if err != nil {
		return
	}
	for rows.Next() {
		if rows.Err() != nil {
			err = rows.Err()
			return
		}
		err = rows.Scan(&assetInfo.CurrencyName, &assetInfo.Balance)

		if err != nil {
			return
		}
		result = append(result, assetInfo)
	}
	data = result
	return
}

// 获取币种列表
func GetCurrencyList(keywords string) (result []CurrencyInfo, err error) {
	rsp := utils.VoucherStatus{}
	cfg := config.GetConfig()
	data := []CurrencyInfo{}
	// 从代理服务器获取ETH和BTC充值地址
	agentStatus, err := utils.HttpRequest("GET", cfg.Agent.Server+cfg.Agent.TokenDepositAddress, "")

	json.Unmarshal(agentStatus, &rsp)

	// 更新数据库信息
	conn, err := db.Conns.Begin()
	if err != nil {
		return
	}

	// 更新ETH地址
	_, err = conn.Exec("update tb_currency set address = ?, tokenAddress = ? where id = 1", rsp.Status.ContractAddress, rsp.Status.ContractAddress)
	if err != nil {
		conn.Rollback()
		return
	}

	// 将所有币种置为不可用状态
	_, err = conn.Exec("update tb_currency set available = 0 where id <> 1")
	if err != nil {
		conn.Rollback()
		return
	}
	// 更新可用的币种列表
	coinList, err := utils.HttpRequest("GET", cfg.Agent.Server+cfg.Agent.CoinList, "")
	coinRsp := utils.VoucherStatus{}
	json.Unmarshal(coinList, &coinRsp)
	for _, r := range coinRsp.CoinStatus {
		if r.Used == true {
			_, err = conn.Exec("insert into tb_currency (id, currency, factor, address, tokenAddress) values (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE available = ?, address = ?, tokenAddress = ?, currency = ? ", r.Category, r.Name, r.Decimals, rsp.Status.BtcAddress, rsp.Status.BtcAddress, 1, rsp.Status.BtcAddress, rsp.Status.BtcAddress, r.Name)
			if err != nil {
				conn.Rollback()
				return
			}
		}
	}

	// 更新可用的代币列表
	tokenList, err := utils.HttpRequest("GET", cfg.Agent.Server+cfg.Agent.TokenList, "")
	tokenRsp := utils.VoucherStatus{}
	json.Unmarshal(tokenList, &tokenRsp)
	for _, r := range tokenRsp.TokenInfos {
		_, err = conn.Exec("insert into tb_currency (id, currency, factor, isToken, address, tokenAddress) values (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE available = 1, address = ?, tokenAddress = ?, currency = ? ", r.Category, r.TokenName, r.Decimals, 1, rsp.Status.ContractAddress, r.ContractAddr, rsp.Status.ContractAddress, r.ContractAddr, r.TokenName)
		if err != nil {
			conn.Rollback()
			return
		}
	}

	conn.Commit()
	// 获取更新后的币种列表
	var rows *sql.Rows
	if keywords != "" {
		// 搜索
		rows, err = db.Conns.Query("select id, currency, ifnull(address, ''),  ifnull(tokenAddress, '') from tb_currency where available = 1 and currency like ?", "%"+keywords+"%")
		defer rows.Close()
		if err != nil {
			return
		}
	} else {
		rows, err = db.Conns.Query("select id, currency, ifnull(address, ''),  ifnull(tokenAddress, '') from tb_currency where available = 1")
		defer rows.Close()
		if err != nil {
			return
		}
	}

	coinInfoData := CurrencyInfo{}
	for rows.Next() {
		if rows.Err() != nil {
			err = rows.Err()
			return
		}
		err = rows.Scan(&coinInfoData.CurrencyID, &coinInfoData.Currency, &coinInfoData.Addr, &coinInfoData.TokenAddr)

		if err != nil {
			return
		}
		data = append(data, coinInfoData)
	}
	result = data
	return
}

// 获取可用的代币列表
type CoinListInfo struct {
	RspNo      string
	CoinStatus []CoinStatu
}

// 获取可用的代币列表
type TokenListInfo struct {
	RspNo      int
	TokenInfos []TokenInfo
}

type TradeHistory struct {
	PageCount
	List []TradeInfo `json:"list"`
}

// 获取交易记录
func GetTradeHistoryListByAppID(name string, id, page, limit int64) (TradeHistory, error) {
	result := TradeHistory{}
	list := []TradeInfo{}
	start := (page - 1) * limit
	res_count := db.Conns.QueryRow(`select count(*) as total from (
	select distinct(orderNum) from tb_deposit_history where currencyID = ?
		union all
		select distinct(orderNum) from tb_transfer where currencyID = ?) as t`, id, id)
	_ = res_count.Scan(&result.Count)

	if result.Count == 0 {
		result.CurrentPage = page
		result.TotalPage = 1
		//result.List = []TradeInfo{}
		return result, nil
	}
	res, err := db.Conns.Query(`select * from (
      		select distinct(orderNum), amount, ? tx_info, 3 progress, 2 arrived, ? currency, UNIX_TIMESTAMP(updatedAt) as apply_at, 1 type from tb_deposit_history where currencyID = ?
    		union all
      		select distinct(orderNum), amount, txInfo as tx_info , progress, arrived, ? currency,UNIX_TIMESTAMP(createdAt) as apply_at, 0 type from tb_transfer where currencyID = ?) as a
  				order by a.apply_at desc limit ?, ?`, "deposit", name, id, name, id, start, limit)
	defer res.Close()
	if err != nil {
		return TradeHistory{}, err
	}

	for res.Next() {
		if res.Err() != nil {
			err = res.Err()
			return TradeHistory{}, err
		}
		tradeInfo := TradeInfo{}

		err = res.Scan(&tradeInfo.OrderNum, &tradeInfo.Amount, &tradeInfo.Tag, &tradeInfo.Progress, &tradeInfo.Arrived, &tradeInfo.Currency, &tradeInfo.ApplyAt, &tradeInfo.TradeType)

		if err != nil {
			return TradeHistory{}, err
		}
		list = append(list, tradeInfo)
	}
	result.CurrentPage = page
	result.List = list
	result.TotalPage = int64(math.Ceil(float64(result.Count) / float64(limit)))
	return result, nil
}

// 获取转账记录详情
func GetTransferInfoByWdHash(wdhash string) (result TradeInfo, err error) {
	res := db.Conns.QueryRow("select amount, currencyID, arrived, ifnull(txID, '') from tb_transfer where transBoxHash = ?", wdhash)
	err = res.Scan(&result.Amount, &result.CurrencyID, &result.Arrived, &result.TxID)
	return
}

// 记录提现到账信息
func AddTransferArrivedInfo(txInfo TradeInfo, wdhash, txid string, arrived int64) (err error) {
	conn, err := db.Conns.Begin()

	if err != nil {
		return
	}

	_, err = conn.Exec("update tb_transfer set arrived = ?, txID = ? where transBoxHash = ? ", arrived, txid, wdhash)

	if err != nil {
		conn.Rollback()
		return
	}


	// 获取币种信息
	var balance float64
	var factor int64

	err = conn.QueryRow("select balance, factor from tb_currency where id = ?", txInfo.CurrencyID).Scan(&balance, &factor)

	if err != nil && err != sql.ErrNoRows {
		conn.Rollback()
		return err
	}
	amt, err := strconv.ParseFloat(txInfo.Amount, 64)

	if err != nil {
		log.Error("转账扣款", err)
		conn.Rollback()
		return err
	}

	// 扣款
	balance = utils.SubFloat64(balance, amt)

	if err != nil {
		conn.Rollback()
		return err
	}

	// 更新余额
	_, err = conn.Exec("update tb_currency set balance = ? where id = ?", balance, txInfo.CurrencyID)

	if err != nil {
		conn.Rollback()
		return err
	}

	conn.Commit()

	return nil
}


func AddTempTransferArrivedInfo(wdhash, txid string, arrived int64) error {
	_, err := db.Conns.Exec("update tb_transfer set arrived = ?, txID = ? where transBoxHash = ?", arrived, txid, wdhash)
	return err
}

func GetCurrencyInfoByID(currencyID string) (currencyInfo CurrencyInfo, err error) {
	currencyInfo = CurrencyInfo{}
	row := db.Conns.QueryRow("select id, factor, currency, balance from tb_currency where id = ? and available = 1", currencyID)
	err = row.Scan(&currencyInfo.CurrencyID, &currencyInfo.Factor, &currencyInfo.Currency, &currencyInfo.Balance)
	return
}

// 记录充值记录
func DepositHistory(orderNum, toAddr, amount, txID string, currencyID int64, fromArry []string) error {
	var txid string
	err := db.Conns.QueryRow("select txID from tb_deposit_history where txID = ?", txID).Scan(&txid)

	if err != nil && err == sql.ErrNoRows {
		conn, err := db.Conns.Begin()

		if err != nil {
			log.Error("Debug", err)
			return err
		}

		// 获取当前余额
		var balance float64
		var factor int64
		amt, err := strconv.ParseFloat(amount, 64)
		err = conn.QueryRow("select balance, factor from tb_currency where id = ?", currencyID).Scan(&balance, &factor)

		if err != nil {
			log.Error("Debug", err)
			conn.Rollback()
			return err
		}
		// 换算充值金额
		amount_converted, err := utils.UnitConversion(amt, factor, 10)

		if err != nil {
			log.Error("换算充值金额", err)
			conn.Rollback()
			return err
		}

		for i:=0; i<len(fromArry);i++ {
			_, err = conn.Exec("insert into tb_deposit_history (orderNum, fromAddr, toAddr, currencyID, amount, txID) values (?, ?, ?, ?, ?, ?)", orderNum, fromArry[i], toAddr, currencyID, amount_converted, txID)

			if err != nil {
				log.Error("Debug", err)
				conn.Rollback()
				return err
			}
		}

		balance = utils.AddFloat64(balance, amount_converted)

		_, err = conn.Exec("update tb_currency set balance = ? where id = ?", balance, currencyID)

		if err != nil {
			log.Error("充值更新余额", err)
			conn.Rollback()
			return err
		}

		conn.Commit()
	}
	return nil
}

// 获取员工待审批的转账信息
func GetTxInfoByApprover(appID string, transID int64) (int64, error) {
	var progress int64
	row := db.Conns.QueryRow(`select rt.comments
	from tb_transfer as t
		left join tb_review_transfer as rt
			on rt.transID = t.id
		left join tb_accounts_info as acc
			on acc.id = rt.managerAccID
	where acc.appAccountID = ? and t.id = ?`, appID, transID)
	err := row.Scan(&progress)

	if err != nil {
		if err == sql.ErrNoRows {
			return -1, nil
		}
		return -1, err
	}
	return progress, nil
}

// 提交审批意见
func ApprovalTransfer(transID, mAccID int64, progress int, signature, reason string) error {
	if progress != TX_REJECTED {
		reason = ""
	}
	_, err := db.Conns.Exec("update tb_review_transfer set comments = ?, sign = ?, reason = ? where transID = ? and managerAccID = ?", progress, signature, reason, transID, mAccID)
	return err
}

// 更新订单进度
func UpdateTxProgress(transID, progress int64) error {
	var arrived int
	if progress == TX_APPROVALED {
		arrived = TX_DOING
	}

	if progress == TX_REJECTED {
		arrived = TX_END
	}

	_, err := db.Conns.Exec("update tb_transfer set progress = ?, arrived = ? where id = ?", progress, arrived, transID)

	return err
}

func GetTxProgress(flowContent []Approvalinfo, transID int64) (int, error) {
	for i := 0; i < len(flowContent); i++ {
		var appr, rej int
		require := flowContent[i].Require
		approvers := flowContent[i].Approvers
		for j := 0; j < len(approvers); j++ {
			comment, err := GetTxInfoByApprover(approvers[j].AppID, transID)
			if err != nil {
				return TX_REJECTED, err
			}

			if comment == TX_REJECTED {
				rej++
			}

			if comment == TX_APPROVALED {
				appr++
			}
		}
		if rej > len(approvers)-require {
			return TX_REJECTED, nil
		} else if appr < require && appr+rej < len(approvers) {
			return TX_DOING, nil
		}
	}
	return TX_APPROVALED, nil
}

// 转账失败
func TransferFailed(transID int64) error {
	res, err := db.Conns.Prepare("update tb_transfer set progress = 3, arrived = -1 where id = ?")
	defer res.Close()

	if err != nil {
		return err
	}
	_, err = res.Exec(transID)
	return err
}

// 查询转账额度
func TransferLimitByCurrencyID(currencyID int64, flowID string) (AmountLeftInfo, error) {
	var amountLeft AmountLeftInfo
	err := db.Conns.QueryRow("SELECT amountLeft, frozen, ifnull(UNIX_TIMESTAMP(frozenTo), 0), period, amount FROM tb_flow_limit WHERE flowID = ? AND currencyID = ? ", flowID, currencyID).Scan(&amountLeft.AmountLeft, &amountLeft.Frozen, &amountLeft.FrozenTo, &amountLeft.Period, &amountLeft.Amount)
	if err != nil && err == sql.ErrNoRows {
		amountLeft.AmountLeft = "-1"
		return amountLeft, nil
	}
	return amountLeft, err
}

// 初始化额度
func InitTxAmount(currencyID, frozenTo int64, flowID string, amount_left float64) error {
	_, err := db.Conns.Exec("INSERT INTO tb_flow_limit (flowID, currencyID, amountLeft, frozenTo) VALUES (?, ?, ?, FROM_UNIXTIME(?))", flowID, currencyID, amount_left, frozenTo)
	return err
}

// 更新额度
func UpdateTxAmount(currencyID int64, flowID string, new_amount_left float64, amountExp string) error {
	var err error
	period, _ := strconv.Atoi(amountExp)
	frozen := 0
	if new_amount_left == 0 {
		frozen = 1
	}else {
		frozen = 0
	}
	frozenTo := time.Now().Add(time.Duration(period)*time.Hour).Unix()
	//if frozenTo is NULL ,update frozenTo data;else use old data.
	_, err = db.Conns.Exec("UPDATE tb_flow_limit SET amountLeft = ?, frozen = ?, frozenTo = IF(frozenTo IS NULL ,FROM_UNIXTIME(?),frozenTo)  WHERE flowID = ? AND currencyID = ?", new_amount_left, frozen, frozenTo, flowID, currencyID)
	if err != nil {
		log.Error("UpdateTxAmount sql exec error:%v",err)
	}
	return err
}

// 重置额度
func ResetTxAmount(currencyID int64, flowID string, amount float64) error {
	//period, _ := strconv.Atoi(amountExp)
	_, err := db.Conns.Exec("UPDATE tb_flow_limit SET amountLeft = ?, frozen = ?, frozenTo = ? WHERE flowID = ? AND currencyID = ?", amount, 0, sql.NullString{}, flowID, currencyID)
	return err
}

// 撤销转账申请
func CancelTxApply(transID, accID int64, reason string) error {
	log.Debug("撤回转账, transID = %v, accID = %v, reason = %v", transID, accID, reason)
	trans_count := 0
	err := db.Conns.QueryRow("SELECT count(transID) FROM tb_review_transfer WHERE transID = ? AND managerAccID = ?", transID, accID).Scan(&trans_count)
	if trans_count == 0 {
		_, err = db.Conns.Exec("INSERT INTO tb_review_transfer (transID, managerAccID, comments, reason) VALUES (?, ?, ?, ?)", transID, accID, TX_CANCEL, reason)
	} else {
		_, err = db.Conns.Exec("UPDATE tb_review_transfer SET comments = ?, reason = ? WHERE transID = ? AND managerAccID = ?", TX_CANCEL, reason, transID, accID)
	}
	return err
}


// 获取指定转账申请的操作记录
func GetTxOperationHistoryByNum(transNum string) ([]OpLog, error) {
	logs := []OpLog{}
	tx_final_log := OpLog{}
	//var flag int
	// 转账申请时间
	tx_apply_log := OpLog{}
	tx_apply_res := db.Conns.QueryRow("select acc.account, UNIX_TIMESTAMP(tx.createdAt), 0 progress from tb_transfer tx left join tb_accounts_info acc on acc.id = tx.applyerID where tx.orderNum = ?", transNum)
	err := tx_apply_res.Scan(&tx_apply_log.Operator, &tx_apply_log.OpTime, &tx_apply_log.FinalProgress)

	if err != nil {
		return nil, err
	}
	logs = append(logs, tx_apply_log)

	// 各级审批人员操作日志
	res, err := db.Conns.Query("select acc.account, txr.comments, ifnull(txr.reason, ''), UNIX_TIMESTAMP(txr.createdAt), tx.progress from tb_review_transfer txr left join tb_accounts_info acc on acc.id = txr.managerAccID left join tb_transfer tx on tx.id = txr.transID where tx.orderNum = ? ", transNum)

	defer res.Close()
	if err != nil {
		return nil, err
	}

	for res.Next() {
		if res.Err() != nil {
			return nil, res.Err()
		}

		tx_op_log := OpLog{}

		res.Scan(&tx_op_log.Operator, &tx_op_log.Progress, &tx_op_log.Reason, &tx_op_log.OpTime, &tx_op_log.FinalProgress)
		if tx_op_log.Progress != 0 && tx_op_log.Progress != -1 {
			logs = append(logs, tx_op_log)
		}

		if tx_op_log.FinalProgress == TX_APPROVALED || tx_op_log.FinalProgress == TX_REJECTED || tx_op_log.FinalProgress == TX_CANCEL || tx_op_log.FinalProgress == TX_INVALID{
			db.Conns.QueryRow("select UNIX_TIMESTAMP(updatedAt), progress from tb_transfer where orderNum = ?", transNum).Scan(&tx_final_log.OpTime, &tx_final_log.FinalProgress)
			//if  tx_op_log.Progress == 0 && tx_op_log.FinalProgress == TX_REJECTED{
			//	logs = append(logs, tx_op_log)
			//}
			tx_final_log.Progress = tx_final_log.FinalProgress
		}
		//if tx_op_log.FinalProgress != TX_WAITING && tx_op_log.FinalProgress != TX_DOING{
		//	tx_final_log := OpLog{}
		//	db.Conns.QueryRow("select UNIX_TIMESTAMP(updatedAt), progress from tb_transfer where orderNum = ?", transNum).Scan(&tx_final_log.OpTime, &tx_final_log.FinalProgress)
		//
		//	logs = append(logs, tx_final_log)
		//
		//} else {
		//	if tx_op_log.Progress != TX_WAITING {
		//		logs = append(logs, tx_op_log)
		//	}
		//}

	}
	if tx_final_log.OpTime != 0 {
		logs = append(logs, tx_final_log)
	}

	// 最终结果
	//if flag != 0 {
	//	tx_final_log := OpLog{}
	//	db.Conns.QueryRow("select UNIX_TIMESTAMP(updatedAt), progress from tb_transfer where orderNum = ?", transNum).Scan(&tx_final_log.OpTime, &tx_final_log.FinalProgress)
	//
	//	logs = append(logs, tx_final_log)
	//}
	//tx_res, err := db.Conns.QueryRow("SELECT ")

	return logs, nil
}

// 作废转账申请
func InvalidTx(flowid string) error {
	_, err := db.Conns.Exec("UPDATE tb_transfer SET progress = ? WHERE flowID = ? AND progress IN (?, ?)", TX_INVALID, flowid, TX_WAITING, TX_DOING)

	return err
}

// 释放转账额度
func ReleaseAmount(period string, currencyID int64, flowID string, amount float64) error {
	period_i, _ := strconv.ParseInt(period, 10, 64)
	frozenTo := time.Now().Add(time.Duration(period_i)*time.Hour).Unix()
	_, err := db.Conns.Exec("UPDATE tb_flow_limit SET frozenTo = IF(frozenTo IS NULL ,FROM_UNIXTIME(?),frozenTo), amountLeft = ? WHERE flowID = ? AND currencyID = ?", frozenTo, amount, flowID, currencyID)
	return err
}