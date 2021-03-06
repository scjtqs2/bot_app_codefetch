package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/scjtqs2/bot_adapter/coolq"
	"github.com/scjtqs2/bot_adapter/event"
	"github.com/scjtqs2/bot_adapter/pb/entity"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

func parseMsg(data string) {
	msg := gjson.Parse(data)
	switch msg.Get("post_type").String() {
	case "message": // 消息事件
		switch msg.Get("message_type").String() {
		case event.MESSAGE_TYPE_PRIVATE:
			var req event.MessagePrivate
			_ = json.Unmarshal([]byte(msg.Raw), &req)
			go Quma(req.RawMessage, 0, req.UserID, false)
			if os.Getenv("ZHUANMA_ENABLE") == "true" {
				go Zhuanma(req.RawMessage, 0, req.UserID, false)
			}
			go Ocr(req.RawMessage, 0, req.UserID, false)
			go RollNum(req.RawMessage, 0, req.UserID, false)
		case event.MESSAGE_TYPE_GROUP:
			var req event.MessageGroup
			_ = json.Unmarshal([]byte(msg.Raw), &req)
			go Quma(req.RawMessage, req.Sender.UserID, req.GroupID, true)
			if os.Getenv("ZHUANMA_ENABLE") == "true" {
				go Zhuanma(req.RawMessage, req.Sender.UserID, req.GroupID, true)
			}
			go Ocr(req.RawMessage, req.Sender.UserID, req.GroupID, true)
			go RollNum(req.RawMessage, req.Sender.UserID, req.GroupID, true)
		}
	case "notice": // 通知事件
		switch msg.Get("notice_type").String() {
		case event.NOTICE_TYPE_FRIEND_ADD:
			var req event.NoticeFriendAdd
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_FRIEND_RECALL:
			var req event.NoticeFriendRecall
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_GROUP_BAN:
			var req event.NoticeGroupBan
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_GROUP_DECREASE:
			var req event.NoticeGroupDecrease
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_GROUP_INCREASE:
			var req event.NoticeGroupIncrease
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_GROUP_ADMIN:
			var req event.NoticeGroupAdmin
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_GROUP_RECALL:
			var req event.NoticeGroupRecall
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_GROUP_UPLOAD:
			var req event.NoticeGroupUpload
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_POKE:
			var req event.NoticePoke
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_HONOR:
			var req event.NoticeHonor
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.NOTICE_TYPE_LUCKY_KING:
			var req event.NoticeLuckyKing
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.CUSTOM_NOTICE_TYPE_GROUP_CARD:
		case event.CUSTOM_NOTICE_TYPE_OFFLINE_FILE:
		}
	case "request": // 请求事件
		switch msg.Get("request_type").String() {
		case event.REQUEST_TYPE_FRIEND:
			var req event.RequestFriend
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.REQUEST_TYPE_GROUP:
			var req event.RequestGroup
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		}
	case "meta_event": // 元事件
		switch msg.Get("meta_event_type").String() {
		case event.META_EVENT_LIFECYCLE:
			var req event.MetaEventLifecycle
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		case event.META_EVENT_HEARTBEAT:
			var req event.MetaEventHeartbeat
			_ = json.Unmarshal([]byte(msg.Raw), &req)
		}
	}
}

// Quma 取码
func Quma(message string, atqq int64, fromqq int64, isGroup bool) {
	cacheKey := fmt.Sprintf("QUMA_STATUS_%d_%d", atqq, fromqq)
	c := cache.Get(cacheKey)
	if c != nil && !c.Expired() && c.Value() != nil {
		// 有状态，直接回复string的消息
		cache.Delete(cacheKey)
		if isGroup {
			_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
				GroupId:    fromqq,
				Message:    []byte(message),
				AutoEscape: true,
			})
			return
		}
		_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
			UserId:     fromqq,
			Message:    []byte(message),
			AutoEscape: true,
		})
		return
	}
	if strings.HasPrefix(message, "#取码") {
		cache.Set(cacheKey, 1, time.Minute)
		text := "请于一分钟内发送需要取码的信息"
		if isGroup {
			_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
				GroupId: fromqq,
				Message: []byte(fmt.Sprintf("%s%s", coolq.EnAtCode(fmt.Sprintf("%d", atqq)), text)),
			})
			return
		}
		_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
			UserId:  fromqq,
			Message: []byte(text),
		})
		return
	}
}

// Zhuanma 转码
// 容易风控，一般不开启
func Zhuanma(message string, atqq int64, fromqq int64, isGroup bool) {
	cacheKey := fmt.Sprintf("ZHUANMA_STATUS_%d_%d", atqq, fromqq)
	c := cache.Get(cacheKey)
	if c != nil && !c.Expired() && c.Value() != nil {
		// 有状态，直接回复string的消息
		cache.Delete(cacheKey)
		if isGroup {
			_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
				GroupId:    fromqq,
				Message:    []byte(decodeText(message)),
				AutoEscape: false,
			})
			return
		}
		_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
			UserId:     fromqq,
			Message:    []byte(decodeText(message)),
			AutoEscape: false,
		})
		return
	}
	// 直接一条命令转码
	patten := `^#转码\s(-.{1,10}? )?(.*)$`
	reg := regexp.MustCompile(patten)
	if reg.MatchString(message) {
		keys := reg.FindStringSubmatch(message)
		var code string
		switch strings.TrimSpace(keys[1]) {
		case "-image":
			code = coolq.EnImageCode(strings.TrimSpace(keys[2]), 0)
		case "-json":
			reg2 := regexp.MustCompile(`^-id\s+(\d{1,10})\s+(.*)$`) // #转码 -json -id 1 {json}
			str := strings.TrimSpace(keys[2])
			if !reg2.MatchString(str) {
				code = "错误的json指令 eg: #转码 -json -id 1 {json}"
			} else {
				keys2 := reg2.FindStringSubmatch(str)
				id, _ := strconv.Atoi(strings.TrimSpace(keys2[1]))
				code = coolq.EnJSONCode(decodeText(strings.TrimSpace(keys2[2])), id)
			}
		case "-xml":
			reg2 := regexp.MustCompile(`^-id\s+(\d{1,10})\s+(.*)$`) // #转码 -xml -id 1 <xml>
			str := strings.TrimSpace(keys[2])
			if !reg2.MatchString(str) {
				code = "错误的xml指令 eg: #转码 -xml -id 1 <xml>"
			} else {
				keys2 := reg2.FindStringSubmatch(str)
				id, _ := strconv.Atoi(strings.TrimSpace(keys2[1]))
				code = coolq.EnXMLCode(decodeText(strings.TrimSpace(keys2[2])), id)
			}
		}
		if code != "" {
			if isGroup {
				_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
					GroupId: fromqq,
					Message: []byte(code),
				})
				return
			}
			_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
				UserId:  fromqq,
				Message: []byte(code),
			})
		}
		return
	}
	if strings.HasPrefix(message, "#转码") {
		cache.Set(cacheKey, 1, time.Minute)
		text := "请于一分钟内发送需要转码的CQ码"
		if isGroup {
			_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
				GroupId: fromqq,
				Message: []byte(fmt.Sprintf("%s%s", coolq.EnAtCode(fmt.Sprintf("%d", atqq)), text)),
			})
			return
		}
		_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
			UserId:  fromqq,
			Message: []byte(text),
		})
		return
	}
}

// Ocr ocr识别图片
func Ocr(message string, atqq int64, fromqq int64, isGroup bool) {
	cacheKey := fmt.Sprintf("OCR_STATUS_%d_%d", atqq, fromqq)
	c := cache.Get(cacheKey)
	if c != nil && !c.Expired() && c.Value() != nil {
		// 有状态，获取图片地址
		cache.Delete(cacheKey)
		var file string
		msgs := coolq.DeCode(message) // 将字符串格式转成 array格式
		for _, v := range msgs {
			if v.Type == coolq.IMAGE {
				file = v.Data["file"]
				res, err := botAdapterClient.CustomOcrImage(context.TODO(), &entity.CustomOcrImageReq{
					Image: file,
				})
				if err != nil {
					log.Errorf("获取 Ocr 错误：%v", err)
					return
				}
				text := ""
				for _, t := range res.GetTexts() {
					text += t.GetText()
				}
				if isGroup {
					_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
						GroupId: fromqq,
						Message: []byte(fmt.Sprintf("%s%s", coolq.EnAtCode(fmt.Sprintf("%d", atqq)), text)),
					})
					return
				}
				_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
					UserId:  fromqq,
					Message: []byte(text),
				})
				return
			}
		}
	}
	if strings.HasPrefix(message, "#OCR") {
		cache.Set(cacheKey, 1, time.Minute)
		text := "请于一分钟内发送需要ocr的图片"
		if isGroup {
			_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
				GroupId: fromqq,
				Message: []byte(fmt.Sprintf("%s%s", coolq.EnAtCode(fmt.Sprintf("%d", atqq)), text)),
			})
			return
		}
		_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
			UserId:  fromqq,
			Message: []byte(text),
		})
		return
	}
}

// RollNum 随机roll点数 0-100
func RollNum(message string, atqq int64, fromqq int64, isGroup bool) {
	if strings.HasPrefix(message, "#ROLL") {
		r := rand.New(rand.NewSource(time.Now().Unix()))
		n := r.Intn(100)
		text := fmt.Sprintf("你得到了roll点数：%d", n)
		if isGroup {
			_, _ = botAdapterClient.SendGroupMsg(context.TODO(), &entity.SendGroupMsgReq{
				GroupId: fromqq,
				Message: []byte(fmt.Sprintf("%s%s", coolq.EnAtCode(fmt.Sprintf("%d", atqq)), text)),
			})
			return
		}
		_, _ = botAdapterClient.SendPrivateMsg(context.TODO(), &entity.SendPrivateMsgReq{
			UserId:  fromqq,
			Message: []byte(text),
		})
		return
	}
}

// decodeText 解码文字
func decodeText(text string) string {
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&#91;", "[")
	text = strings.ReplaceAll(text, "&#93;", "]")
	return text
}
