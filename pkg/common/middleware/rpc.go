package middleware

import (
	"OpenIM/pkg/common/constant"
	"OpenIM/pkg/common/log"
	"OpenIM/pkg/common/tracelog"
	"OpenIM/pkg/errs"
	"OpenIM/pkg/utils"
	"context"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"runtime/debug"
)

func RpcServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	var operationID string
	defer func() {
		if r := recover(); r != nil {
			log.NewError(operationID, info.FullMethod, "type:", fmt.Sprintf("%T", r), "panic:", r, "stack:", string(debug.Stack()))
		}
	}()
	//funcName := path.Base(info.FullMethod)
	funcName := info.FullMethod
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.New(codes.InvalidArgument, "missing metadata").Err()
	}
	if opts := md.Get(constant.OperationID); len(opts) != 1 || opts[0] == "" {
		return nil, status.New(codes.InvalidArgument, "operationID error").Err()
	} else {
		operationID = opts[0]
	}
	var opUserID string
	if opts := md.Get("opUserID"); len(opts) == 1 {
		opUserID = opts[0]
	}
	ctx = tracelog.NewRpcCtx(ctx, funcName, operationID)
	defer log.ShowLog(ctx)
	tracelog.SetCtxInfo(ctx, funcName, err, "opUserID", opUserID, "rpcReq", rpcString(req))
	resp, err = handler(ctx, req)
	if err != nil {
		tracelog.SetCtxInfo(ctx, funcName, err)

		errInfo := constant.ToAPIErrWithErr(err)
		var code codes.Code
		if errInfo.ErrCode == 0 {
			code = codes.Unknown
		} else {
			code = codes.Code(errInfo.ErrCode)
		}
		sta, err := status.New(code, errInfo.ErrMsg).WithDetails(wrapperspb.String(errInfo.DetailErrMsg))
		if err != nil {
			return nil, err
		}
		return nil, sta.Err()
	}
	tracelog.SetCtxInfo(ctx, funcName, nil, "rpcResp", rpcString(resp))
	return
}

func rpcString(v interface{}) string {
	if s, ok := v.(interface{ String() string }); ok {
		return s.String()
	}
	return fmt.Sprintf("%+v", v)
}

func RpcClientInterceptor(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
	//if cc == nil {
	//	return utils.Wrap(errs.ErrRpcConn, "")
	//}
	operationID, ok := ctx.Value(constant.OperationID).(string)
	if !ok {
		return utils.Wrap(errs.ErrArgs, "ctx missing operationID")
	}
	opUserID, ok := ctx.Value("opUserID").(string)
	if !ok {
		return utils.Wrap(errs.ErrArgs, "ctx missing opUserID")
	}
	md := metadata.Pairs(constant.OperationID, operationID, "opUserID", opUserID)
	return invoker(metadata.NewOutgoingContext(ctx, md), method, req, reply, cc, opts...)
}
