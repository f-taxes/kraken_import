package grpc_client

import (
	"context"
	"time"

	"github.com/f-taxes/kraken_import/global"
	"github.com/f-taxes/kraken_import/proto"
	"github.com/kataras/golog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var GrpcClient *FTaxesClient

type FTaxesClient struct {
	conStr     string
	Connection *grpc.ClientConn
	GrpcClient proto.FTaxesClient
}

func NewFTaxesClient(conStr string) *FTaxesClient {
	return &FTaxesClient{
		conStr: conStr,
	}
}

func (c *FTaxesClient) Connect(ctx context.Context) error {
	con, err := grpc.DialContext(ctx, c.conStr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithConnectParams(grpc.ConnectParams{
		MinConnectTimeout: time.Second * 30,
		Backoff:           backoff.Config{MaxDelay: time.Second},
	}))

	if err != nil {
		golog.Errorf("Failed to establish grpc connections: %v", err)
		return err
	}

	go func() {
		state := con.GetState()
		for {
			golog.Infof("Connection state: %s", state.String())
			con.WaitForStateChange(context.Background(), state)
			state = con.GetState()
		}
	}()

	c.Connection = con
	c.GrpcClient = proto.NewFTaxesClient(con)

	return nil
}

func (c *FTaxesClient) SubmitTrade(ctx context.Context, t *proto.Trade) error {
	t.Plugin = global.Plugin.ID
	t.PluginVersion = global.Plugin.Version
	t.Created = timestamppb.Now()
	_, err := c.GrpcClient.SubmitTrade(ctx, t)
	return err
}

func (c *FTaxesClient) SubmitTransfer(ctx context.Context, transfer *proto.Transfer) error {
	transfer.Plugin = global.Plugin.ID
	transfer.PluginVersion = global.Plugin.Version
	transfer.Created = timestamppb.Now()
	_, err := c.GrpcClient.SubmitTransfer(ctx, transfer)
	return err
}

func (c *FTaxesClient) SubmitGenericFee(ctx context.Context, gf *proto.SrcGenericFee) error {
	_, err := c.GrpcClient.SubmitGenericFee(ctx, gf)
	return err
}

func (c *FTaxesClient) ShowJobProgress(ctx context.Context, job *proto.JobProgress) error {
	job.Plugin = global.Plugin.Label
	_, err := c.GrpcClient.ShowJobProgress(ctx, job)
	return err
}

func (c *FTaxesClient) GetSettings(ctx context.Context) (*proto.Settings, error) {
	settings, err := c.GrpcClient.GetSettings(ctx, nil)
	if err != nil {
		return nil, err
	}
	return settings, nil
}

func (c *FTaxesClient) AppLog(ctx context.Context, msg *proto.AppLogMsg, opts ...grpc.CallOption) error {
	_, err := c.GrpcClient.AppLog(ctx, msg)
	return err
}
