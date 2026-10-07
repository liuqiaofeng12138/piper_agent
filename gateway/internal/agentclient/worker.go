package agentclient

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	agentv1 "piper_agent/gateway/pkg/pb/agent/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type WorkerClient struct {
	address string
	mu      sync.Mutex
	conn    *grpc.ClientConn
	stub    agentv1.AgentWorkerServiceClient
}

func New(address string) *WorkerClient {
	return &WorkerClient{address: address}
}

func (c *WorkerClient) closeConn() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.stub = nil
	}
}

func (c *WorkerClient) connect(ctx context.Context) (agentv1.AgentWorkerServiceClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stub != nil {
		return c.stub, nil
	}
	conn, err := grpc.NewClient(
		c.address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	c.stub = agentv1.NewAgentWorkerServiceClient(conn)
	return c.stub, nil
}

func (c *WorkerClient) Health(ctx context.Context) error {
	stub, err := c.connect(ctx)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	resp, err := stub.Health(ctx, &agentv1.HealthRequest{})
	if err != nil {
		return err
	}
	if resp.Status != "ok" {
		return fmt.Errorf("worker status: %s", resp.Status)
	}
	return nil
}

func (c *WorkerClient) CancelRun(ctx context.Context, runID string) error {
	stub, err := c.connect(ctx)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = stub.Cancel(ctx, &agentv1.CancelRequest{RunId: runID})
	return err
}

func (c *WorkerClient) Execute(
	ctx context.Context,
	req *agentv1.ExecuteRequest,
	onEvent func(*agentv1.ExecuteEvent) error,
) error {
	stub, err := c.connect(ctx)
	if err != nil {
		return err
	}
	log.Printf("[worker] execute start run=%s conv=%s", req.RunId, req.ConversationId)
	stream, err := stub.Execute(ctx, req)
	if err != nil {
		log.Printf("[worker] execute open failed run=%s err=%v", req.RunId, err)
		return err
	}
	for {
		ev, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Printf("[worker] execute done run=%s", req.RunId)
				return nil
			}
			st, ok := status.FromError(err)
			if ok && (st.Code() == codes.Canceled || st.Code() == codes.DeadlineExceeded) {
				log.Printf("[worker] execute closed run=%s code=%s", req.RunId, st.Code())
				return nil
			}
			log.Printf("[worker] execute recv error run=%s err=%v", req.RunId, err)
			return err
		}
		if onEvent != nil {
			if err := onEvent(ev); err != nil {
				return err
			}
		}
	}
}

func (c *WorkerClient) Close() {
	c.mu.Lock()
	c.closeConn()
	c.mu.Unlock()
}
