package server

import (
	"github.com/s-min-sys/notifier-share/pkg"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/ptl"
)

func doNotify(logger l.Wrapper, notifyURL, text string) {
	go func() {
		logger.Info("start send")

		fnSend := func(senderID pkg.SenderID, receiverType pkg.ReceiverType, text string) {
			code, errMsg := pkg.SendTextMessage(notifyURL, &pkg.TextMessage{
				SenderID:     senderID,
				ReceiverType: receiverType,
				Text:         text,
			})
			if code != ptl.CodeSuccess {
				logger.WithFields(l.StringField("errMsg", errMsg), l.StringField("senderID", string(senderID)),
					l.IntField("receiverType", int(receiverType)), l.StringField("text", text), l.IntField("code", int(code))).
					Info("send failed")
			}
		}

		//fnSend(pkg.SenderIDTelegram, pkg.ReceiverTypeAdminUsers, text)
		fnSend(pkg.SenderIDTelegram, pkg.ReceiverTypeAdminGroups, text)

		fnSend(pkg.SenderIDWeChat, pkg.ReceiverTypeAdminUsers, text)
		//fnSend(pkg.SenderIDWeChat, pkg.ReceiverTypeAdminGroups, text)

		logger.Info("finish send")
	}()
}
