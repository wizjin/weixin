package weixin

import (
	"crypto/sha1"
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
	// Event Type
	msgEvent         = "event"
	EventSubscribe   = "subscribe"
	EventUnsubscribe = "unsubscribe"
	EventScan        = "scan"
	EventClick       = "CLICK"
	// Message Type - Callback register
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
)

type MessageHeader struct {
	ToUserName   string
	FromUserName string
	CreateTime   int
	MsgType      string
}

// Message request
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

type ReplyArticle struct {
	Title       string
	Description string
	PicUrl      string
	Url         string
}

type ResponseWriter interface {
	WriteText(text string)
	WriteImage(mediaId string)
	WriteVoice(mediaId string)
	WriteVideo(mediaId string, title string, description string)
	WriteMusic(title string, description string, musicUrl string, hqMusicUrl string, thumbMediaId string)
	WriteNews(articles []ReplyArticle)
}

type responseWriter struct {
	writer       http.ResponseWriter
	toUserName   string
	fromUserName string
}

type HandlerFunc func(ResponseWriter, *Request)

type route struct {
	regex   *regexp.Regexp
	handler HandlerFunc
}

type Weixin struct {
	token  string
	routes []*route
}

func New(token string) *Weixin {
	return &Weixin{token: token}
}

func (wx *Weixin) HandleFunc(pattern string, handler HandlerFunc) {
	regex, err := regexp.Compile(pattern)
	if err != nil {
		panic(err)
		return
	}
	route := &route{regex, handler}
	wx.routes = append(wx.routes, route)
}

func (wx *Weixin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !checkSignature(wx.token, w, r) {
		http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
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
		http.Error(w, "400 Bad Request", http.StatusBadRequest)
	} else {
		var msg Request
		if err := xml.Unmarshal(data, &msg); err != nil {
			log.Println("Weixin parse message failed:", err)
			http.Error(w, "400 Bad Request", http.StatusBadRequest)
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
		writer.writer = w
		writer.toUserName = r.FromUserName
		writer.fromUserName = r.ToUserName
		route.handler(writer, r)
		return
	}
	http.Error(w, "404 Not Found", http.StatusNotFound)
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

const (
	replyText    = "<xml>%s<MsgType><![CDATA[text]]></MsgType><Content><![CDATA[%s]]></Content></xml>"
	replyImage   = "<xml>%s<MsgType><![CDATA[image]]></MsgType><Image><MediaId><![CDATA[%s]]></MediaId></Image></xml>"
	replyVoice   = "<xml>%s<MsgType><![CDATA[voice]]></MsgType><Voice><MediaId><![CDATA[%s]]></MediaId></Voice></xml>"
	replyVideo   = "<xml>%s<MsgType><![CDATA[video]]></MsgType><Video><MediaId><![CDATA[%s]]></MediaId><Title><![CDATA[%s]]></Title><Description><![CDATA[%s]]></Description></Video></xml>"
	replyMusic   = "<xml>%s<MsgType><![CDATA[music]]></MsgType><Music><Title><![CDATA[%s]]></Title><Description><![CDATA[%s]]></Description><MusicUrl><![CDATA[%s]]></MusicUrl><HQMusicUrl><![CDATA[%s]]></HQMusicUrl><ThumbMediaId><![CDATA[%s]]></ThumbMediaId></Music></xml>"
	replyNews    = "<xml>%s<MsgType><![CDATA[news]]></MsgType><ArticleCount>%d</ArticleCount><Articles>%s</Articles></xml>"
	replyHeader  = "<ToUserName><![CDATA[%s]]></ToUserName><FromUserName><![CDATA[%s]]></FromUserName><CreateTime>%d</CreateTime>"
	replyArticle = "<item><Title><![CDATA[%s]]></Title> <Description><![CDATA[%s]]></Description><PicUrl><![CDATA[%s]]></PicUrl><Url><![CDATA[%s]]></Url></item>"
)

func (w responseWriter) fmtHeader() string {
	return fmt.Sprintf(replyHeader, w.toUserName, w.fromUserName, time.Now().Unix())
}

func (w responseWriter) WriteText(text string) {
	msg := fmt.Sprintf(replyText, w.fmtHeader(), text)
	w.writer.Write([]byte(msg))
}

func (w responseWriter) WriteImage(mediaId string) {
	msg := fmt.Sprintf(replyImage, w.fmtHeader(), mediaId)
	w.writer.Write([]byte(msg))
}

func (w responseWriter) WriteVoice(mediaId string) {
	msg := fmt.Sprintf(replyVoice, w.fmtHeader(), mediaId)
	w.writer.Write([]byte(msg))
}

func (w responseWriter) WriteVideo(mediaId string, title string, description string) {
	msg := fmt.Sprintf(replyVideo, w.fmtHeader(), mediaId, title, description)
	w.writer.Write([]byte(msg))
}

func (w responseWriter) WriteMusic(title string, description string, musicUrl string, hqMusicUrl string, thumbMediaId string) {
	msg := fmt.Sprintf(replyMusic, w.fmtHeader(), title, description, musicUrl, hqMusicUrl, thumbMediaId)
	w.writer.Write([]byte(msg))
}

func (w responseWriter) WriteNews(articles []ReplyArticle) {
	var ctx string
	for _, article := range articles {
		ctx += fmt.Sprintf(replyArticle, article.Title, article.Description, article.PicUrl, article.Url)
	}
	msg := fmt.Sprintf(replyNews, w.fmtHeader(), len(articles), ctx)
	w.writer.Write([]byte(msg))
}
