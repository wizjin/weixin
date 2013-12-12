# 微信公众平台库——Go语言版本

## 简介

这是一个使用Go语言对微信公众平台的封装。参考微信公众平台API的文档：

<http://mp.weixin.qq.com/wiki/index.php>

### 支持功能

* Token验证URL有效性
* 接收普通消息：文本消息，图片消息，视频消息，地理位置消息，链接消息
* 接收事件推送：关注/取消关注，扫描二维码事件，上报地理位置，自定义菜单
* 发送被动响应消息：文本消息，图片消息，语音消息，视频消息，音乐消息，图文消息

### 下一版本计划

* 发送客服消息：文本消息，图片消息，语音消息，视频消息，音乐消息，图文消息

## 入门

### 安装

通过执行下列语句就可以完成安装

	go get github.com/wizjin/weixin

### 注册微信公众平台

注册微信公众平台，填写验证微信公众平台的Token

<https://mp.weixin.qq.com>

### 示例

```Go
package main

import (
	"github.com/wizjin/weixin"
	"net/http"
)

// 文本消息的处理函数
func Echo(w weixin.ResponseWriter, r *weixin.Request) {
	txt := r.Content // 获取用户发送的消息
	w.WriteText(txt) // 返回一条文本消息
}

// 关注事件的处理函数
func Subscribe(w weixin.ResponseWriter, r *weixin.Request) {
	w.WriteText("欢迎关注") // 有新人关注，返回欢迎消息
}

func main() {
	// my-token 验证微信公众平台的Token
	mux := weixin.New("my-token")
	// 注册文本消息的处理函数
	mux.HandleFunc(weixin.MsgTypeText, Echo)
	// 注册关注事件的处理函数
	mux.HandleFunc(weixin.MsgTypeEventSubscribe, Subscribe)
	http.Handle("/", mux) // 注册接收微信服务器数据的接口URI
	http.ListenAndServe(":80", nil) // 启动接收微信数据服务器
}
```

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
* `weixin.MsgTypeVoice`				接收声音消息	
* `weixin.MsgTypeVideo`				接收视频消息	
* `weixin.MsgTypeLocation`			接收地理位置消息
* `weixin.MsgTypeLink`				接收链接消息
* `weixin.MsgTypeEventSubscribe`	接收关注事件
* `weixin.MsgTypeEventUnsubscribe`	接收取消关注事件
* `weixin.MsgTypeEventScan`			接收扫描二维码事件
* `weixin.MsgTypeEventClick`		接收自定义菜单事件

## 版权声明

This project is licensed under the MIT license, see `LICENSE`.
