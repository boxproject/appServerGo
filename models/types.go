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

const (
	MAX_ATTEMPTS = 5		// 密码最多尝试次数
	FROZEN_HOUR = 8			// 账户冻结周期
	TOKEN_EXP = 24 			// token有效周期
)

const (
	TXTYPE_ID = 0			// 根据id获取转账信息
	TXTYPE_ORDERNUM = 1		// 根据uuid获取转账信息
)

// 审批流状态
const (
	FLOW_PENDING = 0		// 待审批
	FLOW_REJECTED = 2		// 被拒绝
	FLOW_APPROVALED = 3		// 审批通过
	FLOW_INVALID = 4		// 作废
)

// 转账审批进度
const (
	TX_ALL = -1				// 所有状态
	TX_WAITING = 0			// 待审批
	TX_DOING = 1			// 审批中,转账中
	TX_REJECTED = 2			// 拒绝
	TX_APPROVALED = 3		// 审批通过
	TX_CANCEL = 4			// 撤销
	TX_INVALID = 5			// 作废
	TX_END = -1 			// 审批拒绝，转账失败
)
// user
/*=================================================================*/
// 账户信息
type AccountInfo struct {
	RegID        string `json:"reg_id,omitempty"`
	Pubkey       string `json:"pub_key,omitempty"`
	EnPubkey     string `json:"msg,omitempty"`
	CipherText   string `json:"cipher_text,omitempty"`
	AccLft       int64  `json:"lft,omitempty"`
	AccRgt       int64  `json:"rgt,omitempty"`
	Uploaded     bool   `json:"is_uploaded,omitempty"`
	Departured   bool   `json:"departured,omitempty"`
	ManagerAppID string `json:"manager_account_id"`
	ID 		int64    	`json:"id,omitempty"`
	AppID 	string		`json:"app_account_id,omitempty"`
	Account string 		`json:"account,omitempty"`
	Depth 	int64		`json:"depth,omitempty"`
	EmployeeNum int64 	`json:"employee_num"`
	BranchID int64
	BranchName string
	timeBase
	Frozen bool
	Attempts int
	FrozenTo int64
}

// 账号简要信息
type AccBriefInfo struct {
	ID 		int64
	AppID 	string
	Account string
	Depth 	int64
	EmployeeNum int64
	BranchID int64
}

type timeBase struct {
	ApplyAt    int64 `json:"apply_at"`
	ApprovalAt int64 `json:"approval_at"`
	RejectAt   int64 `json:"reject_at"`
}

// 注册信息
type RegistrationInfo struct {
	ID               int64  `json:"id,omitempty"`
	RegID            string `json:"reg_id,omitempty"`
	Msg              string `json:"msg,omitempty"`
	ApplicantID      string `json:"applyer_id,omitempty"`
	CaptainID        string `json:"manager_id,omitempty"`
	ApplicantAccount string `json:"applyer_account,omitempty"`
	Consent          string `json:"consent,omitempty"`
	timeBase
}

// 指定账户对应的注册信息
type regInfoWithAcc struct {
	RegistrationInfo
	Depth      int64  `json:"depth"`
	CipherText string `json:"cipher_text"`
	LeaderID   string `json:"captain_id,omitempty"`
}

type PageCount struct {
	Count       int64 `json:"count"`
	TotalPage   int64 `json:"total_pages"`
	CurrentPage int64 `json:"current_page"`
}


//business
/*=================================================================*/
// 审批流信息
type FlowInfo struct {
	ID           string         `json:"id"`
	FlowID       string         `json:"flow_id"`
	Hash         string         `json:"flow_hash"`
	Name         string         `json:"flow_name"`
	Progress     int64          `json:"progress"`
	FounderID    int64 		    `json:"founder_id"`
	ApprovalInfo []Approvalinfo `json:"approval_info"`
	SingleLimit  string         `json:"single_limit"`
	Period 		 string 		`json:"period"`
	CreatedAt    string         `json:"created_at"`
	CreatedBy	 string 		`json:"createdBy"`
	UpdatedAt    string         `json:"updated_at"`
	ApprovalAt   string         `json:"approval_at"`
	FlowLimit		[]flowLimit	`json:"flow_limit"`
	PendingTxNum int64 			`json:"pending_tx_num"`
}

type flowLimit struct {
	CurrencyName string 		`json:"currency"`
	Limit 	   	 string 		`json:"limit"`
}

type AmountLeftInfo struct {
	FlowID string
	CurrencyID int64
	Frozen bool
	FrozenTo int64
	AmountLeft string
	Period int64
	Amount string
}

type FlowContent struct {
	Name         string         `json:"flow_name"`
	SingleLimit  string         `json:"single_limit"`
	ApprovalInfo []Approvalinfo `json:"approval_info"`
}

type RPCFlowInfo struct {
	Hash string
	Name string
	AppId string
	CaptainId string
	Flow string
	Sign string
	Status string
}

type Approvalinfo struct {
	Require   int        `json:"require"`
	Approvers []Approver `json:"approvers"`
	Total     int64      `json:"total"`
}

type Approver struct {
	ID       int64  `json:"id"`
	Account  string `json:"account"`
	ItemType int    `json:"itemType"`
	PubKey   string `json:"pub_key"`
	AppID    string `json:"app_account_id"`
	Progress int64  `json:"progress"`
	Sign     string `json:"sign"`
}

type Location struct {
	Number  int
	Level   int
	Require int
}


// 部门
type Branch struct {
	ID int64
	Name string
	Index int64
	Employees int64
	CreatedAt string
	UpdatedAt string
	EmployeesList []AccBriefInfo
}

//capital
/*=================================================================*/
// 转账记录
type TxRecordInfo struct {
	txBase
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
	ApplyAt  int64  `json:"apply_at"`
}

type TransferRecordInfo struct {
	PageCount
	List []TxRecordInfo `json:"list"`
}

// 转账内容
type TransferContent struct {
	TxInfo         string `json:"tx_info"`    // 申请理由
	ToAddr         string `json:"to_address"` // 目的地址
	Miner          string `json:"miner"`      // 矿工费
	Amount         string `json:"amount"`     // 转账金额
	Currency       string `json:"currency"`   // 币种
	ApplyTimestamp string  `json:"timestamp"`  // 申请时间戳
}

// 币种信息
type CurrencyInfo struct {
	CurrencyID int64  `json:"currency_id"`
	Factor     string `json:"factor"`
	Currency   string `json:"currency"`
	Balance    string `json:"balance"`
	Addr       string `json:"address"`
	TokenAddr  string `json:"tokenAddr"`
}

type txBase struct {
	OrderNum string `json:"order_number"`
	Tag      string `json:"tx_info"`
	Progress int64  `json:"progress"`
	Arrived  int64  `json:"arrived"`
	TxID 	string 	`json:"trans_id"`
}

// 转账信息
type TransferInfo struct {
	TxID         int64            `json:"trans_id"`
	txBase
	TxHash       string           `json:"transfer_hash"`
	Applyer      string           `json:"applyer"`
	ApplyerAppID string           `json:"applyer_uid"`
	timeBase
	ApplyInfo    string           `json:"apply_info"`
	SingleLimit  string           `json:"single_limit"`
	ApprovalInfo []TxApprovalInfo `json:"approvaled_info"`
	CurrencyID	 int64 			  `json:"currency_id"`
	Amount 		 string 		  `json:"amount"`
	FlowName	 string 		  `json:"flow_name"`
}

type TxApprovalInfo struct {
	Approvalinfo
	CurrentProgress int `json:"current_progress"`
}

type CoinStatu struct {
	Name     string
	Category int64
	Decimals int64
	Used     bool
}

type TokenInfo struct {
	TokenName    string
	Decimals     int64
	ContractAddr string
	Category     int64
}

type TradeInfo struct {
	txBase
	Amount     string `json:"amount"`
	Currency   string `json:"currency"`
	ApplyAt    int64  `json:"apply_at"`
	TradeType  int64  `json:"type"`
	CurrencyID int64  `json:"currency_id"`
}

type FlowListInfo struct {
	PageCount
	List []FlowInfo `json:"list"`
}


// api struct
/*=================================================================*/
type EmployeeList struct {
	PageCount
	List []AccountInfo `json:"list"`
}

//type RetError struct {
//	Code int
//	Message string
//}


type OpLog struct {
	Operator string
	Progress int
	Reason	 string
	OpTime 	 int64
	FinalProgress int
}