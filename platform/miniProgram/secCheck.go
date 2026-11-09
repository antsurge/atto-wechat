package miniProgram

// The 内容安全 family -- 文本内容安全识别 and 多媒体内容安全识别 -- answers with the
// same shape: a suggested action and, when something was hit, a label naming what
// was hit. The types live here so the synchronous reply and the asynchronous push
// decode into one structure.

// Values of suggest.
const (
	// SecCheckSuggestPass means the content is clean.
	SecCheckSuggestPass = "pass"
	// SecCheckSuggestRisky means the content must not be shown.
	SecCheckSuggestRisky = "risky"
	// SecCheckSuggestReview means a human has to decide.
	SecCheckSuggestReview = "review"
)

// SecCheckSuggestMap holds the descriptions of the suggest values.
//
// Exported for the same reason the errcode maps are: callers render these without
// copying the table.
var SecCheckSuggestMap = map[string]string{
	SecCheckSuggestPass:   "通过",
	SecCheckSuggestRisky:  "risky（违规，需拦截）",
	SecCheckSuggestReview: "review（可疑，需人工审核）",
}

// SecCheckSuggestMsg returns the description of a suggest value, or "" when the
// value is undocumented.
func SecCheckSuggestMsg(suggest string) string {
	return SecCheckSuggestMap[suggest]
}

// Values of label, the category that was hit.
const (
	// SecCheckLabelNormal is not a hit but the "everything is fine" label.
	SecCheckLabelNormal = 100
	// SecCheckLabelAd is advertising.
	SecCheckLabelAd = 10001
	// SecCheckLabelPolitics is 时政.
	SecCheckLabelPolitics = 20001
	// SecCheckLabelPorn is 色情.
	SecCheckLabelPorn = 20002
	// SecCheckLabelAbuse is 辱骂.
	SecCheckLabelAbuse = 20003
	// SecCheckLabelCrime is 违法犯罪.
	SecCheckLabelCrime = 20006
	// SecCheckLabelFraud is 欺诈.
	SecCheckLabelFraud = 20008
	// SecCheckLabelVulgar is 低俗.
	SecCheckLabelVulgar = 20012
	// SecCheckLabelCopyright is 版权.
	SecCheckLabelCopyright = 20013
	// SecCheckLabelOther is everything else.
	SecCheckLabelOther = 21000
)

// SecCheckLabelMap holds the descriptions of the label values.
//
// The 多媒体 reply documents fewer labels than the 文本 one (no advertising, no
// abuse, ...), but a label means the same thing wherever it appears, so one table
// serves both.
var SecCheckLabelMap = map[int]string{
	SecCheckLabelNormal:    "正常",
	SecCheckLabelAd:        "广告",
	SecCheckLabelPolitics:  "时政",
	SecCheckLabelPorn:      "色情",
	SecCheckLabelAbuse:     "辱骂",
	SecCheckLabelCrime:     "违法犯罪",
	SecCheckLabelFraud:     "欺诈",
	SecCheckLabelVulgar:    "低俗",
	SecCheckLabelCopyright: "版权",
	SecCheckLabelOther:     "其他",
}

// SecCheckLabelMsg returns the description of a label, or "" when the label is
// undocumented.
func SecCheckLabelMsg(label int) string {
	return SecCheckLabelMap[label]
}

// SecCheckResult is the overall verdict of a check.
type SecCheckResult struct {
	// Suggest is one of the SecCheckSuggest* values.
	Suggest string `xml:"suggest" json:"suggest"`
	// Label is one of the SecCheckLabel* values.
	Label int `xml:"label" json:"label"`
}

// Passed reports whether the content may be shown as it is.
func (r SecCheckResult) Passed() bool {
	return r.Suggest == SecCheckSuggestPass
}

// Risky reports whether the content must be blocked.
//
// It is the only value that has to be acted on: review means "hold it back until a
// human decides", which is a different decision from rejecting outright.
func (r SecCheckResult) Risky() bool {
	return r.Suggest == SecCheckSuggestRisky
}

// Review reports whether the content needs a human decision.
func (r SecCheckResult) Review() bool {
	return r.Suggest == SecCheckSuggestReview
}

// LabelMsg returns the description of the hit category, or "" when unknown.
func (r SecCheckResult) LabelMsg() string {
	return SecCheckLabelMsg(r.Label)
}

// SecCheckDetail is one strategy's opinion.
//
// The overall verdict is the strictest of these, but an individual entry can fail
// on its own -- ErrCode is what tells whether this one ran at all.
type SecCheckDetail struct {
	// Strategy is the strategy that produced the entry, e.g. "content_model" or
	// "keyword".
	Strategy string `xml:"strategy" json:"strategy"`
	// ErrCode is 0 when the entry is valid; a non-zero value means this strategy
	// did not produce a verdict and the entry must be ignored.
	ErrCode int `xml:"errcode" json:"errcode"`
	// Suggest is one of the SecCheckSuggest* values.
	Suggest string `xml:"suggest" json:"suggest"`
	// Label is one of the SecCheckLabel* values.
	Label int `xml:"label" json:"label"`
	// Keyword is the custom keyword that was hit, for the keyword strategy.
	Keyword string `xml:"keyword" json:"keyword"`
	// Prob is the confidence of the label, 0-100.
	Prob int `xml:"prob" json:"prob"`
	// Level is the keyword strategy's hit level.
	//
	// It appears in the reply examples but not in the documented field table, so it
	// is read as-is rather than interpreted.
	Level int `xml:"level" json:"level"`
}

// Valid reports whether this strategy produced a usable verdict.
func (d SecCheckDetail) Valid() bool {
	return d.ErrCode == 0
}

// LabelMsg returns the description of the hit category, or "" when unknown.
func (d SecCheckDetail) LabelMsg() string {
	return SecCheckLabelMsg(d.Label)
}
