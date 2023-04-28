package msgtransfer

import (
	"context"

	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/config"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/constant"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/db/controller"
	kfk "github.com/OpenIMSDK/Open-IM-Server/pkg/common/kafka"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/log"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/common/mcontext"
	pbMsg "github.com/OpenIMSDK/Open-IM-Server/pkg/proto/msg"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/proto/sdkws"
	"github.com/OpenIMSDK/Open-IM-Server/pkg/utils"
	"github.com/Shopify/sarama"
	"google.golang.org/protobuf/proto"
)

type OnlineHistoryMongoConsumerHandler struct {
	historyConsumerGroup *kfk.MConsumerGroup
	msgDatabase          controller.MsgDatabase
}

func NewOnlineHistoryMongoConsumerHandler(database controller.MsgDatabase) *OnlineHistoryMongoConsumerHandler {
	mc := &OnlineHistoryMongoConsumerHandler{
		historyConsumerGroup: kfk.NewMConsumerGroup(&kfk.MConsumerGroupConfig{KafkaVersion: sarama.V2_0_0_0,
			OffsetsInitial: sarama.OffsetNewest, IsReturnErr: false}, []string{config.Config.Kafka.MsgToMongo.Topic},
			config.Config.Kafka.MsgToMongo.Addr, config.Config.Kafka.ConsumerGroupID.MsgToMongo),
		msgDatabase: database,
	}
	return mc
}

func (mc *OnlineHistoryMongoConsumerHandler) handleChatWs2Mongo(ctx context.Context, cMsg *sarama.ConsumerMessage, msgKey string, session sarama.ConsumerGroupSession) {
	msg := cMsg.Value
	msgFromMQ := pbMsg.MsgDataToMongoByMQ{}
	operationID := mcontext.GetOperationID(ctx)
	err := proto.Unmarshal(msg, &msgFromMQ)
	if err != nil {
		log.Error("msg_transfer Unmarshal msg err", "", "msg", string(msg), "err", err.Error())
		return
	}
	log.Info(operationID, "BatchInsertChat2DB userID: ", msgFromMQ.AggregationID, "msgFromMQ.LastSeq: ", msgFromMQ.LastSeq)
	err = mc.msgDatabase.BatchInsertChat2DB(ctx, msgFromMQ.AggregationID, msgFromMQ.Messages, msgFromMQ.LastSeq)
	if err != nil {
		log.NewError(operationID, "single data insert to mongo err", err.Error(), msgFromMQ.Messages, msgFromMQ.AggregationID, msgFromMQ.TriggerID)
	}
	err = mc.msgDatabase.DeleteMessageFromCache(ctx, msgFromMQ.AggregationID, msgFromMQ.Messages)
	if err != nil {
		log.NewError(operationID, "remove cache msg from redis err", err.Error(), msgFromMQ.Messages, msgFromMQ.AggregationID, msgFromMQ.TriggerID)
	}
	for _, v := range msgFromMQ.Messages {
		if v.MsgData.ContentType == constant.DeleteMessageNotification {
			tips := sdkws.TipsComm{}
			DeleteMessageTips := sdkws.DeleteMessageTips{}
			err := proto.Unmarshal(v.MsgData.Content, &tips)
			if err != nil {
				log.NewError(operationID, "tips unmarshal err:", err.Error(), v.String())
				continue
			}
			err = proto.Unmarshal(tips.Detail, &DeleteMessageTips)
			if err != nil {
				log.NewError(operationID, "deleteMessageTips unmarshal err:", err.Error(), v.String())
				continue
			}
			if totalUnExistSeqs, err := mc.msgDatabase.DelMsgBySeqs(ctx, DeleteMessageTips.UserID, DeleteMessageTips.Seqs); err != nil {
				log.NewError(operationID, utils.GetSelfFuncName(), "DelMsgBySeqs args: ", DeleteMessageTips.UserID, DeleteMessageTips.Seqs, "error:", err.Error(), "totalUnExistSeqs: ", totalUnExistSeqs)
			}
		}
	}
}

func (OnlineHistoryMongoConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (OnlineHistoryMongoConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }
func (mc *OnlineHistoryMongoConsumerHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error { // a instance in the consumer group
	log.NewDebug("", "online new session msg come", claim.HighWaterMarkOffset(), claim.Topic(), claim.Partition())
	for msg := range claim.Messages() {
		log.NewDebug("", "kafka get info to mongo", "msgTopic", msg.Topic, "msgPartition", msg.Partition, "msg", string(msg.Value), "key", string(msg.Key))
		if len(msg.Value) != 0 {
			ctx := mc.historyConsumerGroup.GetContextFromMsg(msg)
			mc.handleChatWs2Mongo(ctx, msg, string(msg.Key), sess)
		} else {
			log.Error("", "mongo msg get from kafka but is nil", msg.Key)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}
