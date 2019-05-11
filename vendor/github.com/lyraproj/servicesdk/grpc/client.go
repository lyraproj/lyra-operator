package grpc

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
	"github.com/lyraproj/issue/issue"
	"github.com/lyraproj/pcore/px"
	"github.com/lyraproj/pcore/types"
	"github.com/lyraproj/servicesdk/serviceapi"
	"github.com/lyraproj/servicesdk/servicepb"
	"google.golang.org/grpc"

	// Ensure that service is initialized
	_ "github.com/lyraproj/servicesdk/service"
)

var handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "PLUGIN_MAGIC_COOKIE",
	MagicCookieValue: "7468697320697320616e20616d617a696e67206d6167696320636f6f6b69652c206e6f6d206e6f6d206e6f6d",
}

type PluginClient struct {
	plugin.NetRPCUnsupportedPlugin
}

func (a *PluginClient) GRPCServer(*plugin.GRPCBroker, *grpc.Server) error {
	return fmt.Errorf(`%T has no server implementation for rpc`, a)
}

func (a *PluginClient) GRPCClient(ctx context.Context, broker *plugin.GRPCBroker, clientConn *grpc.ClientConn) (interface{}, error) {
	return &Client{client: servicepb.NewDefinitionServiceClient(clientConn)}, nil
}

type Client struct {
	client servicepb.DefinitionServiceClient
}

func (c *Client) Identifier(ctx px.Context) px.TypedName {
	rr, err := c.client.Identity(ctx, &servicepb.EmptyRequest{})
	if err != nil {
		panic(err)
	}
	return FromDataPB(ctx, rr).(px.TypedName)
}

func (c *Client) Invoke(ctx px.Context, identifier, name string, arguments ...px.Value) px.Value {
	rq := servicepb.InvokeRequest{
		Identifier: identifier,
		Method:     name,
		Arguments:  ToDataPB(ctx, types.WrapValues(arguments)),
	}
	rr, err := c.client.Invoke(ctx, &rq)
	if err != nil {
		panic(err)
	}
	result := FromDataPB(ctx, rr)
	if eo, ok := result.(serviceapi.ErrorObject); ok {
		panic(px.Error(InvocationError, issue.H{`identifier`: identifier, `name`: name, `code`: eo.IssueCode(), `message`: eo.Message()}))
	}
	return result
}

func (c *Client) Metadata(ctx px.Context) (typeSet px.TypeSet, definitions []serviceapi.Definition) {
	rr, err := c.client.Metadata(ctx, &servicepb.EmptyRequest{})
	if err != nil {
		panic(err)
	}
	if ts := rr.GetTypeset(); ts != nil {
		typeSet = FromDataPB(ctx, rr.GetTypeset()).(px.TypeSet)
	}
	ds := FromDataPB(ctx, rr.GetDefinitions()).(px.List)
	definitions = make([]serviceapi.Definition, ds.Len())
	ds.EachWithIndex(func(d px.Value, i int) { definitions[i] = d.(serviceapi.Definition) })
	return
}

func (c *Client) State(ctx px.Context, identifier string, parameters px.OrderedMap) px.PuppetObject {
	rq := servicepb.StateRequest{Identifier: identifier, Parameters: ToDataPB(ctx, parameters)}
	rr, err := c.client.State(ctx, &rq)
	if err != nil {
		panic(err)
	}
	return FromDataPB(ctx, rr).(px.PuppetObject)
}

// Load  ...
func Load(cmd *exec.Cmd, logger hclog.Logger) (serviceapi.Service, error) {
	if logger == nil {
		logger = hclog.Default()
	}

	level := "warn"
	switch {
	case logger.IsTrace():
		level = "trace"
	case logger.IsDebug():
		level = "debug"
	case logger.IsInfo():
		level = "info"
	case logger.IsWarn():
		level = "warn"
	case logger.IsError():
		level = "error"
	}
	cmd.Env = append(cmd.Env, fmt.Sprintf("LYRA_LOG_LEVEL=%s", level))

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: handshake,
		Plugins: map[string]plugin.Plugin{
			"server": &PluginClient{},
		},
		Managed:          true,
		Cmd:              cmd,
		Logger:           logger,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
	})

	grpcClient, err := client.Client()
	if err != nil {
		hclog.Default().Error("error creating GRPC client", "error", err)
		return nil, err
	}

	// Request the plugin
	pluginName := "server"
	raw, err := grpcClient.Dispense(pluginName)
	if err != nil {
		hclog.Default().Error("error dispensing plugin", "plugin", pluginName, "error", err)
		return nil, err
	}
	return raw.(serviceapi.Service), nil
}
