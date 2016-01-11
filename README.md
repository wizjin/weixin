# 微信公众平台库 – Go语言版本

## 简介

这是一个使用Go语言对微信公众平台的封装。参考了微信公众平台API文档

[![Build Status](https://travis-ci.org/wizjin/weixin.png?branch=master)](https://travis-ci.org/wizjin/weixin)
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
如果只使用Post接口发送消息，则需要先调用ReplyOK来告知微信不用等待回复。

### 处理函数

处理函数的定义可以使用下面的形式

```Go
func Func(w weixin.ResponseWriter, r *weixin.Request) {
	...
}
```

可以注册的处理函数类型有以下几种

- `weixin.MsgTypeText`				接收文本消息
- `weixin.MsgTypeImage`				接收图片消息
- `weixin.MsgTypeVoice`				接收语音消息
- `weixin.MsgTypeVideo`				接收视频消息
- `weixin.MsgTypeShortVideo`		接收小视频消息
- `weixin.MsgTypeLocation`			接收地理位置消息
- `weixin.MsgTypeLink`				接收链接消息
- `weixin.MsgTypeEventSubscribe`	接收关注事件
- `weixin.MsgTypeEventUnsubscribe`	接收取消关注事件
- `weixin.MsgTypeEventScan`			接收扫描二维码事件
- `weixin.MsgTypeEventView`			接收点击菜单跳转链接时的事件
- `weixin.MsgTypeEventClick`		接收自定义菜单事件
- `weixin.MsgTypeEventLocation`		接收上报地理位置事件
- `weixin.MsgTypeEventTemplateSent` 接收模版消息发送结果

### 发送被动响应消息

需要发送被动响应消息，可通过`weixin.ResponseWriter`的下列方法完成

- `ReplyOK()`								无同步消息回复
- `ReplyText(text)`							回复文本消息
- `ReplyImage(mediaId)`						回复图片消息
- `ReplyVoice(mediaId)`						回复语音消息
- `ReplyVideo(mediaId, title, description)`	回复视频消息
- `ReplyMusic(music)`						回复音乐消息
- `ReplyNews(articles)`						回复图文消息

### 发送客服消息

- `PostText(text)`							发送文本消息
- `PostImage(mediaId)`						发送图片消息
- `PostVoice(mediaId)`						发送语音消息
- `PostVideo(mediaId, title, description)`	发送视频消息
- `PostMusic(music)`						发送音乐消息
- `PostNews(articles)`						发送图文消息

### 发送模版消息

如需要发送模版消息，需要先获取模版ID，之后再根据ID发送。

```Go
func GetTemplateId(wx *weixin.Weixin) {
	tid, err := wx.AddTemplate("TM00015")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(ips)	// 模版ID
	}
}
```

随后可以发送模版消息了。

```Go
func SendTemplateMessage(w weixin.ResponseWriter, r *weixin.Request) {
	templateId := ...
	url := ...
	msgid, err := w.PostTemplateMessage(templateId, url,
		weixin.TmplData{ "first": weixin.TmplItem{"Hello World!", "#173177"}})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(msgid)
	}
}
```

在模版消息发送成功后，还会通过类型为`MsgTypeEventTemplateSent`的消息推送获得发送结果。

- `TemplateSentStatusSuccess` 		发送成功
- `TemplateSentStatusUserBlock`		发送失败，用户拒绝接收
- `TemplateSentStatusSystemFailed`	发送失败，系统原因


### 上传/下载多媒体文件

使用如下函数可以用来上传多媒体文件:

`UploadMediaFromFile(mediaType string, filepath string)`

示例 (用一张本地图片来返回图片消息):

```Go
func ReciveMessage(w weixin.ResponseWriter, r *weixin.Request) {
	// 上传本地文件并获取MediaID
	mediaId, err := w.UploadMediaFromFile(weixin.MediaTypeImage, "/my-file-path")
	if err != nil {
		w.ReplyText("上传图片失败")
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
	// 下载文件并保存到本地
	err := w.DownloadMediaToFile(r.MediaId, "/my-file-path")
	if err != nil {
		w.ReplyText("保存图片失败")
	} else {
		w.ReplyText("保存图片成功")
	}
}
```

### 获取微信服务器IP地址

```Go
func GetIpList(wx *weixin.Weixin) {
	ips, err := wx.GetIpList()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(ips)	// Ip地址列表
	}
}
```

### 创建/换取二维码

示例，创建临时二维码

```Go
func CreateQRScene(wx *weixin.Weixin) {
	// 二维码ID - 1000
	// 过期时间 - 1800秒
	qr, err := wx.CreateQRScene(1000, 1800)
	if err != nil {
		fmt.Println(err)
	} else {
		url := qr.ToURL() // 获取二维码的URL
		fmt.Println(url)
	}
}
```

示例，创建永久二维码

```Go
func CreateQRScene(wx *weixin.Weixin) {
	// 二维码ID - 1001
	qr, err := wx.CreateQRLimitScene(1001)
	if err != nil {
		fmt.Println(err)
	} else {
		url := qr.ToURL() // 获取二维码的URL
		fmt.Println(url)
	}
}
```

### 长链接转短链接接口

```Go
func ShortURL(wx *weixin.Weixin) {
	// 长链接转短链接
	url, err := wx.ShortURL("http://mp.weixin.qq.com/wiki/10/165c9b15eddcfbd8699ac12b0bd89ae6.html")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(url)
	}
}
```

### 自定义菜单

示例，创建自定义菜单

```Go
func CreateMenu(wx *weixin.Weixin) {
	menu := &weixin.Menu{make([]weixin.MenuButton, 2)}
	menu.Buttons[0].Name = "我的菜单"
	menu.Buttons[0].Type = weixin.MenuButtonTypeUrl
	menu.Buttons[0].Url = "https://mp.weixin.qq.com"
	menu.Buttons[1].Name = "我的子菜单"
	menu.Buttons[1].SubButtons = make([]weixin.MenuButton, 1)
	menu.Buttons[1].SubButtons[0].Name = "测试"
	menu.Buttons[1].SubButtons[0].Type = weixin.MenuButtonTypeKey
	menu.Buttons[1].SubButtons[0].Key = "MyKey001"
	err := wx.CreateMenu(menu)
	if err != nil {
		fmt.Println(err)
	}
}
```

自定义菜单的类型有如下几种

- `weixin.MenuButtonTypeKey`				点击推事件
- `weixin.MenuButtonTypeUrl`				跳转URL
- `weixin.MenuButtonTypeScancodePush`		扫码推事件
- `weixin.MenuButtonTypeScancodeWaitmsg`	扫码推事件且弹出“消息接收中”提示框
- `weixin.MenuButtonTypePicSysphoto`		弹出系统拍照发图
- `weixin.MenuButtonTypePicPhotoOrAlbum`	弹出拍照或者相册发图
- `weixin.MenuButtonTypePicWeixin`			弹出微信相册发图器
- `weixin.MenuButtonTypeLocationSelect`		弹出地理位置选择器
- `weixin.MenuButtonTypeMediaId`			下发消息（除文本消息）
- `weixin.MenuButtonTypeViewLimited`		跳转图文消息URL

示例，获取自定义菜单

```Go
func DeleteMenu(wx *weixin.Weixin) {
	menu, err := wx.GetMenu()
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(menu)
	}
}
```

示例，删除自定义菜单

```Go
func DeleteMenu(wx *weixin.Weixin) {
	err := wx.DeleteMenu()
	if err != nil {
		fmt.Println(err)
	}
}
```

### JSSDK签名

示例，生成JSSDK签名
```Go
func SignJSSDK(wx *weixin.Weixin, url string) {
	timestamp := time.Now().Unix()
	noncestr := fmt.Sprintf("%d", c.randreader.Int())
	sign, err := wx.JsSignature(url, timestamp, noncestr)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(sign)
	}
}
```

### 重定向URL生成

示例，生成重定向URL
```Go
func CreateRedirect(wx *weixin.Weixin, url string) {
	redirect := wx.CreateRedirectURL(url, weixin.RedirectURLScopeBasic, "")
}
```

URL的类型有如下几种:

- `RedirectURLScopeBasic`					基本授权，仅用来获取OpenId或UnionId
- `RedirectURLScopeUserInfo`				高级授权，可以用于获取用户基本信息，需要用户同意

### 用户OpenId和UnionId获取

示例，获取用户OpenId和UnionId
```Go
func GetUserId(wx *weixin.Weixin, code string) {
	user, err := wx.GetUserAccessToken(code)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(user.OpenId)
		fmt.Println(user.UnionId)
	}
}
```

### 用户信息获取

示例，获取用户信息
```Go
func GetUserInfo(wx *weixin.Weixin, openid string) {
	user, err := wx.GetUserInfo(openid)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(user.Nickname)
		fmt.Println(user.Sex)
		fmt.Println(user.City)
		fmt.Println(user.Country)
		fmt.Println(user.Province)
		fmt.Println(user.HeadImageUrl)
		fmt.Println(user.SubscribeTime)
		fmt.Println(user.Remark)
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

### Version 0.5.4 - upcoming

- 用户管理

### Version 0.5.3 - 2016/01/05

- 添加模版消息送

### Version 0.5.2 - 2015/12/05

- 添加JSSDK签名生成
- 添加重定向URL生成
- 添加获取用户OpenId和UnionId
- 添加获取授权用户信息

### Version 0.5.1 - 2015/06/26

- 获取微信服务器IP地址
- 接收小视频消息

### Version 0.5.0 - 2015/06/25

- 自定义菜单
- 长链接转短链接

### Version 0.4.1 - 2014/09/07

- 添加将消息转发到多客服功能

### Version 0.4.0 - 2014/02/07

- 创建/换取二维码

### Version 0.3.0 - 2014/01/07

- 多媒体文件处理：上传/下载多媒体文件

### Version 0.2.0 - 2013/12/19

- 发送客服消息：文本消息，图片消息，语音消息，视频消息，音乐消息，图文消息

### Version 0.1.0 – 2013/12/17

- Token验证URL有效性
- 接收普通消息：文本消息，图片消息，语音消息，视频消息，地理位置消息，链接消息
- 接收事件推送：关注/取消关注，扫描二维码事件，上报地理位置，自定义菜单
- 发送被动响应消息：文本消息，图片消息，语音消息，视频消息，音乐消息，图文消息
