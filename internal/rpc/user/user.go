package user

import (
	"Open_IM/internal/common/convert"
	"Open_IM/internal/common/rpc_server"
	chat "Open_IM/internal/rpc/msg"
	"Open_IM/pkg/common/config"
	"Open_IM/pkg/common/constant"
	"Open_IM/pkg/common/db/controller"
	"Open_IM/pkg/common/db/relation"
	relationTb "Open_IM/pkg/common/db/table/relation"
	"Open_IM/pkg/common/log"
	promePkg "Open_IM/pkg/common/prometheus"
	"Open_IM/pkg/common/tokenverify"
	"Open_IM/pkg/common/tracelog"
	sdkws "Open_IM/pkg/proto/sdkws"
	pbUser "Open_IM/pkg/proto/user"
	"Open_IM/pkg/utils"
	"context"
	grpcPrometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	"google.golang.org/grpc"
)

type userServer struct {
	rpcPort         int
	rpcRegisterName string
	*rpc_server.RpcServer
	controller.UserInterface
}

func NewUserServer(port int) *userServer {
	r, err := rpc_server.NewRpcServer(config.Config.RpcRegisterIP, port, config.Config.RpcRegisterName.OpenImUserName, config.Config.Zookeeper.ZkAddr, config.Config.Zookeeper.Schema)
	if err != nil {
		panic(err)
	}
	//mysql init
	var mysql relation.Mysql
	var model relation.UserGorm
	err = mysql.InitConn().AutoMigrateModel(&model)
	if err != nil {
		panic("db init err:" + err.Error())
	}
	if mysql.GormConn() != nil {
		model.DB = mysql.GormConn()
	} else {
		panic("db init err:" + "conn is nil")
	}
	return &userServer{RpcServer: r, UserInterface: controller.NewUserController(model.DB)}
}

func (s *userServer) Run() {
	operationID := utils.OperationIDGenerator()
	log.NewInfo(operationID, "rpc user start...")
	listener, address, err := rpc_server.GetTcpListen(config.Config.ListenIP, s.Port)
	if err != nil {
		panic(err)
	}
	log.NewInfo(operationID, "listen ok ", address)
	defer listener.Close()
	//grpc server
	var grpcOpts []grpc.ServerOption
	if config.Config.Prometheus.Enable {
		promePkg.NewGrpcRequestCounter()
		promePkg.NewGrpcRequestFailedCounter()
		promePkg.NewGrpcRequestSuccessCounter()
		grpcOpts = append(grpcOpts, []grpc.ServerOption{
			// grpc.UnaryInterceptor(promePkg.UnaryServerInterceptorProme),
			grpc.StreamInterceptor(grpcPrometheus.StreamServerInterceptor),
			grpc.UnaryInterceptor(grpcPrometheus.UnaryServerInterceptor),
		}...)
	}
	srv := grpc.NewServer(grpcOpts...)
	defer srv.GracefulStop()
	//Service registers with etcd
	pbUser.RegisterUserServer(srv, s)

	err = srv.Serve(listener)
	if err != nil {
		panic(err)
	}
	log.NewInfo(operationID, "rpc  user success")
}

// ok
func (s *userServer) SyncJoinedGroupMemberFaceURL(ctx context.Context, userID string, faceURL string, operationID string, opUserID string) {
	members, err := s.GetJoinedGroupMembers(ctx, userID)
	if err != nil {
		return
	}
	for _, group := range members {
		err := s.SetGroupMemberFaceURL(ctx, faceURL, group.GroupID, userID)
		if err != nil {
			return
		}
		chat.GroupMemberInfoSetNotification(operationID, opUserID, group.GroupID, userID)
	}
}

// ok
func (s *userServer) SyncJoinedGroupMemberNickname(ctx context.Context, userID string, newNickname, oldNickname string, operationID string, opUserID string) {
	members, err := s.GetJoinedGroupMembers(ctx, userID)
	if err != nil {
		return
	}
	for _, v := range members {
		if v.Nickname == oldNickname {
			err := s.SetGroupMemberNickname(ctx, newNickname, v.GroupID, v.UserID)
			if err != nil {
				return
			}
			chat.GroupMemberInfoSetNotification(operationID, opUserID, v.GroupID, userID)
		}
	}
}

// 设置群昵称
func (s *userServer) SetGroupMemberNickname(ctx context.Context, nickname string, groupID string, userID string) (err error) {
	return
}

// 设置群头像
func (s *userServer) SetGroupMemberFaceURL(ctx context.Context, faceURL string, groupID string, userID string) (err error) {
	return
}

// 获取加入的群成员信息
func (s *userServer) GetJoinedGroupMembers(ctx context.Context, userID string) (members []*sdkws.GroupMemberFullInfo, err error) {
	return
}

// ok
func (s *userServer) GetDesignateUsers(ctx context.Context, req *pbUser.GetDesignateUsersReq) (resp *pbUser.GetDesignateUsersResp, err error) {
	resp = &pbUser.GetDesignateUsersResp{}
	users, err := s.FindWithError(ctx, req.UserIDs)
	if err != nil {
		return nil, err
	}
	resp.UsersInfo, err = (*convert.DBUser)(nil).DB2PB(users)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *userServer) GetAllPageFriends(ctx context.Context, ownerUserID string) (resp []*sdkws.FriendInfo, err error) {
	return
}

// ok
func (s *userServer) UpdateUserInfo(ctx context.Context, req *pbUser.UpdateUserInfoReq) (resp *pbUser.UpdateUserInfoResp, err error) {
	resp = &pbUser.UpdateUserInfoResp{}
	err = tokenverify.CheckAccessV3(ctx, req.UserInfo.UserID)
	if err != nil {
		return nil, err
	}
	oldNickname := ""
	if req.UserInfo.Nickname != "" {
		u, err := s.FindWithError(ctx, []string{req.UserInfo.UserID})
		if err != nil {
			return nil, err
		}
		oldNickname = u[0].Nickname
	}
	user, err := convert.NewPBUser(req.UserInfo).Convert()
	if err != nil {
		return nil, err
	}
	err = s.Update(ctx, []*relationTb.UserModel{user})
	if err != nil {
		return nil, err
	}
	friends, err := s.GetAllPageFriends(ctx, req.UserInfo.UserID)
	if err != nil {
		return nil, err
	}
	go func() {
		for _, v := range friends {
			chat.FriendInfoUpdatedNotification(ctx, req.UserInfo.UserID, v.FriendUser.UserID, tracelog.GetOpUserID(ctx))
		}
	}()

	chat.UserInfoUpdatedNotification(ctx, tracelog.GetOpUserID(ctx), req.UserInfo.UserID)
	if req.UserInfo.FaceURL != "" {
		go s.SyncJoinedGroupMemberFaceURL(ctx, req.UserInfo.UserID, req.UserInfo.FaceURL, tracelog.GetOperationID(ctx), tracelog.GetOpUserID(ctx))
	}
	if req.UserInfo.Nickname != "" {
		go s.SyncJoinedGroupMemberNickname(ctx, req.UserInfo.UserID, req.UserInfo.Nickname, oldNickname, tracelog.GetOperationID(ctx), tracelog.GetOpUserID(ctx))
	}
	return resp, nil
}

// ok
func (s *userServer) SetGlobalRecvMessageOpt(ctx context.Context, req *pbUser.SetGlobalRecvMessageOptReq) (resp *pbUser.SetGlobalRecvMessageOptResp, err error) {
	resp = &pbUser.SetGlobalRecvMessageOptResp{}
	if _, err := s.FindWithError(ctx, []string{req.UserID}); err != nil {
		return nil, err
	}
	m := make(map[string]interface{}, 1)
	m["global_recv_msg_opt"] = req.GlobalRecvMsgOpt
	if err := s.UpdateByMap(ctx, req.UserID, m); err != nil {
		return nil, err
	}
	chat.UserInfoUpdatedNotification(ctx, req.UserID, req.UserID)
	return resp, nil
}

// ok
func (s *userServer) AccountCheck(ctx context.Context, req *pbUser.AccountCheckReq) (resp *pbUser.AccountCheckResp, err error) {
	resp = &pbUser.AccountCheckResp{}
	if utils.Duplicate(req.CheckUserIDs) {
		return nil, constant.ErrArgs.Wrap("userID repeated")
	}
	err = tokenverify.CheckAdmin(ctx)
	if err != nil {
		return nil, err
	}
	users, err := s.Find(ctx, req.CheckUserIDs)
	if err != nil {
		return nil, err
	}
	userIDs := make(map[string]interface{}, 0)
	for _, v := range users {
		userIDs[v.UserID] = nil
	}
	for _, v := range req.CheckUserIDs {
		temp := &pbUser.AccountCheckRespSingleUserStatus{UserID: v}
		if _, ok := userIDs[v]; ok {
			temp.AccountStatus = constant.Registered
		} else {
			temp.AccountStatus = constant.UnRegistered
		}
		resp.Results = append(resp.Results, temp)
	}
	return resp, nil
}

// ok
func (s *userServer) GetPaginationUsers(ctx context.Context, req *pbUser.GetPaginationUsersReq) (resp *pbUser.GetPaginationUsersResp, err error) {
	resp = &pbUser.GetPaginationUsersResp{}
	usersDB, total, err := s.Page(ctx, req.Pagination.PageNumber, req.Pagination.ShowNumber)
	if err != nil {
		return nil, err
	}
	resp.Total = int32(total)
	resp.Users, err = (*convert.DBUser)(nil).DB2PB(usersDB)
	return resp, nil
}

// ok
func (s *userServer) UserRegister(ctx context.Context, req *pbUser.UserRegisterReq) (resp *pbUser.UserRegisterResp, err error) {
	resp = &pbUser.UserRegisterResp{}
	if utils.DuplicateAny(req.Users, func(e *sdkws.UserInfo) string { return e.UserID }) {
		return nil, constant.ErrArgs.Wrap("userID repeated")
	}
	userIDs := make([]string, 0)
	for _, v := range req.Users {
		userIDs = append(userIDs, v.UserID)
	}

	exist, err := s.IsExist(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	if exist {
		return nil, constant.ErrRegisteredAlready.Wrap("userID registered already")
	}
	users, err := (*convert.PBUser)(nil).PB2DB(req.Users)
	if err != nil {
		return nil, err
	}
	err = s.Create(ctx, users)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
