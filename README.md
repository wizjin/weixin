# 微信公众平台库 – Go语言版本

## 简介

这是一个使用Go语言对微信公众平台的封装。参考了微信公众平台API文档

[![GoDoc](http://godoc.org/github.com/wizjin/weixin?status.png)](http://godoc.org/github.com/wizjin/weixin)

## 入门

### 安装

通过执行下列语句就可以完成安装

	go get github.com/wizjin/weixin

### 注册微信公众平台

注册微信公众平台，填写验证微信公众平台的Token

### 示例

```Go
package main

import (
	"github.com/wizjin/weixin"
	"net/http"
)

// 文本消息的处理函数
func Echo(w weixin.ResponseWriter, r *weixin.Request) {
	txt := r.Content			// 获取用户发送的消息
	w.ReplyText(txt)			// 回复一条文本消息
	w.PostText("Post:" + txt)	// 发送一条文本消息
}

// 关注事件的处理函数
func Subscribe(w weixin.ResponseWriter, r *weixin.Request) {
	w.ReplyText("欢迎关注") // 有新人关注，返回欢迎消息
}

func main() {
	// my-token 验证微信公众平台的Token
	// app-id, app-secret用于高级API调用。
	// 如果仅使用接收/回复消息，则可以不填写，使用下面语句
	// mux := weixin.New("my-token", "", "")
	mux := weixin.New("my-token", "app-id", "app-secret")
	// 注册文本消息的处理函数
	mux.HandleFunc(weixin.MsgTypeText, Echo)
	// 注册关注事件的处理函数
	mux.HandleFunc(weixin.MsgTypeEventSubscribe, Subscribe)
	http.Handle("/", mux) // 注册接收微信服务器数据的接口URI
	http.ListenAndServe(":80", nil) // 启动接收微信数据服务器
}
```

微信公众平台要求在收到消息后5秒内回复消息（Reply接口）
如果时间操作很长，则可以使用Post接口发送消息

### 处理函数

处理函数的定义可以使用下面的形式

```Go
func Func(w weixin.ResponseWriter, r *weixin.Request) {
	...
}
```

可以注册的处理函数类型有以下几种

* `weixin.MsgTypeText`				接收文本消息	
* `weixin.MsgTypeImage`				接收图片消息	
* `weixin.MsgTypeVoice`				接收语音消息	
* `weixin.MsgTypeVideo`				接收视频消息	
* `weixin.MsgTypeLocation`			接收地理位置消息
* `weixin.MsgTypeLink`				接收链接消息
* `weixin.MsgTypeEventSubscribe`	接收关注事件
* `weixin.MsgTypeEventUnsubscribe`	接收取消关注事件
* `weixin.MsgTypeEventScan`			接收扫描二维码事件
* `weixin.MsgTypeEventClick`		接收自定义菜单事件

### 发送被动响应消息

需要发送被动响应消息，可通过`weixin.ResponseWriter`的下列方法完成

* `ReplyText(text)`							回复文本消息
* `ReplyImage(mediaId)`						回复图片消息
* `ReplyVoice(mediaId)`						回复语音消息
* `ReplyVideo(mediaId, title, description)`	回复视频消息
* `ReplyMusic(music)`						回复音乐消息
* `ReplyNews(articles)`						回复图文消息

### 发送客服消息

* `PostText(text)`							发送文本消息
* `PostImage(mediaId)`						发送图片消息
* `PostVoice(mediaId)`						发送语音消息
* `PostVideo(mediaId, title, description)`	发送视频消息
* `PostMusic(music)`						发送音乐消息
* `PostNews(articles)`						发送图文消息

### 上传/下载多媒体文件

使用如下函数可以用来上传多媒体文件:

`UploadMediaFromFile(mediaType string, filepath string)`

示例 (用一张本地图片来返回图片消息):

```Go
func ReciveMessage(w weixin.ResponseWriter, r *weixin.Request) {
	mediaId, err := w.UploadMediaFromFile(weixin.MediaTypeImage, "/my-file-path") // 上传本地文件并获取MediaID
	if err != nil {
		w.ReplyText("保存图片失败")
	} else {
		w.ReplyImage(mediaId)	// 利用获取的MediaId来返回图片消息
	}
}
```

使用如下函数可以用来下载多媒体文件:

`DownloadMediaToFile(mediaId string, filepath string)`

示例 (收到一条图片消息，然后保存图片到本地文件):

```Go
func ReciveImageMessage(w weixin.ResponseWriter, r *weixin.Request) {
	err := w.DownloadMediaToFile(r.MediaId, "/my-file-path") // 下载文件并保存到本地
	if err != nil {
		w.ReplyText("保存图片失败")
	} else {
		w.ReplyText("保存图片成功")
	}
}
```

## 参考连接

* [Wiki](https://github.com/wizjin/weixin/wiki)
* [API文档](http://godoc.org/github.com/wizjin/weixin)
* [微信公众平台](https://mp.weixin.qq.com)
* [微信公众平台API文档](http://mp.weixin.qq.com/wiki/index.php)

## 版权声明

This project is licensed under the MIT license, see [LICENSE](LICENSE).

## 更新日志

### Version 0.4 - upcoming

* 创建/换取二维码

### Version 0.3 - 2014/01/07

* 多媒体文件处理：上传/下载多媒体文件

### Version 0.2 - 2013/12/19

* 发送客服消息：文本消息，图片消息，语音消息，视频消息，音乐消息，图文消息

### Version 0.1 – 2013/12/17

* Token验证URL有效性
* 接收普通消息：文本消息，图片消息，语音消息，视频消息，地理位置消息，链接消息
* 接收事件推送：关注/取消关注，扫描二维码事件，上报地理位置，自定义菜单
* 发送被动响应消息：文本消息，图片消息，语音消息，视频消息，音乐消息，图文消息
