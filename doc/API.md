# appServer

> appServer服务API文档
>
> 版本：V0.5
>
> 修改时间：2018-07-25

## 0 多语言切换

+ 需要在每次请求时，在请求头`Header`中设置语言选项，具体为：

|      参数名      |  类型  | 描述 | 必填 | 默认值 | 参考值 |
| :--------------: | :----: | :--: | :--: | :----: | :----: |
| content-language | string |  -   |  是  | zh_cn  |   -    |

+ 对应支持的语言选项为

```json
{
    "zh-Hans": 简体中文,
    "en": 英文
}
```

## 1 下级员工APP递交加密后的注册申请

- router:  /api/v1/registrations
- 请求方式： POST
- 参数：

|     参数名      |  类型  |         描述          | 必填 | 默认值 | 参考值 |
| :-------------: | :----: | :-------------------: | :--: | :----: | :----: |
|       msg       | string | 员工APP提交的加密信息 |  是  |   -    |   -    |
|   applyer_id    | string |   申请者唯一识别码    |  是  |   -    |   -    |
|   captain_id    | string |  直属上级唯一识别码   |  是  |   -    |   -    |
| applyer_account | string |    新注册员工账号     |  是  |   -    |   -    |
|    password     | string | 用户注册时设置的密码  |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "提交信息成功。",
    "data": {
        "reg_id": 		// 服务端申请表ID, string 
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1002": "您已提交注册申请，请耐心等待。",
    "1004": "指定账号不存在。",		// 针对对应上级账号
    "1010": "您的账号已经存在，请勿重复提交注册申请。",
    "1011": "您的账号已被停用。",
    "1018": "该账号已被冻结。",		// 上级账号被冻结
    "1020": "非法token。"
}
~~~

## 2 上级APP轮询注册申请

- router:  /api/v1/registrations/pending
- 请求方式： GET
- 参数：

| 参数名 |  类型  | 描述  | 必填 | 默认值 | 参考值 |
| :----: | :----: | :---: | :--: | :----: | :----: |
| token  | string | token |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取注册申请信息成功。",
    	// 如果当前无注册申请，则data值为null
    "data": {
        [{	
            "reg_id":			// 服务端申请表ID, string
            "msg":				// 加密后的注册信息, string
            "applyer_id":		// 申请者唯一标识符, string
            "applyer_account":  // 申请者账号, string
            "manager_id":		// 直属上级唯一标识符, stirng
            "consent":			// 审批结果, ing 0待审批 1拒绝 2同意
            "apply_at":			// 申请提交时间戳, int64
            "applyer_account":	// 申请者账号 string
    	}]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1004": "指定账号不存在。",
    "1011":	"您的账号已被停用。",
    "1020": "非法token。"
}
~~~

## 3 下级员工APP轮询注册审批结果

- router:  /api/v1/registrations/approval/result
- 请求方式： GET
- 参数：

| 参数名 |  类型  |    描述    | 必填 | 默认值 |                 参考值                 |
| :----: | :----: | :--------: | :--: | :----: | :------------------------------------: |
| reg_id | string | 注册信息ID |  是  |   -    | "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" |

- 返回值

```json
{
    "code": 0,
    "message": "获取授权结果成功。",
    "data": {
        "id":					// 注册申请ID，int64
        "reg_id":				// 服务端申请表ID, string
        "applyer_id": 			// 申请者唯一标识符，string
        "captain_id":			// 直属上级唯一标识符, string
        "msg":					// 扫码注册是提交的加密信息, string
        "consent":				// 审批结果 1拒绝 2同意, string
        "depth":				// 直属上级是否为私钥APP，0是, int64
        "applyer_account":      // 申请者账号, string
        "cipher_text"           // 上级对该账号的公钥的摘要信息,string
    }
    "token":					// 注册成功后系统分发的token
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1003": "未找到该注册申请。"
}
~~~

## 4 上级APP提交对注册申请的审批信息

- router:  /api/v1/registrations/approval
- 请求方式： POST
- 参数：

|     参数名      |  类型  |               描述               | 必填 | 默认值 |                 参考值                 |
| :-------------: | :----: | :------------------------------: | :--: | :----: | :------------------------------------: |
|     reg_id      | string |            注册信息ID            |  是  |   -    | "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" |
|     consent     |  int   |     是否同意 (1拒绝，2同意)      |  是  |   -    |                   -                    |
| applyer_pub_key | string |          新注册员工公钥          |  是  |   -    |                   -                    |
|   cipher_text   | string | 该账号对申请者公钥生成的信息摘要 |  是  |   -    |                   -                    |
|   en_pub_key    | string |   该账号对申请者公钥的签名信息   |  是  |   -    |                   -                    |

- 返回值

```json
{
    "code": 0,
    "message": "提交授权结果成功。"
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1003": "未找到该注册申请。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1011": "您的账号已被停用。",
    "1014": "直属上级账号已被停用。",
    "1020": "非法token。"
}
~~~

## 5.员工反馈上级审核注册结果有误

- router:  /api/v1/registrations/approval/cancel
- 请求方式： POST 
- 参数：

|   参数名   |  类型  |         描述         | 必填 | 默认值 |                 参考值                 |
| :--------: | :----: | :------------------: | :--: | :----: | :------------------------------------: |
|   reg_id   | string |      注册信息ID      |  是  |   -    | "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" |
| applyer_id | string | 申请者账号唯一标识符 |  是  |   -    |                   -                    |
|    sign    | string |       签名信息       |  是  |   -    |                   -                    |

- 返回值

```json
{
    "code": 0,
    "message": "通知成功。"
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1003": "未找到该注册申请。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1007": "权限不足。",
    "1009": "注册失败，请稍候重试。",
    "1011": "您的账号已被停用。",
    "1012": "请求代理服务器失败。",
    "1018": "该账号已被冻结。"
}
~~~

## 6. 用户登录

- router:  /api/v1/accounts/login
- 请求方式： POST
- 参数

|  参数名  |  类型  |        描述        | 必填 | 默认值 | 参考值 |
| :------: | :----: | :----------------: | :--: | :----: | :----: |
|  appid   | string | 用户账号唯一标识符 |  是  |   -    |   -    |
| password | string |      登录密码      |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "登录成功。",
    "data": {
        "token": 			// 登录生成的token,token有效期暂定为8小时
        "attempts":			// errorcode== 1016时提示已尝试次数
        "frozenFo":			// 账户冻结周期，暂定8小时
        "frozenTo":			// errorcode == 1018时提示解冻时间戳
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不能为空。",
    "1004": "指定账号不存在。",
    "1011": "您的账号已被停用。",
    "1016": "密码错误。",
    "1018": "该账号已被冻结。"
}
~~~

## 7. 修改密码

+ router: /api/v1/accounts/passwords/modify
+ 请求方式：POST
+ 参数

| 参数名 |  类型  |   描述   | 必填 | 默认值 | 参考值 |
| :----: | :----: | :------: | :--: | :----: | :----: |
| oldpwd | string | 原始密码 |  是  |   -    |   -    |
| newpwd | string |  新密码  |  是  |   -    |   -    |
| token  | string |  token   |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "登录成功。",
    "data": {
        "token": 			// 登录生成的token,token有效期暂定为8小时
        "attempts":			// errorcode== 1016时提示已尝试次数
        "frozenFo":			// 账户冻结周期，暂定8小时
        "frozenTo":			// errorcode == 1018时提示解冻时间戳
}
```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1001": "参数不能为空。",
    "1004": "指定账号不存在。",
    "1011": "您的账号已被停用。",
    "1016": "密码错误。",
    "1018": "该账号已被冻结。",
    "1020": "非法token"
}
```

## 8. 根节点获取指定非直属下属的公钥信息

- router: /api/v1/employee/pubkeys/info
- 请求方式：GET
- 参数

|       参数名        |  类型  |          描述          | 必填 | 默认值 | 参考值 |
| :-----------------: | :----: | :--------------------: | :--: | :----: | :----: |
| employee_account_id | string | 指定下属账号唯一标识符 |  是  |   -    |   -    |
|        token        | string |         token          |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "登录成功。",
    "data": {
        "applyer": 				// 下属账号唯一标识符 string
        "applyer_account":		// 下属账号用户名 string
        "captain":				// 上级账号唯一标识符 string
        "pub_key":				// 下属公钥 string
        "msg":					// 直属上级对该下属公钥的加密信息 string
        "cipher_text":			// 直属上级对该账号公钥生成的信息摘要 string
        "apply_at":				// 下属账号创建时间戳 int
}
```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1001": "参数不能为空。",
    "1004": "指定账号不存在。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",		// 针对管理员账号
    "1018": "该账号已被冻结。",			// 针对管理员账号
    "1020": "非法token"
}
```

## 9. 上级管理员获取下属员工账号详情

- router:  /api/v1/accounts/info
- 请求方式： GET
- 参数：

|       参数名        |  类型  |          描述          | 必填 | 默认值 | 参考值 |
| :-----------------: | :----: | :--------------------: | :--: | :----: | :----: |
| employee_account_id | string | 指定下属账号唯一标识符 |  是  |   -    |   -    |
|        token        | string |         token          |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取员工账号详情成功。",
    "data": {
        "app_account_id":       // 下属账号唯一标识符
        "cipher_text":          // 上级对该账号公钥的摘要信息
        "employee_accounts_info": [
            {
                "app_account_id":       // 该账号直属下级账号唯一标识符
                "account":              // 账号
                "cipher_text":          // 摘要信息
            }
        ]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",	
    "1018": "该账号已被冻结。",			
    "1020": "非法token"
}
~~~

## 10. 删除/替换员工账号

- router:  /api/v1/employee/account/change
- 请求方式： POST
- 参数：

|       参数名        |  类型  |                描述                | 必填 | 默认值 | 参考值 |
| :-----------------: | :----: | :--------------------------------: | :--: | :----: | :----: |
| employee_account_id | string |       指定下属账号唯一标识符       |  是  |   -    |   -    |
|    cipher_texts     | string |      上级对下属公钥的摘要信息      |  是  |   -    |   -    |
| replacer_account_id | string | 被替换者账号唯一标识符(删除时必填) |  否  |   -    |   -    |
|        sign         | string |              签名信息              |  是  |   -    |   -    |
|        token        | string |               token                |  是  |   -    |   -    |

- 备注

其中`cipher_texts`的结构为：

```json
[{
    "app_account_id":       // 被删除/替换员工直属下属账号唯一标识符
    "cipher_text":          // 新生成的摘要信息
}]
```

- 返回值

```json
{
    "code": 0,
    "message": "操作成功。"
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1007": "权限不足。",
    "1008": "指定下级账号不存在。",
    "1011": "您的账号已被停用。",
    "1013": "指定下属账号已被停用。",
    "1015": "非同级用户账号无法替换。",
    "1018": "该账号已被冻结。",			
    "1020": "非法token"
}
~~~

## 11. 获取余额

- router:  /api/v1/capital/balance
- 请求方式： GET
- 参数：

| 参数名 |  类型  |             描述             | 必填 | 默认值 | 参考值 |
| :----: | :----: | :--------------------------: | :--: | :----: | :----: |
| token  | string |            token             |  是  |   -    |   -    |
|  page  |  int   |        设定分页起始页        |  否  |   1    |   -    |
| limit  |  int   | 设定分页信息单页展示信息数量 |  否  |   20   |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取余额成功。",
    "data": [{
        "currency":             // 币种, string
        "balance":              // 余额，string
    }]
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",
    "1018": "该账号已被冻结。",			
    "1020": "非法token"
}
~~~

## 12. 提交转账申请

- router:  /api/v1/transfer/application
- 请求方式： POST
- 参数：

|   参数名   |  类型  |      描述      | 必填 | 默认值 | 参考值 |
| :--------: | :----: | :------------: | :--: | :----: | :----: |
| apply_info | string |    转账信息    |  是  |   -    |   -    |
|  flow_id   |  int   | 对应审批流编号 |  是  |   -    |   -    |
|    sign    | string |    签名信息    |  是  |   -    |   -    |
|  password  | string | 该账号登录密码 |  是  |   -    |   -    |
|   token    | string |     token      |  是  |   -    |   -    |

- 备注

其中`apply_info`的结构为：

```json
{
    "tx_info":             // 申请理由
    "to_address":          // 目的地址
    "miner":               // 矿工费
    "amount":              // 转账金额
    "currency":            // 币种
    "timestamp":           // 申请时间戳
}
```

- 返回值

```json
{
    "code": 0,
    "message": "提交转账申请成功。",
    "data": {
        "order_number":         // 转账记录编号, string
        "attempts":			// errorcode== 1016时提示已尝试次数
        "frozenFo":			// 账户冻结周期，暂定8小时
        "frozenTo":			// errorcode == 1018时提示解冻时间戳
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1016": "密码错误。",
    "1018": "该账号已被冻结。",			
    "1020": "非法token",
    "2001": "转账信息有误，请查验后重新提交。",
    "2010": "转账金额超过上限，请核对后重新提交。",
    "2011": "不支持对应币种。",
    "2012": "当前币种转账额度已用完。"
}
~~~

## 13. 获取转账记录列表

- router:  /api/v1/transfer/records/list
- 请求方式： GET
- 参数：

|  参数名  |  类型  |                          描述                          | 必填 | 默认值 | 参考值 |
| :------: | :----: | :----------------------------------------------------: | :--: | :----: | :----: |
|   type   |  int   |         转账记录类型，0作为发起者 1作为审批者          |  否  |   0    |   -    |
| progress |  int   | 审批进度  -1所有记录 0待审批 1审批中 2被驳回 3审批成功 |  否  |   0    |   -    |
|   page   |  int   |                列表分页起始页，不小于1                 |  否  |   1    |   -    |
|  limit   |  int   |                    单页显示记录条数                    |  否  |   20   |   -    |
|  token   | string |                         token                          |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取转账列表成功。",
    "data": {
        "count": 		// 总数据量, int
        "total_pages":  // 总页码, int
        "current_page":	// 当前页码, int
        "list": [{
            "order_number":	// 转账记录编号, string
            "tx_info":	    // 申请理由, string
            "amount":       // 转账金额, string
            "currency":     // 币种, string
            "single_limit": // 单笔转账限额, string
            "progress": 	// 审批进度 0待审批 1审批中 2被驳回 3审批成功 4撤回, int
            "apply_at": 	// 该笔转账申请时间戳, int
        }]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1004": "指定账号不存在。",
    "1011": "您的账号已被停用。",
    "1018": "该账号已被冻结。",			
    "1020": "非法token"
}
~~~

## 14. 获取转账记录详情

- router:  /api/v1/transfer/records
- 请求方式： GET
- 参数：

|    参数名    |  类型  |     描述     | 必填 | 默认值 | 参考值 |
| :----------: | :----: | :----------: | :--: | :----: | :----: |
| order_number | string | 转账记录编号 |  是  |   -    |   -    |
|    token     | string |    token     |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取转账信息成功。",
    "data": {
            "transfer_hash":    // 该笔转账对应私链的哈希值, string
            "order_number":	    // 转账记录编号, string
            "tx_info":	        // 申请理由, string
            "applyer":          // 转账申请提交者账号
            "applyer_uid":      // 申请者账号唯一标识符
            "progress":		    // 订单审批总进度 0待审批 1审批中 2被驳回 3审批成功 4已撤回, int
            "apply_at": 	    // 申请提交时间戳, number
            "approval_at": 	    // 审批通过时间戳，默认null, string
            "reject_at": 	    // 审批拒绝时间戳，默认null, string
            "apply_info":       // 申请者提交的转账信息, string
            "single_limit":     // 本次转账单笔限额, string
            "approvaled_info": [{
                "require": 			    // 该层级需要审批通过的最少人数, int
                "total":                // 参与该层审批人员总数, int
                "current_progress":     // 该层当前审批进度, 0待审批 1审批中 2驳回 3同意  int
            	"approvers": [{		  // 审批信息
            		"account":			    // 该审批者账号, string
            		"app_account_id":	    // 该账号唯一标识符, string
            		"sign":				    // 该账号对该笔转账的签名信息, string
            		"progress":			    // 该账号对该笔转账的审批结果 0待审批 2驳回 3同意 4撤回, int
        		}]
        	},
        	...
    		]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1018": "该账号已被冻结。",			
    "1020": "非法token",
    "2005": "未找到对应的转账申请。"
}
~~~

## 15. 提交审批意见

- router:  /api/v1/transfer/approval
- 请求方式： POST
- 参数：

|    参数名    |  类型  |         描述          |       必填        | 默认值 | 参考值 |
| :----------: | :----: | :-------------------: | :---------------: | :----: | :----: |
| order_number | string |     转账记录编号      |        是         |   -    |   -    |
|   progress   |  int   | 审批意见  2驳回 3同意 |        是         |   -    |   -    |
|     sign     | string |       签名信息        |        是         |   -    |   -    |
|   password   | string |     账号登录密码      |        是         |   -    |   -    |
|    reason    | string |   驳回转账请求原因    | progress==2时必填 |   -    |   -    |
|    token     | string |         token         |        是         |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "提交审批意见成功。",
    "data": {
        "attempts":			// errorcode== 1016时提示已尝试次数
        "frozenFo":			// 账户冻结周期，暂定8小时
        "frozenTo":			// errorcode == 1018时提示解冻时间戳    
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1016": "密码错误。",
    "1018": "该账号已被冻结。",			
    "1020": "非法token",
    "2003": "无权审批该转账申请。",
    "2005": "未找到对应的转账申请。",
    "2006": "您已提交审批意见，请勿重复提交。",
    "2007": "转账信息哈希上链失败，请稍候重试。",
    "2008": "未找到对应币种信息。"
}
~~~

## 16. 获取审批流模板列表

- router:  /api/v1/business/flows/list
- 请求方式： GET
- 参数：

|  参数名   |  类型  |                       描述                        | 必填 | 默认值 | 参考值 |
| :-------: | :----: | :-----------------------------------------------: | :--: | :----: | :----: |
|   type    | string | 审批流状态 1已通过审批 否则获取所有状态审批流列表 |  否  |   -    |   -    |
| key_words | string |                    搜索关键字                     |  否  |   -    |   -    |
|   page    |  int   |                    分页，页码                     |  否  |   1    |   -    |
|   limit   |  int   |               分页，单页显示数据量                |  否  |   20   |   -    |
|   token   | string |                       token                       |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取审批流模板列表成功。",
    "data": {
        "count": 		// 总数据量, int
        "total_pages":  // 总页码, int
        "current_page":	// 当前页码, int
        "list": [
            {
                "flow_id":          // 审批流模板编号, string
                "flow_name":        // 审批流模板名称, string
                "progress":         // 审批流模板审批进度 0待审批 2审批拒绝 3审批通过 4已撤回, int
                "single_limit":     // 单笔转账上限, string (后续版本会废弃该字段)
        		"flow_limit": [		// 针对不同币种的限额信息
        			"currency": 		// 币种名称 string	
        			"limit":			// 限额 string
        		]
            }
        ]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1012": "请求代理服务器失败。",			
    "1020": "非法token",
    "2002": "未找到对应币种。"
}
~~~

## 17. 获取审批流模板详情

- router:  /api/v1/business/flow/info
- 请求方式： GET
- 参数：

| 参数名  |  类型  |      描述      | 必填 | 默认值 | 参考值 |
| :-----: | :----: | :------------: | :--: | :----: | :----: |
| flow_id | string | 审批流模板编号 |  是  |   -    |   -    |
|  token  | string |     token      |  是  |   -    |   -    |

- 返回值

```javascript
{
    "code": 0,
    "message": "获取审批流模板详情成功。",
    "data": {
        "progress":          // 私钥APP对该模板的审批进度 0待审批 2审批拒绝 3审批同意 4已撤回, int
        "createdBy":         // 创建者账号唯一标识符，string
        "flow_name":         // 审批流模板名称
        "flow_limit": [		// 针对不同币种的限额信息
        	"currency": 		// 币种名称 string	
        	"limit":			// 限额 string
        ]
        "approval_info": [
            {
                "require":          // 该层所需最小审批通过人数, int
                "total":            // 参与该层审批者总数, int
                "approvers": [
                 {
                    "account":          // 审批者账号, string
                    "app_account_id":   // 审批者账号唯一标识符, string
                    "pub_key":          // 账号公钥
                 }
                ]
            }
        ]
    }
    
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1020": "非法token"
}
~~~

## 18. 根节点获取非直属下属的公钥信息列表

- router:  /api/v1/employee/pubkeys/list
- 请求方式： GET
- 参数：

| 参数名 |  类型  | 描述  | 必填 | 默认值 | 参考值 |
| :----: | :----: | :---: | :--: | :----: | :----: |
| token  | string | token |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取员工公钥信息成功。",
    "data": [
        {
            "applyer": 		    // 待上传公钥的员工账号唯一标识符, string
            "applyer_account":  // 该员工账号，string
            "pub_key": 		    // 该员工账号的公钥, string
            "captain": 		    // 该员工账号直属上级账号唯一标识符, string
            "msg": 			    // 直属上级对其公钥的加密信息, string
            "cipher_text":      // 直属上级对该账号公钥生成的信息摘要
            "apply_at": 	    // 该员工账号申请创建时间戳, number
        }
    ]
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1006": "未找到对应的业务流程。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",
    "1020": "非法token"
}
~~~

## 19. 上级管理员获取下属账号列表

- router:  /api/v1/accounts/list
- 请求方式： POST
- 参数：

|  参数名   |  类型  |         描述         | 必填 | 默认值 | 参考值 |
| :-------: | :----: | :------------------: | :--: | :----: | :----: |
|   page    |  int   |      分页，页码      |  否  |   1    |   -    |
|   limit   |  int   | 分页，单页显示数据量 |  否  |   20   |   -    |
| key_words | string |      搜索时使用      |  否  |   -    |   -    |
|   token   | string |        token         |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取下属账号列表成功。",
    "data": {
        "count": 		// 总数据量, int
        "total_pages":  // 总页码, int
        "current_page":	// 当前页码, int
        "list":[        // 账号列表信息
            {
                "account":              // 账号，string
                "app_account_id":       // 账号唯一标识符，string
                "manager_account_id":   // 对应上级账号唯一标识符，string
                "cipher_text":          // 上级对该账号公钥生成的信息摘要，string
                "is_uploaded":          // 公钥是否上传到根节点账户, 1是 0否，int
                "employee_num":         // 该账号下属个数，int
            }
        ]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1004": "指定账号不存在。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1020": "非法token"
}
~~~

## 20. 创建审批流模板

- router:  /api/v1/business/flow
- 请求方式： POST
- 参数：

| 参数名 |  类型  |              描述              | 必填 | 默认值 | 参考值 |
| :----: | :----: | :----------------------------: | :--: | :----: | :----: |
|  flow  | string |         审批流模板内容         |  是  |   -    |   -    |
|  sign  | srting | 创建者对审批流模板内容的签名值 |  是  |   -    |   -    |
| token  | string |             token              |  是  |   -    |   -    |

- 备注

其中`flow`的结构为：

```json
{
    "flow_name":            // 审批流模板名称
    "single_limit":         // 单笔限额(v0.5后续版本不再使用)
    "approval_info":[
        {
            "require":          // 该层所需最小审批同意人数
            "total":			// 该层审批者人数
            "approvers"[        // 审批者信息
                {
                    "account":          // 审批者账号
                    "app_account_id":   // 审批者账号唯一标识符
                    "pub_key":          // 审批者公钥
            		"itemType"
                }
            ]
        }
    ],
    "flow_limit" : [		// 审批流支持的币种列表及对应的额度信息
    	{
      	"currency" : 			// 币种名称
      	"limit" : 				// 额度
    	}
  	],
  	"period" : 				// 额度恢复时间，单位为小时，数值范围为0~240
}
```

- 返回值

```javascript
{
    "code": 0,
    "message": "创建审批流模板成功。",
    "data": {
        "flow_id":              // 创建后的审批流编号
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1003": "未找到该注册申请。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1012": "请求代理服务器失败。",
    "1020": "非法token",
    "3002": "指定业务流模板已存在，请勿重复提交。"
}
~~~

## 21. 获取币种列表

- router:  /api/v1/capital/currency/list
- 请求方式： GET
- 参数：

|  参数名   |  类型  |                   描述                   | 必填 | 默认值 | 参考值 |
| :-------: | :----: | :--------------------------------------: | :--: | :----: | :----: |
| key_words | string | 搜索字段，币种名称，若为空则显示全部列表 |  否  |   -    |  ETH   |
|   token   | srting |                  token                   |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取币种列表成功。",
    "data": {
        "currency_list": [{
            "currency_id": 		// 币种编号，int
            "currency":			// 币种名称, string
            "address":          // 收款地址, string
            "tokenAddr":		// 对应代币的合约地址，string
        	},
        ]         
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1004": "指定账号不存在。",
    "1011": "您的账号已被停用。",
    "1020": "非法token"
}
~~~

## 22.获取交易记录列表

- router:  /api/v1/capital/trade/history/list
- 请求方式： GET 
- 参数：

|  参数名  |  类型  |         描述         | 必填 | 默认值 | 参考值 |
| :------: | :----: | :------------------: | :--: | :----: | :----: |
| currency | string |       币种名称       |  是  |   -    |  ETH   |
|   page   |  int   |      分页，页码      |  否  |   1    |   -    |
|  limit   |  int   | 分页，单页显示数据量 |  否  |   20   |   -    |
|  token   | srting |        token         |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取交易记录列表成功",
    "data": {
        "count":                // 总数据量 int
        "total_pages":          // 总页数 int
        "current_page":         // 当前页码 int
        "list": [
            {
                "order_number":     // 订单号 string
                "amount":           // 充值/转账金额 string
                "tx_info":          // 充值/转账信息 string
                "progress":         // 最终审批意见，0待审批 1审批中 2驳回 3审批同意 4撤回 int
                "currency":         // 记录对应的币种名称 string
                "updated_at":       // 更新时间，时间戳 int
                "type":             // 交易类型 1充值 0转账 int
        		"currency_id": 		// 对应币种编号 int
        		"trans_id": 		// 对应公链编号 string
        		"arrived": 			// 是否到账 1-到账 0-未到账 int
            },
            ...
        ]
    }
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1011": "您的账号已被停用。",
    "1020": "非法token",
    "2002": "未找到对应币种。"
}
~~~

## 23. 添加部门

- router:  /api/v1/branch/add
- 请求方式： POST
- 参数：

| 参数名 |  类型  |          描述          | 必填 | 默认值 | 参考值 |
| :----: | :----: | :--------------------: | :--: | :----: | :----: |
|  name  | string |        部门名称        |  是  |   -    |   -    |
|  sign  | string | 签名信息(对name值签名) |  是  |   -    |   -    |
| token  | srting |         token          |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "添加部门成功。"
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",
    "1020": "非法token",
    "3003": "该部门已存在。"
}
~~~

## 24. 删除或修改部门

- router:  /api/v1/branch/change
- 请求方式： POST
- 参数

|  参数名   |  类型  |        描述        | 必填 | 默认值 | 参考值 |
| :-------: | :----: | :----------------: | :--: | :----: | :----: |
|    bid    | string |       部门ID       |  是  |   -    |   -    |
| new_index |  int   |   被修改部门索引   |  否  |   -    |   -    |
| new_name  | string | 被修改后部门名称， |  否  |   -    |   -    |
|   sign    | string |     签名(bid)      |  是  |   -    |   -    |
|   token   | srting |       token        |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "操作成功。",
    "data":{
        "list": [
            {
                "ID": 1,					// 部门ID
                "Name": "",					// 部门名称		
                "Index": 1,					// 部门列表索引
                "Employees": 10,			// 该部门员工数量
                "Available": true,			// 该部门是否被删除
                "CreatedAt": "1530252222",	// 创建时间戳
                "UpdatedAt": "1530254172",	// 更新时间戳
            }
        ]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",
    "1020": "非法token",
    "3005": "所选部门不存在。"
}
~~~

## 25. 获取部门列表

- router:  /api/v1/branch/list
- 请求方式： GET
- 参数

| 参数名 |  类型  | 描述  | 必填 | 默认值 | 参考值 |
| :----: | :----: | :---: | :--: | :----: | :----: |
| token  | string | token |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取部门列表成功。",
    "data": {
        "list": [
            {
                "ID": 1,					// 部门ID
                "Name": "",					// 部门名称		
                "Index": 1,					// 部门列表索引
                "Employees": 10,			// 该部门员工数量
                "Available": true,			// 该部门是否被删除
                "CreatedAt": "1530252222",	// 创建时间戳
                "UpdatedAt": "1530254172",	// 更新时间戳
               	
            }
        ]
	}
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1011": "您的账号已被停用。"
}
~~~

## 26. 修改账号所属部门

- router:  /api/v1/branch/select
- 请求方式： POST
- 参数

| 参数名 |  类型  |  描述  | 必填 | 默认值 | 参考值 |
| :----: | :----: | :----: | :--: | :----: | :----: |
|  bid   | string | 部门ID |  是  |   -    |   -    |
| token  | string | token  |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "选择部门成功。"
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1011": "您的账号已被停用。",
    "3005": "所选部门不存在。"
}
~~~

## 27. 获取部门详情

- router:  /api/v1/branch/info
- 请求方式： GET
- 参数

| 参数名 |  类型  |  描述  | 必填 | 默认值 | 参考值 |
| :----: | :----: | :----: | :--: | :----: | :----: |
|  bid   | string | 部门ID |  是  |   -    |   -    |
| token  | string | token  |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取部门信息详情成功。",
    "data": {
        Name:				// 部门名称
        Employees: 			// 该部门员工数量
        EmployeesList: [{
                ID 			// 该部门下属员工账户编号 int    	
                AppID 		// 该员工账号唯一标识符 string		
                Account 	// 该员工账户名称 string 		
                Depth 		// 该员工所属层级 int		
                EmployeeNum // 该员工直属下级数量 int
            	BranchID 	// 该员工所属部门ID int
    		}
        ]
    }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",
    "3005": "所选部门不存在。"
}
~~~

## 28. 获取账号信息详情

- router:  /api/v1/accounts/detail
- 请求方式： GET
- 参数

| 参数名 |  类型  |              描述              | 必填 | 默认值 | 参考值 |
| :----: | :----: | :----------------------------: | :--: | :----: | :----: |
|  sign  | string | 签名信息(对自身账号唯一标识符) |  是  |   -    |   -    |
| token  | string |             token              |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "操作成功。",
    "data": {
        Department: {
            ID:			// 所属部门ID
            Name:		// 所属部门名称
        }
}
```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1011": "您的账号已被停用。"
}
~~~

## 29.撤销转账申请

- router:  /api/v1/transfer/application/cancel
- 请求方式： POST
- 参数

|    参数名    |  类型  |   描述   | 必填 | 默认值 | 参考值 |
| :----------: | :----: | :------: | :--: | :----: | :----: |
| order_number | string | 订单编号 |  是  |   -    |   -    |
|    reason    | string | 撤回原因 |  是  |   -    |   -    |
|   password   | string | 登录密码 |  是  |   -    |   -    |
|    token     | string |  token   |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "撤回转账申请成功！",
    "data": {
        "attempts":			// errorcode== 1016时提示已尝试次数
        "frozenFo":			// 账户冻结周期，暂定8小时
        "frozenTo":			// errorcode == 1018时提示解冻时间戳    
    }
}

```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",
    "1016": "密码错误。",
    "1018": "该账号已被冻结。",
    "1020": "非法token",
    "2013": "该转账申请已审批完成，无法撤回！"
}
~~~

## 30. 作废审批流

- router:  /api/v1/business/flow/disuse
- 请求方式： POST
- 参数

|  参数名  |  类型  |          描述          | 必填 | 默认值 | 参考值 |
| :------: | :----: | :--------------------: | :--: | :----: | :----: |
| flow_id  | string |     对应审批流编号     |  是  |   -    |   -    |
|   sign   | string | 对审批流名称的签名信息 |  是  |   -    |   -    |
| password | string |        登录密码        |  是  |   -    |   -    |
|  token   | string |         token          |  是  |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "操作成功。",
    "data": {
        "attempts":			// errorcode== 1016时提示已尝试次数
        "frozenFo":			// 账户冻结周期，暂定8小时
        "frozenTo":			// errorcode == 1018时提示解冻时间戳    
    }
}

```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1006": "未找到对应的业务流程。",
    "1007": "权限不足。",
    "1011": "您的账号已被停用。",
    "1016": "密码错误。",
    "1018": "该账号已被冻结。",
    "1020": "非法token",
    "3006": "指定业务流模板尚未通过审批。"
}
```

## 31. 获取审批转账操作日志

- router:  /api/v1/history/transfer/operation
- 请求方式： GET
- 参数

|    参数名    |  类型  |  描述  | 必填 | 默认值 | 参考值 |
| :----------: | :----: | :----: | :--: | :----: | :----: |
| order_number | string | 订单号 |  是  |   -    |   -    |
|    token     | string | token  |  是  |   -    |   -    |

- 返回值

```json
{

    "code": 0,
    "message": "操作成功。"
    "data":[
        {
            "Operator"   // 审批者账号
			"Progress" 	 // 审批意见
			"Reason"	 // 拒绝或撤回的原因
			"OpTime"     // 提交审批意见的时间，时间戳 int64
        }
    ]
}

```

- 错误代码

~~~json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1005": "签名信息错误。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1020": "非法token"
}
~~~

## 32. 获取审批流操作日志

- router:  /api/v1/history/flow/operation
- 请求方式： GET
- 参数

| 参数名  |  类型  |    描述    | 必填 | 默认值 | 参考值 |
| :-----: | :----: | :--------: | :--: | :----: | :----: |
| flow_id | string | 审批流编号 |  是  |   -    |   -    |
|  token  | string |   token    |  是  |   -    |   -    |

- 返回值

```json
{

    "code": 0,
    "message": "操作成功。"
    "data":{
    	"HashOperates": [
    		{
      			"ApplyerAccount": 		// 申请者账号
      			"CaptainId": 			// 操作者账号唯一标识符
     			"Option": 				
      			"Opinion": 				
      			"CreateTime": 			// 审批流创建时间 string(yyyy-mm-dd hh:mm:ss)
    		},
			...
  		]

	}
}

```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1001": "参数不完整。",
    "1004": "指定账号不存在。",
    "1006": "未找到对应的业务流程。",
    "1011": "您的账号已被停用。",
    "1020": "非法token"
}
```

## Licence

Licensed under the Apache License, Version 2.0, Copyright 2018. box.la authors.

```
 Copyright 2018. box.la authors.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
```

