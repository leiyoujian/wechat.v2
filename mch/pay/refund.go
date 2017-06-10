package pay

import (
	"crypto/md5"
	"fmt"
	"strconv"

	"github.com/chanxuehong/rand"
	"github.com/chanxuehong/util"

	"github.com/chanxuehong/wechat.v2/mch/core"
)

// Refund 申请退款.
//  NOTE: 请求需要双向证书.
func Refund(clt *core.Client, req map[string]string) (resp map[string]string, err error) {
	return clt.PostXML(core.APIBaseURL()+"/secapi/pay/refund", req)
}

type RefundRequest struct {
	NonceStr      string `xml:"nonce_str"`       // 随机字符串，不长于32位。NOTE: 如果为空则系统会自动生成一个随机字符串。
	SignType      string `xml:"sign_type"`       // 签名类型，目前支持HMAC-SHA256和MD5，默认为MD5
	TransactionId string `xml:"transaction_id"`  // 微信生成的订单号，在支付通知中有返回
	OutTradeNo    string `xml:"out_trade_no"`    // 商户侧传给微信的订单号
	OutRefundNo   string `xml:"out_refund_no"`   // 商户系统内部的退款单号，商户系统内部唯一，同一退款单号多次请求只退一笔
	TotalFee      int64  `xml:"total_fee"`       // 订单总金额，单位为分，只能为整数，详见支付金额
	RefundFee     int64  `xml:"refund_fee"`      // 退款总金额，订单总金额，单位为分，只能为整数，详见支付金额
	RefundFeeType string `xml:"refund_fee_type"` // 货币类型，符合ISO 4217标准的三位字母代码，默认人民币：CNY，其他值列表详见货币类型
	RefundAccount string `xml:"refund_account"`  // 退款资金来源
}

type RefundResponse struct {
	TransactionId string `xml:"transaction_id"` // 微信订单号
	OutTradeNo    string `xml:"out_trade_no"`   // 商户系统内部的订单号
	OutRefundNo   string `xml:"out_refund_no"`  // 商户退款单号
	RefundId      string `xml:"refund_id"`      // 微信退款单号
	RefundFee     int64  `xml:"refund_fee"`     // 退款总金额,单位为分,可以做部分退款
	TotalFee      int64  `xml:"total_fee"`      // 订单总金额，单位为分，只能为整数，详见支付金额
	CashFee       int64  `xml:"cash_fee"`       // 现金支付金额，单位为分，只能为整数，详见支付金额

	// 下面字段都是可选返回的(详细见微信支付文档), 为空值表示没有返回, 程序逻辑里需要判断
	FeeType             string `xml:"fee_type"`              // 订单金额货币类型，符合ISO 4217标准的三位字母代码，默认人民币：CNY，其他值列表详见货币类型
	SettlementRefundFee *int64 `xml:"settlement_refund_fee"` // 退款金额=申请退款金额-非充值代金券退款金额，退款金额<=申请退款金额
	SettlementTotalFee  *int64 `xml:"settlement_total_fee"`  // 应结订单金额=订单金额-非充值代金券金额，应结订单金额<=订单金额。
	CashFeeType         string `xml:"cash_fee_type"`         // 货币类型，符合ISO 4217标准的三位字母代码，默认人民币：CNY，其他值列表详见货币类型
	CashRefundFee       *int64 `xml:"cash_refund_fee"`       // 现金退款金额，单位为分，只能为整数，详见支付金额
}

// Refund2 申请退款.
//  NOTE:
//  1. 请求需要双向证书.
//  2. 该函数不支持 代金券 功能, 如果有 代金券 功能请使用 Refund 函数.
func Refund2(clt *core.Client, req *RefundRequest) (resp *RefundResponse, err error) {
	m1 := make(map[string]string, 16)
	m1["appid"] = clt.AppId()
	m1["mch_id"] = clt.MchId()
	if req.NonceStr != "" {
		m1["nonce_str"] = req.NonceStr
	} else {
		m1["nonce_str"] = string(rand.NewHex())
	}
	if req.TransactionId != "" {
		m1["transaction_id"] = req.TransactionId
	}
	if req.OutTradeNo != "" {
		m1["out_trade_no"] = req.OutTradeNo
	}
	m1["out_refund_no"] = req.OutRefundNo
	m1["total_fee"] = strconv.FormatInt(req.TotalFee, 10)
	m1["refund_fee"] = strconv.FormatInt(req.RefundFee, 10)
	if req.RefundFeeType != "" {
		m1["refund_fee_type"] = req.RefundFeeType
	}
	if req.RefundAccount != "" {
		m1["refund_account"] = req.RefundAccount
	}
	if req.SignType != "" {
		// m1["sign_type"] = req.SignType
		m1["sign_type"] = "MD5" // TODO(chanxuehong): 目前只支持 MD5, 后期修改
	}
	m1["sign"] = core.Sign(m1, clt.ApiKey(), md5.New) // TODO(chanxuehong): 目前只支持 MD5, 后期修改

	m2, err := Refund(clt, m1)
	if err != nil {
		return nil, err
	}

	resp = &RefundResponse{
		TransactionId: m2["transaction_id"],
		OutTradeNo:    m2["out_trade_no"],
		OutRefundNo:   m2["out_refund_no"],
		RefundId:      m2["refund_id"],
		FeeType:       m2["fee_type"],
		CashFeeType:   m2["cash_fee_type"],
	}

	if str := m2["refund_fee"]; str != "" {
		if n, err := strconv.ParseInt(str, 10, 64); err != nil {
			err = fmt.Errorf("parse refund_fee:%q to int64 failed: %s", str, err.Error())
			return nil, err
		} else {
			resp.RefundFee = n
		}
	}
	if str := m2["total_fee"]; str != "" {
		if n, err := strconv.ParseInt(str, 10, 64); err != nil {
			err = fmt.Errorf("parse total_fee:%q to int64 failed: %s", str, err.Error())
			return nil, err
		} else {
			resp.TotalFee = n
		}
	}
	if str := m2["cash_fee"]; str != "" {
		if n, err := strconv.ParseInt(str, 10, 64); err != nil {
			err = fmt.Errorf("parse cash_fee:%q to int64 failed: %s", str, err.Error())
			return nil, err
		} else {
			resp.CashFee = n
		}
	}

	if str := m2["settlement_refund_fee"]; str != "" {
		if n, err := strconv.ParseInt(str, 10, 64); err != nil {
			err = fmt.Errorf("parse settlement_refund_fee:%q to int64 failed: %s", str, err.Error())
			return nil, err
		} else {
			resp.SettlementRefundFee = util.Int64(n)
		}
	}
	if str := m2["settlement_total_fee"]; str != "" {
		if n, err := strconv.ParseInt(str, 10, 64); err != nil {
			err = fmt.Errorf("parse settlement_total_fee:%q to int64 failed: %s", str, err.Error())
			return nil, err
		} else {
			resp.SettlementTotalFee = util.Int64(n)
		}
	}
	if str := m2["cash_refund_fee"]; str != "" {
		if n, err := strconv.ParseInt(str, 10, 64); err != nil {
			err = fmt.Errorf("parse cash_refund_fee:%q to int64 failed: %s", str, err.Error())
			return nil, err
		} else {
			resp.CashRefundFee = util.Int64(n)
		}
	}

	// 校验返回参数
	if req.TransactionId != "" && resp.TransactionId != "" && req.TransactionId != resp.TransactionId {
		err = fmt.Errorf("transaction_id mismatch, have: %s, want: %s", resp.TransactionId, req.TransactionId)
		return nil, err
	}
	if req.OutTradeNo != "" && resp.OutTradeNo != "" && req.OutTradeNo != resp.OutTradeNo {
		err = fmt.Errorf("out_trade_no mismatch, have: %s, want: %s", resp.OutTradeNo, req.OutTradeNo)
		return nil, err
	}
	if req.OutRefundNo != "" && resp.OutRefundNo != "" && req.OutRefundNo != resp.OutRefundNo {
		err = fmt.Errorf("out_refund_no mismatch, have: %s, want: %s", resp.OutRefundNo, req.OutRefundNo)
		return nil, err
	}
	if req.TotalFee != resp.TotalFee {
		err = fmt.Errorf("total_fee mismatch, have: %d, want: %d", resp.TotalFee, req.TotalFee)
		return nil, err
	}
	if req.RefundFee != resp.RefundFee {
		err = fmt.Errorf("refund_fee mismatch, have: %d, want: %d", resp.RefundFee, req.RefundFee)
		return nil, err
	}

	return resp, nil
}