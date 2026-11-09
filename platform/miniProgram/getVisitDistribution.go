package miniProgram

import (
	"context"

	"github.com/antsurge/atto-wechat/core/response"
)

// getVisitDistributionPath reads visit distribution of one day.
const getVisitDistributionPath = "/datacube/getweanalysisappidvisitdistribution"

// Values of index: which distribution a list entry describes.
const (
	// VisitDistributionIndexSource is where the visitors came from.
	VisitDistributionIndexSource = "access_source_session_cnt"
	// VisitDistributionIndexStayTime is how long a visit lasted.
	VisitDistributionIndexStayTime = "access_staytime_info"
	// VisitDistributionIndexDepth is how many pages a visit covered.
	VisitDistributionIndexDepth = "access_depth_info"
)

// VisitDistributionIndexMap holds the descriptions of the index values.
var VisitDistributionIndexMap = map[string]string{
	VisitDistributionIndexSource:   "访问来源分布",
	VisitDistributionIndexStayTime: "访问时长分布",
	VisitDistributionIndexDepth:    "访问深度分布",
}

// VisitDistributionIndexMsg returns the description of an index, or "" when the
// value is undocumented.
func VisitDistributionIndexMsg(index string) string {
	return VisitDistributionIndexMap[index]
}

// VisitDistributionSourceMap maps the key of an access_source_session_cnt entry
// to the entry point it stands for.
//
// The keys are not the scene values from the client: WeChat groups the scenes
// into buckets of its own, and 10 collects everything not listed. Without this
// table the numbers in a source distribution cannot be read.
var VisitDistributionSourceMap = map[int]string{
	1:  "小程序历史列表",
	2:  "搜索",
	3:  "会话",
	4:  "扫一扫二维码",
	5:  "公众号主页",
	6:  "聊天顶部",
	7:  "系统桌面",
	8:  "小程序主页",
	9:  "附近的小程序",
	10: "其他",
	11: "模板消息",
	12: "客服消息",
	13: "公众号菜单",
	14: "APP分享",
	15: "支付完成页",
	16: "长按识别二维码",
	17: "相册选取二维码",
	18: "公众号文章",
	19: "钱包",
	20: "卡包",
	21: "小程序内卡券",
	22: "其他小程序",
	23: "其他小程序返回",
	24: "卡券适用门店列表",
	25: "搜索框快捷入口",
	26: "小程序客服消息",
	27: "公众号下发",
	28: "系统会话菜单",
	29: "任务栏-最近使用",
	30: "长按小程序菜单圆点",
	31: "连 wifi 成功页",
	32: "城市服务",
	33: "微信广告",
	34: "其他移动应用",
	35: "发现入口-我的小程序",
	36: "任务栏-我的小程序",
	37: "微信圈子",
	38: "手机充值",
	39: "H5",
	40: "插件",
	41: "大家在用",
	42: "发现页",
	43: "浮窗",
	44: "附近的人",
	45: "看一看",
	46: "朋友圈",
	47: "企业微信",
	48: "视频",
	49: "收藏",
	50: "微信红包",
	51: "微信游戏中心",
	52: "摇一摇",
	53: "公众号导购消息",
	54: "识物",
	55: "小程序订单",
	56: "小程序直播",
	57: "群工具",
}

// VisitDistributionStayTimeMap maps the key of an access_staytime_info entry to
// its stay-time bucket.
var VisitDistributionStayTimeMap = map[int]string{
	1: "0-2s",
	2: "3-5s",
	3: "6-10s",
	4: "11-20s",
	5: "20-30s",
	6: "30-50s",
	7: "50-100s",
	8: ">100s",
}

// VisitDistributionDepthMap maps the key of an access_depth_info entry to the
// number of pages it stands for.
var VisitDistributionDepthMap = map[int]string{
	1: "1 页",
	2: "2 页",
	3: "3 页",
	4: "4 页",
	5: "5 页",
	6: "6-10 页",
	7: ">10 页",
}

// DistributionBucketMsg returns the label of one bucket.
//
// Which table applies depends on the index of the entry the key came from, so the
// index decides: source, stay time or depth.
func DistributionBucketMsg(index string, key int) string {
	switch index {
	case VisitDistributionIndexSource:
		return VisitDistributionSourceMap[key]
	case VisitDistributionIndexStayTime:
		return VisitDistributionStayTimeMap[key]
	case VisitDistributionIndexDepth:
		return VisitDistributionDepthMap[key]
	default:
		return ""
	}
}

// DatacubeDistributionItem is one bucket of a distribution.
type DatacubeDistributionItem struct {
	// Key is the bucket, read with DistributionBucketMsg and the parent Index.
	Key int `json:"key"`
	// Value is the number of visits in the bucket.
	Value int `json:"value"`
}

// DatacubeVisitDistribution is one distribution: by source, by stay time or by
// depth.
type DatacubeVisitDistribution struct {
	// Index is one of the VisitDistributionIndex* values.
	Index string `json:"index"`
	// ItemList is the buckets of this distribution.
	ItemList []DatacubeDistributionItem `json:"item_list"`
}

// GetVisitDistributionResponse is the reply of the distribution endpoint.
type GetVisitDistributionResponse struct {
	// Common reply parameters.
	response.WechatResponse

	// RefDate is the day the figures belong to, yyyymmdd.
	RefDate string `json:"ref_date"`
	// List holds one entry per distribution type.
	List []DatacubeVisitDistribution `json:"list"`
}

// OK reports whether the call succeeded.
//
// An empty List is a normal answer, not a failure.
func (r *GetVisitDistributionResponse) OK() bool {
	return r != nil && r.ErrCode == 0
}

// Items returns the buckets of one distribution, or nil when the reply does not
// contain it.
//
// Which distributions come back depends on the data available, so a caller that
// needs the source distribution has to check rather than index into List.
func (r *GetVisitDistributionResponse) Items(index string) []DatacubeDistributionItem {
	if r == nil {
		return nil
	}

	for _, distribution := range r.List {
		if distribution.Index == index {
			return distribution.ItemList
		}
	}

	return nil
}

// GetVisitDistributionError turns the errcode from the reply into an error.
func GetVisitDistributionError(r *GetVisitDistributionResponse) error {
	return datacubeError(r.ErrCode)
}

// GetVisitDistribution reads where visits came from, how long they lasted and how
// deep they went, for one day.
//
// begin_date and end_date must be the same day.
func (a *Application) GetVisitDistribution(
	ctx context.Context,
	date DatacubeDateRange,
) (*GetVisitDistributionResponse, error) {
	if err := CheckDatacubeDayRange(date); err != nil {
		return nil, err
	}

	resp, err := a.api.PostJSON(ctx, getVisitDistributionPath, nil, date.body())
	if err != nil {
		return nil, err
	}

	var result GetVisitDistributionResponse
	if err := resp.DecodeJSON(&result); err != nil {
		return nil, err
	}

	return &result, nil
}
