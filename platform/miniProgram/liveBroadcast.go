package miniProgram

// The live broadcast endpoints (小程序直播) all ride the app access token and
// report failures as an errcode. What they share lives here: the enums WeChat
// uses across the group, the errcode wording, and the goods payload that both
// AddLiveGoods and UpdateLiveGoods send.

// Live room types, as accepted by CreateLiveRoom and reported by
// GetLiveRoomList.
const (
	// LiveRoomTypePhone is a room broadcast from the phone.
	LiveRoomTypePhone = 0
	// LiveRoomTypePush is a room broadcast through third-party push software.
	LiveRoomTypePush = 1
)

// Live room states reported by GetLiveRoomList.
const (
	LiveRoomStatusLiving     = 101
	LiveRoomStatusNotStarted = 102
	LiveRoomStatusEnded      = 103
	LiveRoomStatusBanned     = 104
	LiveRoomStatusPaused     = 105
	LiveRoomStatusAbnormal   = 106
	LiveRoomStatusExpired    = 107
)

// Live member roles, as accepted by AddLiveRole and DeleteLiveRole and reported
// by GetLiveRoleList.
const (
	LiveRoleSuperAdmin = 0
	LiveRoleAdmin      = 1
	LiveRoleAnchor     = 2
	LiveRoleOperator   = 3
)

// Live goods price types. The price pair means something different in each one,
// so the caller has to pick the type before filling price and price2.
const (
	LiveGoodsPriceTypeFixed    = 1
	LiveGoodsPriceTypeRange    = 2
	LiveGoodsPriceTypeDiscount = 3
)

// Live goods audit states, as accepted by GetLiveGoodsList and reported by
// GetLiveGoodsAuditInfo.
const (
	LiveGoodsAuditPending   = 0
	LiveGoodsAuditReviewing = 1
	LiveGoodsAuditApproved  = 2
	LiveGoodsAuditRejected  = 3
)

// liveBroadcastErrCodes holds every errcode the live broadcast documentation
// describes, keyed by the errcode as it appears in the reply.
//
// The endpoints do not repeat this table: each one picks the codes WeChat lists
// for it through liveErrCodes, so the wording has a single source of truth.
var liveBroadcastErrCodes = map[int]string{
	-1:      "系统繁忙，请稍后再试",
	1001:    "请求参数非法",
	1007:    "未获取到数据",
	200002:  "参数错误",
	300002:  "名称长度不符合规则",
	300003:  "价格输入不合规",
	300004:  "商品名称存在违规违法内容",
	300006:  "图片上传失败（如 mediaID 过期）",
	300007:  "线上小程序版本不存在该链接",
	300009:  "商品审核撤回失败",
	300010:  "商品审核状态不对",
	300011:  "操作非法（API 不允许操作非 API 创建的商品）",
	300017:  "商品未提审",
	300018:  "商品图片尺寸过大",
	300021:  "商品添加成功，审核失败",
	300022:  "此房间号不存在",
	300023:  "房间状态拦截（当前房间状态不允许此操作）",
	300024:  "商品不存在",
	300025:  "商品审核未通过",
	300027:  "导入商品失败",
	300028:  "房间名称违规",
	300030:  "主播微信号不合法",
	300031:  "直播间封面图不合规",
	300032:  "直播间分享图违规",
	300033:  "添加商品超过直播间上限",
	300034:  "主播微信昵称长度不符合要求",
	300036:  "主播微信号未实名认证",
	300037:  "购物直播频道封面图不合规",
	300038:  "未在小程序管理后台配置客服",
	300039:  "主播副号微信号不合法",
	300040:  "名称含有非限定字符",
	300041:  "创建者微信号不合法",
	300044:  "商品没有讲解视频",
	300045:  "讲解视频未生成",
	300047:  "已有商品正在推送，请稍后再试",
	300048:  "拉取商品列表失败",
	300049:  "商品推送过程中不允许上下架",
	300052:  "已下架的商品无法推送",
	300053:  "直播间未添加此商品",
	400001:  "微信号不合规",
	400002:  "微信号需要实名认证",
	400003:  "添加角色达到上限",
	400004:  "重复添加角色",
	400005:  "主播角色删除失败，该主播存在未开播的直播间",
	500002:  "副号未实名",
	500004:  "不能设置重复的副号",
	500005:  "副号不能和主号重复",
	9410005: "当日另外一个房间已推送过",
	9410006: "推送的直播间非直播中状态",
	40001:   "access_token 无效",
	41001:   "缺少 access_token",
	42001:   "access_token 已过期",
	43001:   "需要使用 GET 请求",
	45009:   "接口调用超过限额",
	45011:   "调用太频繁，请稍后再试",
	47001:   "数据格式错误",
	48001:   "接口未授权",
}

// liveErrCodes builds the errcode table of a single live broadcast endpoint
// from the codes WeChat documents for it.
func liveErrCodes(codes ...int) map[int]string {
	table := make(map[int]string, len(codes))
	for _, code := range codes {
		if msg, ok := liveBroadcastErrCodes[code]; ok {
			table[code] = msg
		}
	}

	return table
}

// IntPtr returns a pointer to v.
//
// It exists for the optional 0/1 switches of the live room endpoints, where an
// omitted field means "let WeChat decide" and 0 is an explicit "off".
func IntPtr(v int) *int {
	return &v
}

// LiveGoodsInfo is the goods payload of AddLiveGoods and UpdateLiveGoods.
//
// GoodsID is only meaningful for an update: an update of an approved goods may
// change the price type and the prices only, and an update of one still under
// review is rejected outright.
type LiveGoodsInfo struct {
	// CoverImgURL is a mediaID from UploadTempMedia, valid for three days. The
	// image must not exceed 300x300 pixels.
	CoverImgURL string `json:"coverImgUrl"`

	// Name is the goods name, at most 14 Chinese characters.
	Name string `json:"name"`

	// PriceType is LiveGoodsPriceTypeFixed, Range or Discount.
	PriceType int `json:"priceType"`

	// Price is the price in yuan, at most two decimals. Its meaning depends on
	// PriceType: the only price, the lower bound, or the original price.
	Price float64 `json:"price"`

	// Price2 is the upper bound or the current price; required unless PriceType
	// is LiveGoodsPriceTypeFixed.
	Price2 float64 `json:"price2,omitempty"`

	// URL is the mini program path of the goods detail page. Parameters in the
	// path must be URL-encoded already.
	URL string `json:"url"`

	// ThirdPartyAppID is the appid of the mini program that owns the goods, and
	// stays empty for goods of this mini program.
	ThirdPartyAppID string `json:"thirdPartyAppid,omitempty"`

	// GoodsID identifies the goods to update.
	GoodsID int64 `json:"goodsId,omitempty"`
}
