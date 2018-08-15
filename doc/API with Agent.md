# API with Agent

## 1. 代理服务器上报临时提现结果

+ router: /api/v1/capital/withdraw/id
+ 请求方式：POST
+ 参数

| 参数名  |  类型  |      描述      | 必填 | 默认值 | 参考值 |
| :-----: | :----: | :------------: | :--: | :----: | :----: |
| wd_hash | string | 提现信息哈希值 |  是  |   -    | 0x...  |
|  tx_id  | string | 对应公链交易ID |  是  |   -    | 0x...  |

- 返回值

~~~json
{
    "code": 0,
    "message": "通知成功。"
}
~~~

- 错误代码

```json
{
    "1000": "系统异常。"
}
```

## 2. 代理服务器上报最终提现结果

- router: /api/v1/capital/withdraw
- 请求方式：POST
- 参数

| 参数名  |  类型  |      描述      | 必填 | 默认值 | 参考值 |
| :-----: | :----: | :------------: | :--: | :----: | :----: |
| wd_hash | string | 提现信息哈希值 |  是  |   -    | 0x...  |
|  tx_id  | string | 对应公链交易ID |  是  |   -    | 0x...  |

- 返回值

```json
{
    "code": 0,
    "message": "通知成功。"
}
```

- 错误代码

```json
{
    "1000": "系统异常。"
}
```

## 3. 代理服务器上报充值记录

- router: /api/v1/capital/deposit
- 请求方式：POST
- 参数

|   参数名   |  类型  |                  描述                  | 必填 | 默认值 | 参考值 |
| :--------: | :----: | :------------------------------------: | :--: | :----: | :----: |
|    from    | string | 付款方地址(如对应多个地址，以逗号分割) |  是  |   -    | 0x...  |
|     to     | string |               收款方地址               |  是  |   -    | 0x...  |
|   amount   | string |                充值金额                |  是  |   -    |        |
|   tx_id    | string |             对应公链交易ID             |  是  |   -    | 0x...  |
| currencyID |  int   |                 币种ID                 |  是  |   -    |        |

- 返回值

```json
{
    "code": 0,
    "message": "通知成功。"
}
```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1001": "参数不能为空。"
}
```

## 4. 代理服务器上报私钥APP审批注册结果

- router: /api/v1/registrations/admin/approval
- 请求方式：POST
- 参数

|   参数名   |  类型  |            描述            |      必填      | 默认值 | 参考值 |
| :--------: | :----: | :------------------------: | :------------: | :----: | :----: |
|   regid    | string |        注册申请编号        |       是       |   -    |   -    |
|   status   | string |    审批结果 1失败 2成功    |       是       |   -    |   -    |
| ciphertext | string | 上级对该员工公钥的摘要信息 | status=2时必填 |   -    |   -    |
|   pubkey   | string |          下属公钥          | status=2时必填 |   -    |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "提交注册审批意见成功。"
}
```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1003": "未找到该注册申请。"
}
```

## 5. 代理服务器获取交易流水

- router: /api/v1/history/trade
- 请求方式：GET
- 参数

|  参数名  |  类型  |        描述        | 必填 | 默认值 | 参考值 |
| :------: | :----: | :----------------: | :--: | :----: | :----: |
|  appid   | string | 用户账号唯一标识符 |  是  |   -    |   -    |
| currency | string |  币种名称，需大写  |  是  |   -    |   -    |
|   page   |  int   |      分页信息      |  否  |   1    |   -    |
|  limit   |  int   | 分页单页展示数据量 |  否  |   20   |   -    |

- 返回值

```json
{
    "code": 0,
    "data": {
        "count": 					// 数据总量，int
        "total_pages": 				// 总页数，int
        "current_page": 			// 当前页码，int
    	"message": "获取交易记录列表成功",
        "list": [
            {
                "order_number": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",		// 编号
                "tx_info": 		// 转账或充值信息
                "progress": 	// 针对充值: 审批进度 int 3-成功；
        						// 针对转账: 审批进度 int 0-待审批 1-审批中 2-驳回 3-通过 4-撤回
                "arrived": 		// 充值或提现是否到账 int 1-进行中 2-到账
                "trans_id": 	// 对应公链交易号
                "amount": 		// 金额 string
                "currency": 	// 币种名称 string
                "apply_at": 	// 交易创建时间戳 int
                "type": 		// 交易类型 int 0-转账 1-充值
            }
        ]
    }
}
```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1001": "参数不能为空。",
    "1007": "权限不足。"
}
```

## 6. 代理服务器获取账户资金信息

- router: /api/v1/capital/assets
- 请求方式：GET
- 参数

| 参数名 |  类型  |        描述        | 必填 | 默认值 | 参考值 |
| :----: | :----: | :----------------: | :--: | :----: | :----: |
| appid  | string | 用户账号唯一标识符 |  是  |   -    |   -    |
|  page  |  int   |      分页信息      |  否  |   1    |   -    |
| limit  |  int   | 分页单页展示数据量 |  否  |   20   |   -    |

- 返回值

```json
{
    "code": 0,
    "message": "获取余额成功。",
    "data": [
        {
            "currency": 			// 币种名称 string
            "balance": 				// 余额 string
        }
    ]
}
```

- 错误代码

```json
{
    "1000": "系统异常。",
    "1007": "权限不足。"
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
