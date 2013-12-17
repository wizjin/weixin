// Weixin MP SDK (Golang)
package weixin

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"time"
)

const (
	// Event type
	msgEvent         = "event"
	EventSubscribe   = "subscribe"
	EventUnsubscribe = "unsubscribe"
	EventScan        = "scan"
	EventClick       = "CLICK"
	// Message type
	MsgTypeDefault          = ".*"
	MsgTypeText             = "text"
	MsgTypeImage            = "image"
	MsgTypeVoice            = "voice"
	MsgTypeVideo            = "video"
	MsgTypeLocation         = "location"
	MsgTypeLink             = "link"
	MsgTypeEvent            = msgEvent + ".*"
	MsgTypeEventSubscribe   = msgEvent + "\\." + EventSubscribe
	MsgTypeEventUnsubscribe = msgEvent + "\\." + EventUnsubscribe
	MsgTypeEventScan        = msgEvent + "\\." + EventScan
	MsgTypeEventClick       = msgEvent + "\\." + EventClick

	// Weixin host URL
	weixinHost = "https://api.weixin.qq.com/cgi-bin"

	// Reply format
	replyText    = "<xml>%s<MsgType><![CDATA[text]]></MsgType><Content><![CDATA[%s]]></Content></xml>"
	replyImage   = "<xml>%s<MsgType><![CDATA[image]]></MsgType><Image><MediaId><![CDATA[%s]]></MediaId></Image></xml>"
	replyVoice   = "<xml>%s<MsgType><![CDATA[voice]]></MsgType><Voice><MediaId><![CDATA[%s]]></MediaId></Voice></xml>"
	replyVideo   = "<xml>%s<MsgType><![CDATA[video]]></MsgType><Video><MediaId><![CDATA[%s]]></MediaId><Title><![CDATA[%s]]></Title><Description><![CDATA[%s]]></Description></Video></xml>"
	replyMusic   = "<xml>%s<MsgType><![CDATA[music]]></MsgType><Music><Title><![CDATA[%s]]></Title><Description><![CDATA[%s]]></Description><MusicUrl><![CDATA[%s]]></MusicUrl><HQMusicUrl><![CDATA[%s]]></HQMusicUrl><ThumbMediaId><![CDATA[%s]]></ThumbMediaId></Music></xml>"
	replyNews    = "<xml>%s<MsgType><![CDATA[news]]></MsgType><ArticleCount>%d</ArticleCount><Articles>%s</Articles></xml>"
	replyHeader  = "<ToUserName><![CDATA[%s]]></ToUserName><FromUserName><![CDATA[%s]]></FromUserName><CreateTime>%d</CreateTime>"
	replyArticle = "<item><Title><![CDATA[%s]]></Title> <Description><![CDATA[%s]]></Description><PicUrl><![CDATA[%s]]></PicUrl><Url><![CDATA[%s]]></Url></item>"
)

// Common message header
type MessageHeader struct {
	ToUserName   string
	FromUserName string
	CreateTime   int
	MsgType      string
}

// Weixin request
type Request struct {
	MessageHeader
	MsgId        int64
	Content      string
	PicUrl       string
	MediaId      string
	Format       string
	ThumbMediaId string
	LocationX    float32 `xml:"Location_X"`
	LocationY    float32 `xml:"Location_Y"`
	Scale        float32
	Label        string
	Title        string
	Description  string
	Url          string
	Event        string
	EventKey     string
	Ticket       string
	Latitude     float32
	Longitude    float32
	Precision    float32
	Recognition  string
}

// Use to reply music message
type Music struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	MusicUrl     string `json:"musicurl"`
	HQMusicUrl   string `json:"hqmusicurl"`
	ThumbMediaId string `json:"thumb_media_id"`
}

// Use to reply news message
type Article struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	PicUrl      string `json:"picurl"`
	Url         string `json:"url"`
}

// Use to output reply
type ResponseWriter interface {
	// Reply message
	ReplyText(text string)
	ReplyImage(mediaId string)
	ReplyVoice(mediaId string)
	ReplyVideo(mediaId string, title string, description string)
	ReplyMusic(music *Music)
	ReplyNews(articles []Article)
	// Post message
	PostText(text string)
	PostImage(mediaId string)
	PostVoice(mediaId string)
	PostVideo(mediaId string, title string, description string)
	PostMusic(music *Music)
	PostNews(articles []Article)
}

type responseWriter struct {
	wx           *Weixin
	writer       http.ResponseWriter
	toUserName   string
	fromUserName string
}

// Callback function
type HandlerFunc func(ResponseWriter, *Request)

type route struct {
	regex   *regexp.Regexp
	handler HandlerFunc
}

type Weixin struct {
	token    string
	routes   []*route
	msgQueue chan interface{}
}

// Create a Weixin instance
func New(token string, appid string, secret string) *Weixin {
	wx := &Weixin{}
	wx.token = token
	if len(appid) > 0 && len(secret) > 0 {
		wx.msgQueue = make(chan interface{}, 10)
		go postMessage(wx.msgQueue, appid, secret)
	}
	return wx
}

// Register request callback.
func (wx *Weixin) HandleFunc(pattern string, handler HandlerFunc) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
		return
	}
	route := &route{regex, handler}
	wx.routes = append(wx.routes, route)
}

// Post text message
func (wx *Weixin) PostText(touser string, text string) {
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
	wx.msgQueue <- &msg
}

// Post image message
func (wx *Weixin) PostImage(touser string, mediaId string) {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Image   struct {
			MediaId string `json:"media_id"`
		} `json:"image"`
	}
	msg.ToUser = touser
	msg.MsgType = "image"
	msg.Image.MediaId = mediaId
	wx.msgQueue <- &msg
}

// Post voice message
func (wx *Weixin) PostVoice(touser string, mediaId string) {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Voice   struct {
			MediaId string `json:"media_id"`
		} `json:"voice"`
	}
	msg.ToUser = touser
	msg.MsgType = "voice"
	msg.Voice.MediaId = mediaId
	wx.msgQueue <- &msg
}

// Post video message
func (wx *Weixin) PostVideo(touser string, m string, t string, d string) {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Video   struct {
			MediaId     string `json:"media_id"`
			Title       string `json:"title"`
			Description string `json:"description"`
		} `json:"video"`
	}
	msg.ToUser = touser
	msg.MsgType = "video"
	msg.Video.MediaId = m
	msg.Video.Title = t
	msg.Video.Description = d
	wx.msgQueue <- &msg
}

// Post music message
func (wx *Weixin) PostMusic(touser string, music *Music) {
	var msg struct {
		ToUser  string `json:"touser"`
		MsgType string `json:"msgtype"`
		Music   *Music `json:"music"`
	}
	msg.ToUser = touser
	msg.MsgType = "video"
	msg.Music = music
	wx.msgQueue <- &msg
}

func (wx *Weixin) PostNews(touser string, articles []Article) {
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
	wx.msgQueue <- &msg
}

// Process weixin request and send response.
func (wx *Weixin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !checkSignature(wx.token, w, r) {
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	// Verify request
	if r.Method == "GET" {
		fmt.Fprintf(w, r.FormValue("echostr"))
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
		} else {
			wx.routeRequest(w, &msg)
		}
	}
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
}

func checkSignature(t string, w http.ResponseWriter, r *http.Request) bool {
	r.ParseForm()
	var signature string = r.FormValue("signature")
	var timestamp string = r.FormValue("timestamp")
	var nonce string = r.FormValue("nonce")
	strs := sort.StringSlice{t, timestamp, nonce}
	sort.Strings(strs)
	var str string
	for _, s := range strs {
		str += s
	}
	h := sha1.New()
	h.Write([]byte(str))
	return fmt.Sprintf("%x", h.Sum(nil)) == signature
}

func authAccessToken(appid string, secret string) (string, float64) {
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
				AccessToken string  `json:"access_token"`
				ExpiresIn   float64 `json:"expires_in"`
			}
			if err := json.Unmarshal(body, &res); err != nil {
				log.Println("Parse access token failed: ", err)
			} else {
				return res.AccessToken, res.ExpiresIn
			}
		}
	}
	return "", 0
}

func postMessage(c chan interface{}, appid string, secret string) {
	var access_token string
	var last_auth time.Time
	var expires float64
	req_url := weixinHost + "/message/custom/send?access_token="
	for {
		msg := <-c
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Parse PostMessage failed: %s", err)
		} else {
			now := time.Now()
			if now.Sub(last_auth).Seconds() > expires {
				access_token, expires = authAccessToken(appid, secret)
				if len(access_token) > 0 {
					last_auth = now
				}
			}
			if len(access_token) > 0 {
				r := bytes.NewReader(data)
				_, err := http.Post(req_url+access_token,
					"application/json; charset=utf-8", r)
				if err != nil {
					log.Println("PostMessage failed: ", err)
				}
			}
		}
	}
}

// Format reply message header
func (w responseWriter) replyHeader() string {
	return fmt.Sprintf(replyHeader, w.toUserName, w.fromUserName, time.Now().Unix())
}

// Reply text message
func (w responseWriter) ReplyText(text string) {
	msg := fmt.Sprintf(replyText, w.replyHeader(), text)
	w.writer.Write([]byte(msg))
}

// Reply image message
func (w responseWriter) ReplyImage(mediaId string) {
	msg := fmt.Sprintf(replyImage, w.replyHeader(), mediaId)
	w.writer.Write([]byte(msg))
}

// Reply voice message
func (w responseWriter) ReplyVoice(mediaId string) {
	msg := fmt.Sprintf(replyVoice, w.replyHeader(), mediaId)
	w.writer.Write([]byte(msg))
}

// Reply video message
func (w responseWriter) ReplyVideo(mediaId string, title string, description string) {
	msg := fmt.Sprintf(replyVideo, w.replyHeader(), mediaId, title, description)
	w.writer.Write([]byte(msg))
}

// Reply music message
func (w responseWriter) ReplyMusic(m *Music) {
	msg := fmt.Sprintf(replyMusic, w.replyHeader(), m.Title, m.Description, m.MusicUrl, m.HQMusicUrl, m.ThumbMediaId)
	w.writer.Write([]byte(msg))
}

// Reply news message (max 10 news)
func (w responseWriter) ReplyNews(articles []Article) {
	var ctx string
	for _, article := range articles {
		ctx += fmt.Sprintf(replyArticle, article.Title, article.Description, article.PicUrl, article.Url)
	}
	msg := fmt.Sprintf(replyNews, w.replyHeader(), len(articles), ctx)
	w.writer.Write([]byte(msg))
}

// Post text message
func (w responseWriter) PostText(text string) {
	w.wx.PostText(w.toUserName, text)
}

// Post image message
func (w responseWriter) PostImage(mediaId string) {
	w.wx.PostImage(w.toUserName, mediaId)
}

// Post voice message
func (w responseWriter) PostVoice(mediaId string) {
	w.wx.PostVoice(w.toUserName, mediaId)
}

// Post video message
func (w responseWriter) PostVideo(mediaId string, title string, desc string) {
	w.wx.PostVideo(w.toUserName, mediaId, title, desc)
}

// Post music message
func (w responseWriter) PostMusic(music *Music) {
	w.wx.PostMusic(w.toUserName, music)
}

// Post news message
func (w responseWriter) PostNews(articles []Article) {
	w.wx.PostNews(w.toUserName, articles)
}
