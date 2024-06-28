package rpcserver

import (
	"context"
	"errors"
	"github.com/czh0526/btc-wallet/netparams"
	"github.com/czh0526/btc-wallet/wallet"
	"google.golang.org/grpc"

	pb "github.com/czh0526/btc-wallet/rpc/walletrpc"
)

///////////////////////////////
//	LoaderServer
///////////////////////////////

type loaderServer struct {
	pb.UnimplementedWalletLoaderServiceServer
	loader    *wallet.Loader
	activeNet *netparams.Params
}

func (s *loaderServer) CreateWallet(ctx context.Context, req *pb.CreateWalletRequest) (
	*pb.CreateWalletResponse, error) {

	return nil, errors.New("CreateWallet is not implemented yet")
}

func (s *loaderServer) OpenWallet(ctx context.Context, req *pb.OpenWalletRequest) (
	*pb.OpenWalletResponse, error) {

	return nil, errors.New("OpenWallet is not implemented yet")
}

func (s *loaderServer) WalletExists(ctx context.Context, req *pb.WalletExistsRequest) (
	*pb.WalletExistsResponse, error) {

	return nil, errors.New("WalletExists is not implemented yet")
}

func StartWalletLoaderService(server *grpc.Server, loader *wallet.Loader, activeNet *netparams.Params) {
	service := &loaderServer{
		loader:    loader,
		activeNet: activeNet,
	}
	pb.RegisterWalletLoaderServiceServer(server, service)
}

//////////////////////////
//  WalletServer
//////////////////////////

type walletServer struct {
	pb.UnimplementedWalletServiceServer
	wallet *wallet.Wallet
}

func StartWalletService(server *grpc.Server, wallet *wallet.Wallet) {
	service := &walletServer{
		wallet: wallet,
	}
	pb.RegisterWalletServiceServer(server, service)
}
