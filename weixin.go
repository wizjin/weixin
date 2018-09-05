// Package weixin MP SDK (Golang)
package weixin

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"
)

// nolint
const (
	// Event type
	msgEvent = "event"

	EventSubscribe    = "subscribe"
	EventUnsubscribe  = "unsubscribe"
	EventScan         = "SCAN"
	EventView         = "VIEW"
	EventClick        = "CLICK"
	EventLocation     = "LOCATION"
	EventTemplateSent = "TEMPLATESENDJOBFINISH"

	// Message type
	MsgTypeDefault           = ".*"
	MsgTypeText              = "text"
	MsgTypeImage             = "image"
	MsgTypeVoice             = "voice"
	MsgTypeVideo             = "video"
	MsgTypeShortVideo        = "shortvideo"
	MsgTypeLocation          = "location"
	MsgTypeLink              = "link"
	MsgTypeEvent             = msgEvent + ".*"
	MsgTypeEventSubscribe    = msgEvent + "\\." + EventSubscribe
	MsgTypeEventUnsubscribe  = msgEvent + "\\." + EventUnsubscribe
	MsgTypeEventScan         = msgEvent + "\\." + EventScan
	MsgTypeEventView         = msgEvent + "\\." + EventView
	MsgTypeEventClick        = msgEvent + "\\." + EventClick
	MsgTypeEventLocation     = msgEvent + "\\." + EventLocation
	MsgTypeEventTemplateSent = msgEvent + "\\." + EventTemplateSent

	// Media type
	MediaTypeImage = "image"
	MediaTypeVoice = "voice"
	MediaTypeVideo = "video"
	MediaTypeThumb = "thumb"
	// Button type
	MenuButtonTypeKey             = "click"
	MenuButtonTypeUrl             = "view"
	MenuButtonTypeScancodePush    = "scancode_push"
	MenuButtonTypeScancodeWaitmsg = "scancode_waitmsg"
	MenuButtonTypePicSysphoto     = "pic_sysphoto"
	MenuButtonTypePicPhotoOrAlbum = "pic_photo_or_album"
	MenuButtonTypePicWeixin       = "pic_weixin"
	MenuButtonTypeLocationSelect  = "location_select"
	MenuButtonTypeMediaId         = "media_id"
	MenuButtonTypeViewLimited     = "view_limited"
	MenuButtonTypeMiniProgram     = "miniprogram"
	// Template Status
	TemplateSentStatusSuccess      = "success"
	TemplateSentStatusUserBlock    = "failed:user block"
	TemplateSentStatusSystemFailed = "failed:system failed"
	// Redirect Scope
	RedirectURLScopeBasic    = "snsapi_base"
	RedirectURLScopeUserInfo = "snsapi_userinfo"
	// Weixin host URL
	weixinHost               = "https://api.weixin.qq.com/cgi-bin"
	weixinQRScene            = "https://api.weixin.qq.com/cgi-bin/qrcode"
	weixinShowQRScene        = "https://mp.weixin.qq.com/cgi-bin/showqrcode"
	weixinMaterialURL        = "https://api.weixin.qq.com/cgi-bin/material"
	weixinShortURL           = "https://api.weixin.qq.com/cgi-bin/shorturl"
	weixinUserInfo           = "https://api.weixin.qq.com/cgi-bin/user/info"
	weixinFileURL            = "http://file.api.weixin.qq.com/cgi-bin/media"
	weixinTemplate           = "https://api.weixin.qq.com/cgi-bin/template"
	weixinRedirectURL        = "https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=%s&state=%s#wechat_redirect"
	weixinUserAccessTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code"
	weixinJsApiTicketURL     = "https://api.weixin.qq.com/cgi-bin/ticket/getticket"
	// Max retry count
	retryMaxN = 3
	// Reply format
	replyText               = "<xml>%s<MsgType><![CDATA[text]]></MsgType><Content><![CDATA[%s]]></Content></xml>"
	replyImage              = "<xml>%s<MsgType><![CDATA[image]]></MsgType><Image><MediaId><![CDATA[%s]]></MediaId></Image></xml>"
	replyVoice              = "<xml>%s<MsgType><![CDATA[voice]]></MsgType><Voice><MediaId><![CDATA[%s]]></MediaId></Voice></xml>"
	replyVideo              = "<xml>%s<MsgType><![CDATA[video]]></MsgType><Video><MediaId><![CDATA[%s]]></MediaId><Title><![CDATA[%s]]></Title><Description><![CDATA[%s]]></Description></Video></xml>"
	replyMusic              = "<xml>%s<MsgType><![CDATA[music]]></MsgType><Music><Title><![CDATA[%s]]></Title><Description><![CDATA[%s]]></Description><MusicUrl><![CDATA[%s]]></MusicUrl><HQMusicUrl><![CDATA[%s]]></HQMusicUrl><ThumbMediaId><![CDATA[%s]]></ThumbMediaId></Music></xml>"
	replyNews               = "<xml>%s<MsgType><![CDATA[news]]></MsgType><ArticleCount>%d</ArticleCount><Articles>%s</Articles></xml>"
	replyHeader             = "<ToUserName><![CDATA[%s]]></ToUserName><FromUserName><![CDATA[%s]]></FromUserName><CreateTime>%d</CreateTime>"
	replyArticle            = "<item><Title><![CDATA[%s]]></Title> <Description><![CDATA[%s]]></Description><PicUrl><![CDATA[%s]]></PicUrl><Url><![CDATA[%s]]></Url></item>"
	transferCustomerService = "<xml>" + replyHeader + "<MsgType><![CDATA[transfer_customer_service]]></MsgType></xml>"

	// Material request
	requestMaterial = `{"type":"%s","offset":%d,"count":%d}`
	// QR scene request
	requestQRScene         = `{"expire_seconds":%d,"action_name":"QR_SCENE","action_info":{"scene":{"scene_id":%d}}}`
	requestQRSceneStr      = `{"expire_seconds":%d,"action_name":"QR_STR_SCENE","action_info":{"scene":{"scene_str":"%s"}}}`
	requestQRLimitScene    = `{"action_name":"QR_LIMIT_SCENE","action_info":{"scene":{"scene_id":%d}}}`
	requestQRLimitSceneStr = `{"action_name":"QR_LIMIT_STR_SCENE","action_info":{"scene":{"scene_str":"%s"}}}`
)

// MessageHeader is the header of common message.
type MessageHeader struct {
	ToUserName   string
	FromUserName string
	CreateTime   int
	MsgType      string
	Encrypt      string
}

// Request is weixin event request.
type Request struct {
	MessageHeader
	MsgId        int64 // nolint
	Content      string
	PicUrl       string // nolint
	MediaId      string // nolint
	Format       string
	ThumbMediaId string  // nolint
	LocationX    float32 `xml:"Location_X"`
	LocationY    float32 `xml:"Location_Y"`
	Scale        float32
	Label        string
	Title        string
	Description  string
	Url          string // nolint
	Event        string
	EventKey     string
	Ticket       string
	Latitude     float32
	Longitude    float32
	Precision    float32
	Recognition  string
	Status       string
}

// Music is the response of music message.
type Music struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	MusicUrl     string `json:"musicurl"`       // nolint
	HQMusicUrl   string `json:"hqmusicurl"`     // nolint
	ThumbMediaId string `json:"thumb_media_id"` // nolint
}

// Article is the response of news message.
type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PicUrl      string `json:"picurl"` // nolint
	Url         string `json:"url"`    // nolint
}

// QRScene is the QR code.
type QRScene struct {
	Ticket        string `json:"ticket"`
	ExpireSeconds int    `json:"expire_seconds"`
	Url           string `json:"url,omitempty"` // nolint
}

// Menu is custom menu.
type Menu struct {
	Buttons []MenuButton `json:"button,omitempty"`
}

// MenuButton is the button of custom menu.
type MenuButton struct {
	Name       string       `json:"name"`
	Type       string       `json:"type,omitempty"`
	Key        string       `json:"key,omitempty"`
	Url        string       `json:"url,omitempty"`      // nolint
	MediaId    string       `json:"media_id,omitempty"` // nolint
	SubButtons []MenuButton `json:"sub_button,omitempty"`
	AppId      string       `json:"appid,omitempty"` // nolint
	PagePath   string       `json:"pagepath,omitempty"`
}

// UserAccessToken access token for user.
type UserAccessToken struct {
	AccessToken   string `json:"access_token"`
	RefreshToken  string `json:"refresh_token"`
	ExpireSeconds int    `json:"expires_in"`
	OpenId        string `json:"openid"` // nolint
	Scope         string `json:"scope"`
	UnionId       string `json:"unionid,omitempty"` // nolint
}

// UserInfo store user information.
type UserInfo struct {
	Subscribe     int    `json:"subscribe,omitempty"`
	Language      string `json:"language,omitempty"`
	OpenId        string `json:"openid,omitempty"`  // nolint
	UnionId       string `json:"unionid,omitempty"` // nolint
	Nickname      string `json:"nickname,omitempty"`
	Sex           int    `json:"sex,omitempty"`
	City          string `json:"city,omitempty"`
	Country       string `json:"country,omitempty"`
	Province      string `json:"province,omitempty"`
	HeadImageUrl  string `json:"headimgurl,omitempty"` // nolint
	SubscribeTime int64  `json:"subscribe_time,omitempty"`
	Remark        string `json:"remark,omitempty"`
	GroupId       int    `json:"groupid,omitempty"` // nolint
}

// Material data.
type Material struct {
	MediaId    string `json:"media_id,omitempty"` // nolint
	Name       string `json:"name,omitempty"`
	UpdateTime int64  `json:"update_time,omitempty"`
	CreateTime int64  `json:"create_time,omitempty"`
	Url        string `json:"url,omitempty"` // nolint
	Content    struct {
		NewsItem []struct {
			Title            string `json:"title,omitempty"`
			ThumbMediaId     string `json:"thumb_media_id,omitempty"` // nolint
			ShowCoverPic     int    `json:"show_cover_pic,omitempty"`
			Author           string `json:"author,omitempty"`
			Digest           string `json:"digest,omitempty"`
			Content          string `json:"content,omitempty"`
			Url              string `json:"url,omitempty"`                // nolint
			ContentSourceUrl string `json:"content_source_url,omitempty"` // nolint
		} `json:"news_item,omitempty"`
	} `json:"content,omitempty"`
}

// Materials is the list of material
type Materials struct {
	TotalCount int        `json:"total_count,omitempty"`
	ItemCount  int        `json:"item_count,omitempty"`
	Items      []Material `json:"item,omitempty"`
}

// TmplData for mini program
type TmplData map[string]TmplItem

// TmplItem for mini program
type TmplItem struct {
	Value string `json:"value,omitempty"`
	Color string `json:"color,omitempty"`
}

// TmplMiniProgram for mini program
type TmplMiniProgram struct {
	AppId    string `json:"appid,omitempty"` // nolint
	PagePath string `json:"pagepath,omitempty"`
}

// TmplMsg for mini program
type TmplMsg struct {
	ToUser      string           `json:"touser"`
	TemplateId  string           `json:"template_id"`           // nolint
	Url         string           `json:"url,omitempty"`         // nolint 若填写跳转小程序 则此为版本过低的替代跳转url
	MiniProgram *TmplMiniProgram `json:"miniprogram,omitempty"` // 跳转小程序 选填
	Data        TmplData         `json:"data,omitempty"`
	Color       string           `json:"color,omitempty"` // 全局颜色
}

// ResponseWriter is used to output reply
// nolint
type ResponseWriter interface {
	// Get weixin
	GetWeixin() *Weixin
	GetUserData() interface{}
	// Reply message
	replyMsg(msg string)
	ReplyOK()
	ReplyText(text string)
	ReplyImage(mediaId string)
	ReplyVoice(mediaId string)
	ReplyVideo(mediaId string, title string, description string)
	ReplyMusic(music *Music)
	ReplyNews(articles []Article)
	TransferCustomerService(serviceId string)
	// Post message
	PostText(text string) error
	PostImage(mediaId string) error
	PostVoice(mediaId string) error
	PostVideo(mediaId string, title string, description string) error
	PostMusic(music *Music) error
	PostNews(articles []Article) error
	PostTemplateMessage(templateid string, url string, data TmplData) (int32, error)
	// Media operator
	UploadMediaFromFile(mediaType string, filepath string) (string, error)
	DownloadMediaToFile(mediaId string, filepath string) error
	UploadMedia(mediaType string, filename string, reader io.Reader) (string, error)
	DownloadMedia(mediaId string, writer io.Writer) error
}

type responseWriter struct {
	wx           *Weixin
	writer       http.ResponseWriter
	toUserName   string
	fromUserName string
}

type response struct {
	ErrorCode    int    `json:"errcode,omitempty"`
	ErrorMessage string `json:"errmsg,omitempty"`
}

// HandlerFunc is callback function handler
type HandlerFunc func(ResponseWriter, *Request)

type route struct {
	regex   *regexp.Regexp
	handler HandlerFunc
}

// AccessToken define weixin access token.
type AccessToken struct {
	Token   string
	Expires time.Time
}

type jsAPITicket struct {
	ticket  string
	expires time.Time
}

// Weixin instance
type Weixin struct {
	token          string
	routes         []*route
	tokenChan      chan AccessToken
	ticketChan     chan jsAPITicket
	userData       interface{}
	appID          string
	appSecret      string
	refreshToken   int32
	encodingAESKey []byte
}

// ToURL convert qr scene to url.
func (qr *QRScene) ToURL() string {
	return (weixinShowQRScene + "?ticket=" + qr.Ticket)
}

// New create a Weixin instance.
func New(token string, appid string, secret string) *Weixin {
	wx := &Weixin{}
	wx.token = token
	wx.appID = appid
	wx.appSecret = secret
	wx.refreshToken = 0
	wx.encodingAESKey = []byte{}
	if len(appid) > 0 && len(secret) > 0 {
		wx.tokenChan = make(chan AccessToken)
		go wx.createAccessToken(wx.tokenChan, appid, secret)
		wx.ticketChan = make(chan jsAPITicket)
		go createJsAPITicket(wx.tokenChan, wx.ticketChan)
	}
	return wx
}

// NewWithUserData create data with userdata.
func NewWithUserData(token string, appid string, secret string, userData interface{}) *Weixin {
	wx := New(token, appid, secret)
	wx.userData = userData
	return wx
}

// SetEncodingAESKey set AES key
func (wx *Weixin) SetEncodingAESKey(key string) error {
	k, err := base64.StdEncoding.DecodeString(key + "=")
	if err != nil {
		return err
	}
	wx.encodingAESKey = k
	return nil
}

// GetAppId retrun app id.
func (wx *Weixin) GetAppId() string { // nolint
	return wx.appID
}

// GetAppSecret return app secret.
func (wx *Weixin) GetAppSecret() string {
	return wx.appSecret
}

// RefreshAccessToken update access token.
func (wx *Weixin) RefreshAccessToken() {
	atomic.StoreInt32(&wx.refreshToken, 1)
	<-wx.tokenChan
}

// GetAccessToken read access token.
func (wx *Weixin) GetAccessToken() AccessToken {
	for i := 0; i < retryMaxN; i++ {
		token := <-wx.tokenChan
		if time.Since(token.Expires).Seconds() < 0 {
			return token
		}
	}
	return AccessToken{}
}

// HandleFunc used to register request callback.
func (wx *Weixin) HandleFunc(pattern string, handler HandlerFunc) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
	}
	route := &route{regex, handler}
	wx.routes = append(wx.routes, route)
}

// PostText used to post text message.
func (wx *Weixin) PostText(touser string, text string) error {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	}
	msg.ToUser = touser
	msg.MsgType = "text"
	msg.Text.Content = text
	return postMessage(wx.tokenChan, &msg)
}

// PostImage used to post image message.
func (wx *Weixin) PostImage(touser string, mediaID string) error {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Image   struct {
			MediaID string `json:"media_id"`
		} `json:"image"`
	}
	msg.ToUser = touser
	msg.MsgType = "image"
	msg.Image.MediaID = mediaID
	return postMessage(wx.tokenChan, &msg)
}

// PostVoice used to post voice message.
func (wx *Weixin) PostVoice(touser string, mediaID string) error {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Voice   struct {
			MediaID string `json:"media_id"`
		} `json:"voice"`
	}
	msg.ToUser = touser
	msg.MsgType = "voice"
	msg.Voice.MediaID = mediaID
	return postMessage(wx.tokenChan, &msg)
}

// PostVideo used to post video message.
func (wx *Weixin) PostVideo(touser string, m string, t string, d string) error {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Video   struct {
			MediaID     string `json:"media_id"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"video"`
	}
	msg.ToUser = touser
	msg.MsgType = "video"
	msg.Video.MediaID = m
	msg.Video.Title = t
	msg.Video.Description = d
	return postMessage(wx.tokenChan, &msg)
}

// PostMusic used to post music message.
func (wx *Weixin) PostMusic(touser string, music *Music) error {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Music   *Music `json:"music"`
	}
	msg.ToUser = touser
	msg.MsgType = "video"
	msg.Music = music
	return postMessage(wx.tokenChan, &msg)
}

// PostNews used to post news message.
func (wx *Weixin) PostNews(touser string, articles []Article) error {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		News    struct {
			Articles []Article `json:"articles"`
		} `json:"news"`
	}
	msg.ToUser = touser
	msg.MsgType = "news"
	msg.News.Articles = articles
	return postMessage(wx.tokenChan, &msg)
}

// UploadMediaFromFile used to upload media from local file.
func (wx *Weixin) UploadMediaFromFile(mediaType string, fp string) (string, error) {
	file, err := os.Open(fp)
	if err != nil {
		return "", err
	}
	defer file.Close()
	return wx.UploadMedia(mediaType, filepath.Base(fp), file)
}

// DownloadMediaToFile used to download media and save to local file.
func (wx *Weixin) DownloadMediaToFile(mediaID string, fp string) error {
	file, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer file.Close()
	return wx.DownloadMedia(mediaID, file)
}

// UploadMedia used to upload media with media.
func (wx *Weixin) UploadMedia(mediaType string, filename string, reader io.Reader) (string, error) {
	return uploadMedia(wx.tokenChan, mediaType, filename, reader)
}

// DownloadMedia used to download media with media.
func (wx *Weixin) DownloadMedia(mediaID string, writer io.Writer) error {
	return downloadMedia(wx.tokenChan, mediaID, writer)
}

// BatchGetMaterial used to batch get Material.
func (wx *Weixin) BatchGetMaterial(materialType string, offset int, count int) (*Materials, error) {
	reply, err := postRequest(weixinMaterialURL+"/batchget_material?access_token=", wx.tokenChan,
		[]byte(fmt.Sprintf(requestMaterial, materialType, offset, count)))
	if err != nil {
		return nil, err
	}
	var materials Materials
	if err := json.Unmarshal(reply, &materials); err != nil {
		return nil, err
	}
	return &materials, nil
}

// GetIpList used to get ip list.
func (wx *Weixin) GetIpList() ([]string, error) { // nolint
	reply, err := sendGetRequest(weixinHost+"/getcallbackip?access_token=", wx.tokenChan)
	if err != nil {
		return nil, err
	}
	var result struct {
		IPList []string `json:"ip_list"`
	}
	if err := json.Unmarshal(reply, &result); err != nil {
		return nil, err
	}
	return result.IPList, nil
}

// CreateQRScene used to create QR scene.
func (wx *Weixin) CreateQRScene(sceneID int, expires int) (*QRScene, error) {
	reply, err := postRequest(weixinQRScene+"/create?access_token=", wx.tokenChan, []byte(fmt.Sprintf(requestQRScene, expires, sceneID)))
	if err != nil {
		return nil, err
	}
	var qr QRScene
	if err := json.Unmarshal(reply, &qr); err != nil {
		return nil, err
	}
	return &qr, nil
}

// CreateQRSceneByString used to create QR scene by str.
func (wx *Weixin) CreateQRSceneByString(sceneStr string, expires int) (*QRScene, error) {
	reply, err := postRequest(weixinQRScene+"/create?access_token=", wx.tokenChan, []byte(fmt.Sprintf(requestQRSceneStr, expires, sceneStr)))
	if err != nil {
		return nil, err
	}
	var qr QRScene
	if err := json.Unmarshal(reply, &qr); err != nil {
		return nil, err
	}
	return &qr, nil
}

// CreateQRLimitScene used to create QR limit scene.
func (wx *Weixin) CreateQRLimitScene(sceneID int) (*QRScene, error) {
	reply, err := postRequest(weixinQRScene+"/create?access_token=", wx.tokenChan, []byte(fmt.Sprintf(requestQRLimitScene, sceneID)))
	if err != nil {
		return nil, err
	}
	var qr QRScene
	if err := json.Unmarshal(reply, &qr); err != nil {
		return nil, err
	}
	return &qr, nil
}

// CreateQRLimitSceneByString used to create QR limit scene by str.
func (wx *Weixin) CreateQRLimitSceneByString(sceneStr string) (*QRScene, error) {
	reply, err := postRequest(weixinQRScene+"/create?access_token=", wx.tokenChan, []byte(fmt.Sprintf(requestQRLimitSceneStr, sceneStr)))
	if err != nil {
		return nil, err
	}
	var qr QRScene
	if err := json.Unmarshal(reply, &qr); err != nil {
		return nil, err
	}
	return &qr, nil
}

// ShortURL used to convert long url to short url
func (wx *Weixin) ShortURL(url string) (string, error) {
	var request struct {
		Action  string `json:"action"`
		LongURL string `json:"long_url"`
	}
	request.Action = "long2short"
	request.LongURL = url
	data, err := marshal(request)
	if err != nil {
		return "", err
	}
	reply, err := postRequest(weixinShortURL+"?access_token=", wx.tokenChan, data)
	if err != nil {
		return "", err
	}
	var shortURL struct {
		URL string `json:"short_url"`
	}
	if err := json.Unmarshal(reply, &shortURL); err != nil {
		return "", err
	}
	return shortURL.URL, nil
}

// CreateMenu used to create custom menu.
func (wx *Weixin) CreateMenu(menu *Menu) error {
	data, err := marshal(menu)
	if err != nil {
		return err
	}
	_, err = postRequest(weixinHost+"/menu/create?access_token=", wx.tokenChan, data)
	return err
}

// GetMenu used to get menu.
func (wx *Weixin) GetMenu() (*Menu, error) {
	reply, err := sendGetRequest(weixinHost+"/menu/get?access_token=", wx.tokenChan)
	if err != nil {
		return nil, err
	}
	var result struct {
		MenuCtx *Menu `json:"menu"`
	}
	if err := json.Unmarshal(reply, &result); err != nil {
		return nil, err
	}
	return result.MenuCtx, nil
}

// DeleteMenu used to delete menu.
func (wx *Weixin) DeleteMenu() error {
	_, err := sendGetRequest(weixinHost+"/menu/delete?access_token=", wx.tokenChan)
	return err
}

// SetTemplateIndustry used to set template industry.
func (wx *Weixin) SetTemplateIndustry(id1 string, id2 string) error {
	var industry struct {
		ID1 string `json:"industry_id1,omitempty"`
		ID2 string `json:"industry_id2,omitempty"`
	}
	industry.ID1 = id1
	industry.ID2 = id2
	data, err := marshal(industry)
	if err != nil {
		return err
	}
	_, err = postRequest(weixinTemplate+"/api_set_industry?access_token=", wx.tokenChan, data)
	return err
}

// AddTemplate used to add template.
func (wx *Weixin) AddTemplate(shortid string) (string, error) {
	var request struct {
		Shortid string `json:"template_id_short,omitempty"`
	}
	request.Shortid = shortid
	data, err := marshal(request)
	if err != nil {
		return "", err
	}
	reply, err := postRequest(weixinTemplate+"/api_set_industry?access_token=", wx.tokenChan, data)
	if err != nil {
		return "", err
	}
	var templateID struct {
		ID string `json:"template_id,omitempty"`
	}
	if err := json.Unmarshal(reply, &templateID); err != nil {
		return "", err
	}
	return templateID.ID, nil
}

// PostTemplateMessage used to post template message.
func (wx *Weixin) PostTemplateMessage(touser string, templateid string, url string, data TmplData) (int32, error) {
	var msg struct {
		ToUser     string   `json:"touser"`
		TemplateID string   `json:"template_id"`
		URL        string   `json:"url,omitempty"`
		Data       TmplData `json:"data,omitempty"`
	}
	msg.ToUser = touser
	msg.TemplateID = templateid
	msg.URL = url
	msg.Data = data
	msgStr, err := marshal(msg)
	if err != nil {
		return 0, err
	}
	reply, err := postRequest(weixinHost+"/message/template/send?access_token=", wx.tokenChan, msgStr)
	if err != nil {
		return 0, err
	}
	var resp struct {
		MsgID int32 `json:"msgid,omitempty"`
	}
	if err := json.Unmarshal(reply, &resp); err != nil {
		return 0, err
	}
	return resp.MsgID, nil
}

// PostTemplateMessageMiniProgram 兼容模板消息跳转小程序
func (wx *Weixin) PostTemplateMessageMiniProgram(msg *TmplMsg) (int64, error) {
	msgStr, err := marshal(msg)
	if err != nil {
		return 0, err
	}
	reply, err := postRequest(weixinHost+"/message/template/send?access_token=", wx.tokenChan, msgStr)
	if err != nil {
		return 0, err
	}
	var resp struct {
		MsgID int64 `json:"msgid,omitempty"`
	}
	if err := json.Unmarshal(reply, &resp); err != nil {
		return 0, err
	}
	return resp.MsgID, nil
}

// CreateRedirectURL used to create redirect url
func (wx *Weixin) CreateRedirectURL(urlStr string, scope string, state string) string {
	return fmt.Sprintf(weixinRedirectURL, wx.appID, url.QueryEscape(urlStr), scope, state)
}

// GetUserAccessToken used to get open id
func (wx *Weixin) GetUserAccessToken(code string) (*UserAccessToken, error) {
	resp, err := http.Get(fmt.Sprintf(weixinUserAccessTokenURL, wx.appID, wx.appSecret, code))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var res UserAccessToken
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// GetUserInfo used to get user info
func (wx *Weixin) GetUserInfo(openid string) (*UserInfo, error) {
	reply, err := sendGetRequest(fmt.Sprintf("%s?openid=%s&lang=zh_CN&access_token=", weixinUserInfo, openid), wx.tokenChan)
	if err != nil {
		return nil, err
	}
	var result UserInfo
	if err := json.Unmarshal(reply, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetJsAPITicket used to get js api ticket.
func (wx *Weixin) GetJsAPITicket() (string, error) {
	for i := 0; i < retryMaxN; i++ {
		ticket := <-wx.ticketChan
		if time.Since(ticket.expires).Seconds() < 0 {
			return ticket.ticket, nil
		}
	}
	return "", errors.New("Get JsApi Ticket Timeout")
}

// JsSignature used to sign js url.
func (wx *Weixin) JsSignature(url string, timestamp int64, noncestr string) (string, error) {
	ticket, err := wx.GetJsAPITicket()
	if err != nil {
		return "", err
	}
	h := sha1.New()
	h.Write([]byte(fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", // nolint
		ticket, noncestr, timestamp, url)))
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// CreateHandlerFunc used to create handler function.
func (wx *Weixin) CreateHandlerFunc(w http.ResponseWriter, r *http.Request) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wx.ServeHTTP(w, r)
	}
}

// ServeHTTP used to process weixin request and send response.
func (wx *Weixin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !checkSignature(wx.token, w, r) {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}
	// Verify request
	if r.Method == "GET" {
		fmt.Fprintf(w, r.FormValue("echostr")) // nolint
		return
	}
	// Process message
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Weixin receive message failed:", err)
		http.Error(w, "", http.StatusBadRequest)
	} else {
		var msg Request
		if err := xml.Unmarshal(data, &msg); err != nil {
			log.Println("Weixin parse message failed:", err)
			http.Error(w, "", http.StatusBadRequest)
			return
		}
		if len(wx.encodingAESKey) > 0 && len(msg.Encrypt) > 0 {
			// check encrypt
			d, err := base64.StdEncoding.DecodeString(msg.Encrypt)
			if err != nil {
				log.Println("Weixin decode base64 message failed:", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			if len(d) <= 20 {
				log.Println("Weixin invalid aes message:", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			// valid
			strs := sort.StringSlice{wx.token, r.FormValue("timestamp"), r.FormValue("nonce"), msg.Encrypt}
			sort.Strings(strs)
			if fmt.Sprintf("%x", sha1.Sum([]byte(strings.Join(strs, "")))) != r.FormValue("msg_signature") {
				log.Println("Weixin check message sign failed!")
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			// decode
			key := wx.encodingAESKey
			b, err := aes.NewCipher(key)
			if err != nil {
				log.Println("Weixin create cipher failed:", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
			bs := b.BlockSize()
			bm := cipher.NewCBCDecrypter(b, key[:bs])
			data = make([]byte, len(d))
			bm.CryptBlocks(data, d)
			data = fixPKCS7UnPadding(data)
			len := binary.BigEndian.Uint32(data[16:20])
			if err := xml.Unmarshal(data[20:(20+len)], &msg); err != nil {
				log.Println("Weixin parse aes message failed:", err)
				http.Error(w, "", http.StatusBadRequest)
				return
			}
		}
		wx.routeRequest(w, &msg)
	}
	return
}

func (wx *Weixin) routeRequest(w http.ResponseWriter, r *Request) {
	requestPath := r.MsgType
	if requestPath == msgEvent {
		requestPath += "." + r.Event
	}
	for _, route := range wx.routes {
		if !route.regex.MatchString(requestPath) {
			continue
		}
		writer := responseWriter{}
		writer.wx = wx
		writer.writer = w
		writer.toUserName = r.FromUserName
		writer.fromUserName = r.ToUserName
		route.handler(writer, r)
		return
	}
	http.Error(w, "", http.StatusNotFound)
	return
}

func marshal(v interface{}) ([]byte, error) {
	data, err := json.Marshal(v)
	if err == nil {
		data = bytes.Replace(data, []byte("\\u003c"), []byte("<"), -1)
		data = bytes.Replace(data, []byte("\\u003e"), []byte(">"), -1)
		data = bytes.Replace(data, []byte("\\u0026"), []byte("&"), -1)
	}
	return data, err
}

func fixPKCS7UnPadding(data []byte) []byte {
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}

func checkSignature(t string, w http.ResponseWriter, r *http.Request) bool {
	r.ParseForm() // nolint
	signature := r.FormValue("signature")
	timestamp := r.FormValue("timestamp")
	nonce := r.FormValue("nonce")
	strs := sort.StringSlice{t, timestamp, nonce}
	sort.Strings(strs)
	var str string
	for _, s := range strs {
		str += s
	}
	h := sha1.New()
	h.Write([]byte(str)) // nolint
	return fmt.Sprintf("%x", h.Sum(nil)) == signature
}

func authAccessToken(appid string, secret string) (string, time.Duration) {
	resp, err := http.Get(weixinHost + "/token?grant_type=client_credential&appid=" + appid + "&secret=" + secret)
	if err != nil {
		log.Println("Get access token failed: ", err)
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("Read access token failed: ", err)
		} else {
			var res struct {
				AccessToken string `json:"access_token"`
				ExpiresIn   int64  `json:"expires_in"`
			}
			if err := json.Unmarshal(body, &res); err != nil {
				log.Println("Parse access token failed: ", err)
			} else {
				//log.Printf("AuthAccessToken token=%s expires_in=%d", res.AccessToken, res.ExpiresIn)
				return res.AccessToken, time.Duration(res.ExpiresIn * 1000 * 1000 * 1000)
			}
		}
	}
	return "", 0
}

func getJsAPITicket(c chan AccessToken) (*jsAPITicket, error) {
	reply, err := sendGetRequest(weixinJsApiTicketURL+"?type=jsapi&access_token=", c)
	if err != nil {
		return nil, err
	}
	var res struct {
		Ticket    string `json:"ticket"`
		ExpiresIn int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(reply, &res); err != nil {
		return nil, err
	}
	var ticket jsAPITicket
	ticket.ticket = res.Ticket
	ticket.expires = time.Now().Add(time.Duration(res.ExpiresIn * 1000 * 1000 * 1000))
	return &ticket, nil

}

func (wx *Weixin) createAccessToken(c chan AccessToken, appid string, secret string) {
	token := AccessToken{"", time.Now()}
	c <- token
	for {
		swapped := atomic.CompareAndSwapInt32(&wx.refreshToken, 1, 0)
		if swapped || time.Since(token.Expires).Seconds() >= 0 {
			var expires time.Duration
			token.Token, expires = authAccessToken(appid, secret)
			token.Expires = time.Now().Add(expires)
		}
		c <- token
	}
}

func createJsAPITicket(cin chan AccessToken, c chan jsAPITicket) {
	ticket := jsAPITicket{"", time.Now()}
	c <- ticket
	for {
		if time.Since(ticket.expires).Seconds() >= 0 {
			t, err := getJsAPITicket(cin)
			if err == nil {
				ticket = *t
			}
		}
		c <- ticket
	}
}

func sendGetRequest(reqURL string, c chan AccessToken) ([]byte, error) {
	for i := 0; i < retryMaxN; i++ {
		token := <-c
		if time.Since(token.Expires).Seconds() < 0 {
			r, err := http.Get(reqURL + token.Token)
			if err != nil {
				return nil, err
			}
			defer r.Body.Close()
			reply, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			var result response
			if err := json.Unmarshal(reply, &result); err != nil {
				return nil, err
			}
			switch result.ErrorCode {
			case 0:
				return reply, nil
			case 42001: // access_token timeout and retry
				continue
			default:
				return nil, fmt.Errorf("WeiXin send get request reply[%d]: %s", result.ErrorCode, result.ErrorMessage)
			}
		}
	}
	return nil, errors.New("WeiXin post request too many times:" + reqURL)
}

func postRequest(reqURL string, c chan AccessToken, data []byte) ([]byte, error) {
	for i := 0; i < retryMaxN; i++ {
		token := <-c
		if time.Since(token.Expires).Seconds() < 0 {
			r, err := http.Post(reqURL+token.Token, "application/json; charset=utf-8", bytes.NewReader(data))
			if err != nil {
				return nil, err
			}
			defer r.Body.Close()
			reply, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			var result response
			if err := json.Unmarshal(reply, &result); err != nil {
				return nil, err
			}
			switch result.ErrorCode {
			case 0:
				return reply, nil
			case 42001: // access_token timeout and retry
				continue
			default:
				return nil, fmt.Errorf("WeiXin send post request reply[%d]: %s", result.ErrorCode, result.ErrorMessage)
			}
		}
	}
	return nil, errors.New("WeiXin post request too many times:" + reqURL)
}

func postMessage(c chan AccessToken, msg interface{}) error {
	data, err := marshal(msg)
	if err != nil {
		return err
	}
	_, err = postRequest(weixinHost+"/message/custom/send?access_token=", c, data)
	return err
}

// nolint: gocyclo
func uploadMedia(c chan AccessToken, mediaType string, filename string, reader io.Reader) (string, error) {
	reqURL := weixinFileURL + "/upload?type=" + mediaType + "&access_token="
	for i := 0; i < retryMaxN; i++ {
		token := <-c
		if time.Since(token.Expires).Seconds() < 0 {
			bodyBuf := &bytes.Buffer{}
			bodyWriter := multipart.NewWriter(bodyBuf)
			fileWriter, err := bodyWriter.CreateFormFile("filename", filename)
			if err != nil {
				return "", err
			}
			if _, err = io.Copy(fileWriter, reader); err != nil {
				return "", err
			}
			contentType := bodyWriter.FormDataContentType()
			bodyWriter.Close() // nolint
			r, err := http.Post(reqURL+token.Token, contentType, bodyBuf)
			if err != nil {
				return "", err
			}
			defer r.Body.Close()
			reply, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return "", err
			}
			var result struct {
				response
				Type      string `json:"type"`
				MediaID   string `json:"media_id"`
				CreatedAt int64  `json:"created_at"`
			}
			err = json.Unmarshal(reply, &result)
			if err != nil {
				return "", err
			}
			switch result.ErrorCode {
			case 0:
				return result.MediaID, nil
			case 42001: // access_token timeout and retry
				continue
			default:
				return "", fmt.Errorf("WeiXin upload[%d]: %s", result.ErrorCode, result.ErrorMessage)
			}
		}
	}
	return "", errors.New("WeiXin upload media too many times")
}

func downloadMedia(c chan AccessToken, mediaID string, writer io.Writer) error {
	reqURL := weixinFileURL + "/get?media_id=" + mediaID + "&access_token="
	for i := 0; i < retryMaxN; i++ {
		token := <-c
		if time.Since(token.Expires).Seconds() < 0 {
			r, err := http.Get(reqURL + token.Token)
			if err != nil {
				return err
			}
			defer r.Body.Close()
			if r.Header.Get("Content-Type") != "text/plain" {
				_, err = io.Copy(writer, r.Body)
				return err
			}
			reply, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return err
			}
			var result response
			if err := json.Unmarshal(reply, &result); err != nil {
				return err
			}
			switch result.ErrorCode {
			case 0:
				return nil
			case 42001: // access_token timeout and retry
				continue
			default:
				return fmt.Errorf("WeiXin download[%d]: %s", result.ErrorCode, result.ErrorMessage)
			}
		}
	}
	return errors.New("WeiXin download media too many times")
}

// Format reply message header.
func (w responseWriter) replyHeader() string {
	return fmt.Sprintf(replyHeader, w.toUserName, w.fromUserName, time.Now().Unix())
}

// Return weixin instance.
func (w responseWriter) GetWeixin() *Weixin {
	return w.wx
}

// Return user data.
func (w responseWriter) GetUserData() interface{} {
	return w.wx.userData
}

func (w responseWriter) replyMsg(msg string) {
	w.writer.Write([]byte(msg))
}

// ReplyOK used to reply empty message.
func (w responseWriter) ReplyOK() {
	w.replyMsg("success")
}

// ReplyText used to reply text message.
func (w responseWriter) ReplyText(text string) {
	w.replyMsg(fmt.Sprintf(replyText, w.replyHeader(), text))
}

// ReplyImage used to reply image message.
func (w responseWriter) ReplyImage(mediaID string) {
	w.replyMsg(fmt.Sprintf(replyImage, w.replyHeader(), mediaID))
}

// ReplyVoice used to reply voice message.
func (w responseWriter) ReplyVoice(mediaID string) {
	w.replyMsg(fmt.Sprintf(replyVoice, w.replyHeader(), mediaID))
}

// ReplyVideo used to reply video message
func (w responseWriter) ReplyVideo(mediaID string, title string, description string) {
	w.replyMsg(fmt.Sprintf(replyVideo, w.replyHeader(), mediaID, title, description))
}

// ReplyMusic used to reply music message
func (w responseWriter) ReplyMusic(m *Music) {
	msg := fmt.Sprintf(replyMusic, w.replyHeader(), m.Title, m.Description, m.MusicUrl, m.HQMusicUrl, m.ThumbMediaId)
	w.replyMsg(msg)
}

// ReplyNews used to reply news message (max 10 news)
func (w responseWriter) ReplyNews(articles []Article) {
	var ctx string
	for _, article := range articles {
		ctx += fmt.Sprintf(replyArticle, article.Title, article.Description, article.PicUrl, article.Url)
	}
	msg := fmt.Sprintf(replyNews, w.replyHeader(), len(articles), ctx)
	w.replyMsg(msg)
}

// TransferCustomerService used to tTransfer customer service
func (w responseWriter) TransferCustomerService(serviceID string) {
	msg := fmt.Sprintf(transferCustomerService, serviceID, w.fromUserName, time.Now().Unix())
	w.replyMsg(msg)
}

// PostText used to Post text message
func (w responseWriter) PostText(text string) error {
	return w.wx.PostText(w.toUserName, text)
}

// Post image message
func (w responseWriter) PostImage(mediaID string) error {
	return w.wx.PostImage(w.toUserName, mediaID)
}

// Post voice message
func (w responseWriter) PostVoice(mediaID string) error {
	return w.wx.PostVoice(w.toUserName, mediaID)
}

// Post video message
func (w responseWriter) PostVideo(mediaID string, title string, desc string) error {
	return w.wx.PostVideo(w.toUserName, mediaID, title, desc)
}

// Post music message
func (w responseWriter) PostMusic(music *Music) error {
	return w.wx.PostMusic(w.toUserName, music)
}

// Post news message
func (w responseWriter) PostNews(articles []Article) error {
	return w.wx.PostNews(w.toUserName, articles)
}

// Post template message
func (w responseWriter) PostTemplateMessage(templateid string, url string, data TmplData) (int32, error) {
	return w.wx.PostTemplateMessage(w.toUserName, templateid, url, data)
}

// Upload media from local file
func (w responseWriter) UploadMediaFromFile(mediaType string, filepath string) (string, error) {
	return w.wx.UploadMediaFromFile(mediaType, filepath)
}

// Download media and save to local file
func (w responseWriter) DownloadMediaToFile(mediaID string, filepath string) error {
	return w.wx.DownloadMediaToFile(mediaID, filepath)
}

// Upload media with reader
func (w responseWriter) UploadMedia(mediaType string, filename string, reader io.Reader) (string, error) {
	return w.wx.UploadMedia(mediaType, filename, reader)
}

// Download media with writer
func (w responseWriter) DownloadMedia(mediaID string, writer io.Writer) error {
	return w.wx.DownloadMedia(mediaID, writer)
}
