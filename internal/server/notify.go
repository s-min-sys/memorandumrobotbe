package server

import (
	"github.com/s-min-sys/notifier-share/pkg"
	"github.com/s-min-sys/notifier-share/pkg/model"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libeasygo/ptl"
)

func doNotify(logger l.Wrapper, notifyURL, text string) {
	go func() {
		logger.Info("start send")

		fnSend := func(receiverType model.ReceiverType, text string) {
			code, errMsg := pkg.SendTextMessage(notifyURL, &model.TextMessage{
				SendMessageTarget: model.SendMessageTarget{
					SenderBy: model.SenderByAll,
					BizCode:  "z",
					ToType:   receiverType,
					FindOpts: 0,
				},
				Text: text,
			})
			if code != ptl.CodeSuccess {
				logger.WithFields(l.StringField("errMsg", errMsg),
					l.IntField("receiverType", int(receiverType)), l.StringField("text", text), l.IntField("code", int(code))).
					Info("send failed")
			}
		}

		fnSend(model.ReceiverTypeAdminGroups, text)
		fnSend(model.ReceiverTypeAdminUsers, text)

		logger.Info("finish send")
	}()
}
