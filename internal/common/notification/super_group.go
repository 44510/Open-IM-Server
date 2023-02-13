package notification

import (
	"Open_IM/pkg/common/constant"
	"Open_IM/pkg/common/log"
	//sdk "Open_IM/pkg/proto/sdkws"
	"Open_IM/pkg/utils"
	//"github.com/golang/protobuf/jsonpb"
	//"github.com/golang/protobuf/proto"
)

func (c *Check) SuperGroupNotification(operationID, sendID, recvID string) {
	n := &NotificationMsg{
		SendID:      sendID,
		RecvID:      recvID,
		MsgFrom:     constant.SysMsgType,
		ContentType: constant.SuperGroupUpdateNotification,
		SessionType: constant.SingleChatType,
		OperationID: operationID,
	}
	log.NewInfo(operationID, utils.GetSelfFuncName(), string(n.Content))
	c.Notification(n)
}
