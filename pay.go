package payment

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	PAYMENTTYPE_OFFLINE   = 0 // 线下支付
	PAYMENTTYPE_WECHATPAY = 1 // 微信支付
	PAYMENTTYPE_ALIYPAY   = 2 // 支付宝
)

type Pay interface {
	getConfig(name string) interface{}
	checkConfig() bool
	makeSign(params map[string]interface{}) string
	checkSign(params map[string]interface{}) bool
	GenderPayUrl(order Order) (string, error)    // 生成支付URL
	PayNotify(notfiy []byte) (*NofiyData, error) // 支付回调
}

type Order struct {
	ExtraParam         string // 公用回传参数
	IP                 string // 订单用户IP
	OrderID            string // 订单ID
	PriceTotal         int    // 单位为分
	ProudctName        string // 产品名称
	ProudctDescription string // 产品描述
	ProductID          int    // 商品ID
}

type NofiyData struct {
	OrderID       string      // 订单号
	TransactionID string      // 支付交易号
	TotalFee      int         // 订单金额(单位分)
	ReturnData    []byte      // 返回回调数据
	AttachData    interface{} // 用户自定义数据
}

type Payment struct {
	Config map[string]interface{}
}

func NewPayment() *Payment {
	return &Payment{}
}

func (this *Payment) Init(types int, params map[string]interface{}) (Pay, error) {
	var p Pay
	switch types {
	case PAYMENTTYPE_WECHATPAY:
		p = NewWechat(params)
	case PAYMENTTYPE_ALIYPAY:
		p = NewAlipay(params)
	default:
		return p, fmt.Errorf("The pay type not valid")

	}

	if !p.checkConfig() {
		return nil, fmt.Errorf("Config check bad")
	}
	return p, nil
}

var mtx sync.Mutex

// MD5加密
func (this *Payment) MD5Sigin(str string) string {
	md5cts := md5.New()
	md5cts.Write([]byte(str))
	return hex.EncodeToString(md5cts.Sum(nil))
}

// 生成指定位数的随机字符串
func (this *Payment) GenerateString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, n)
	rand.Seed(time.Now().Unix())
	for i, _ := range bytes {

		bytes[i] = alphanum[rand.Intn(len(alphanum))]
	}
	return string(bytes)
}
func (this *Payment) StrutToMap(obj interface{}) map[string]interface{} {
	return this.struct2map(obj)
}

func (this *Payment) struct2map(obj interface{}) map[string]interface{} {
	var data = make(map[string]interface{})
	t := reflect.TypeOf(obj)
	v := reflect.ValueOf(obj)
	for i := 0; i < v.NumField(); i++ {
		var filedName string
		if filedName = t.Field(i).Tag.Get("json"); len(filedName) == 0 {
			if filedName = t.Field(i).Tag.Get("xml"); len(filedName) == 0 {
				filedName = t.Field(i).Name
			}

		}
		if filedName == "xml" {
			continue
		}
		data[filedName] = v.Field(i).Interface()
	}
	return data
}

// url排序后生成字符串
func (this *Payment) makeUrl(params map[string]interface{}) string {
	keys := make([]string, len(params))
	i := 0
	for k, _ := range params {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	tmpstr := []string{}
	for _, v := range keys {
		rv := reflect.ValueOf(params[v])
		if rv.Interface() == "" {
			continue
		}
		tmpstr = append(tmpstr, fmt.Sprintf("%s=%v", v, rv.Interface()))
	}

	return strings.Join(tmpstr, "&")

}
