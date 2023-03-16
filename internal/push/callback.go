package push

import (
	"context"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/callbackstruct"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/config"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/constant"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/http"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/tracelog"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/proto/sdkws"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/utils"
)

func url() string {
	return config.Config.Callback.CallbackUrl
}

func callbackOfflinePush(ctx context.Context, userIDs []string, msg *sdkws.MsgData, offlinePushUserIDs *[]string) error {
	if !config.Config.Callback.CallbackOfflinePush.Enable {
		return nil
	}
	req := &callbackstruct.CallbackBeforePushReq{
		UserStatusBatchCallbackReq: callbackstruct.UserStatusBatchCallbackReq{
			UserStatusBaseCallback: callbackstruct.UserStatusBaseCallback{
				CallbackCommand: constant.CallbackOfflinePushCommand,
				OperationID:     tracelog.GetOperationID(ctx),
				PlatformID:      int(msg.SenderPlatformID),
				Platform:        constant.PlatformIDToName(int(msg.SenderPlatformID)),
			},
			UserIDList: userIDs,
		},
		OfflinePushInfo: msg.OfflinePushInfo,
		ClientMsgID:     msg.ClientMsgID,
		SendID:          msg.SendID,
		GroupID:         msg.GroupID,
		ContentType:     msg.ContentType,
		SessionType:     msg.SessionType,
		AtUserIDs:       msg.AtUserIDList,
		Content:         utils.GetContent(msg),
	}
	resp := &callbackstruct.CallbackBeforePushResp{}
	err := http.CallBackPostReturn(url(), req, resp, config.Config.Callback.CallbackOfflinePush)
	if err != nil {
		return err
	}
	if len(resp.UserIDs) != 0 {
		*offlinePushUserIDs = resp.UserIDs
	}
	if resp.OfflinePushInfo != nil {
		msg.OfflinePushInfo = resp.OfflinePushInfo
	}
	return nil
}

func callbackOnlinePush(ctx context.Context, userIDs []string, msg *sdkws.MsgData) error {
	if !config.Config.Callback.CallbackOnlinePush.Enable || utils.Contain(msg.SendID, userIDs...) {
		return nil
	}
	req := callbackstruct.CallbackBeforePushReq{
		UserStatusBatchCallbackReq: callbackstruct.UserStatusBatchCallbackReq{
			UserStatusBaseCallback: callbackstruct.UserStatusBaseCallback{
				CallbackCommand: constant.CallbackOnlinePushCommand,
				OperationID:     tracelog.GetOperationID(ctx),
				PlatformID:      int(msg.SenderPlatformID),
				Platform:        constant.PlatformIDToName(int(msg.SenderPlatformID)),
			},
			UserIDList: userIDs,
		},
		ClientMsgID: msg.ClientMsgID,
		SendID:      msg.SendID,
		GroupID:     msg.GroupID,
		ContentType: msg.ContentType,
		SessionType: msg.SessionType,
		AtUserIDs:   msg.AtUserIDList,
		Content:     utils.GetContent(msg),
	}
	resp := &callbackstruct.CallbackBeforePushResp{}
	return http.CallBackPostReturn(url(), req, resp, config.Config.Callback.CallbackOnlinePush)
}

func callbackBeforeSuperGroupOnlinePush(ctx context.Context, groupID string, msg *sdkws.MsgData, pushToUserIDs *[]string) error {
	if !config.Config.Callback.CallbackBeforeSuperGroupOnlinePush.Enable {
		return nil
	}
	req := callbackstruct.CallbackBeforeSuperGroupOnlinePushReq{
		UserStatusBaseCallback: callbackstruct.UserStatusBaseCallback{
			CallbackCommand: constant.CallbackSuperGroupOnlinePushCommand,
			OperationID:     tracelog.GetOperationID(ctx),
			PlatformID:      int(msg.SenderPlatformID),
			Platform:        constant.PlatformIDToName(int(msg.SenderPlatformID)),
		},
		ClientMsgID: msg.ClientMsgID,
		SendID:      msg.SendID,
		GroupID:     groupID,
		ContentType: msg.ContentType,
		SessionType: msg.SessionType,
		AtUserIDs:   msg.AtUserIDList,
		Content:     utils.GetContent(msg),
		Seq:         msg.Seq,
	}
	resp := &callbackstruct.CallbackBeforeSuperGroupOnlinePushResp{}
	if err := http.CallBackPostReturn(config.Config.Callback.CallbackUrl, req, resp, config.Config.Callback.CallbackBeforeSuperGroupOnlinePush); err != nil {
		return err
	}
	if len(resp.UserIDs) != 0 {
		*pushToUserIDs = resp.UserIDs
	}
	return nil
}
