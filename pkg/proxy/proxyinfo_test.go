package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"istio.io/istio/pilot/pkg/xds"
	"istio.io/istio/pkg/kube"
)

type fakeKubeClient struct {
	kube.CLIClient     // embed for interface
	allDiscoveryDoFunc func(ctx context.Context, ns, path string) (map[string][]byte, error)
}

func (f *fakeKubeClient) AllDiscoveryDo(ctx context.Context, ns, path string) (map[string][]byte, error) {
	return f.allDiscoveryDoFunc(ctx, ns, path)
}

func TestGetProxyInfo_Success(t *testing.T) {
	// Prepare a fake syncz response
	status := &sidecarSyncStatus{
		SyncStatus: xds.SyncStatus{
			ProxyID:      "sidecar~10.0.0.1~foo.default~cluster.local",
			ProxyType:    "sidecar",
			IstioVersion: "1.18.0",
		},
	}
	b, _ := json.Marshal([]*sidecarSyncStatus{status})
	fake := &fakeKubeClient{
		allDiscoveryDoFunc: func(ctx context.Context, ns, path string) (map[string][]byte, error) {
			return map[string][]byte{"pilot": b}, nil
		},
	}
	infos, err := GetProxyInfo(fake, "istio-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(*infos) != 1 {
		t.Fatalf("expected 1 proxy info, got %d", len(*infos))
	}
	pi := (*infos)[0]
	if pi.ID != "sidecar~10.0.0.1~foo.default~cluster.local" || pi.IstioVersion != "1.18.0" || pi.Type != "sidecar" {
		t.Errorf("unexpected proxy info: %+v", pi)
	}
}

func TestGetProxyInfo_AllDiscoveryDoError(t *testing.T) {
	fake := &fakeKubeClient{
		allDiscoveryDoFunc: func(ctx context.Context, ns, path string) (map[string][]byte, error) {
			return nil, errors.New("fail")
		},
	}
	_, err := GetProxyInfo(fake, "istio-system")
	if err == nil || err.Error() != "fail" {
		t.Errorf("expected error 'fail', got %v", err)
	}
}

func TestGetProxyInfo_JSONUnmarshalError(t *testing.T) {
	fake := &fakeKubeClient{
		allDiscoveryDoFunc: func(ctx context.Context, ns, path string) (map[string][]byte, error) {
			return map[string][]byte{"pilot": []byte("notjson")}, nil
		},
	}
	_, err := GetProxyInfo(fake, "istio-system")
	if err == nil {
		t.Error("expected JSON unmarshal error, got nil")
	}
}

func TestGetIDsFromProxyInfo_Success(t *testing.T) {
	status := &sidecarSyncStatus{
		SyncStatus: xds.SyncStatus{
			ProxyID:      "sidecar~10.0.0.1~foo.default~cluster.local",
			ProxyType:    "sidecar",
			IstioVersion: "1.18.0",
		},
	}
	b, _ := json.Marshal([]*sidecarSyncStatus{status})
	fake := &fakeKubeClient{
		allDiscoveryDoFunc: func(ctx context.Context, ns, path string) (map[string][]byte, error) {
			return map[string][]byte{"pilot": b}, nil
		},
	}
	ids, err := GetIDsFromProxyInfo(fake, "istio-system")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"sidecar~10.0.0.1~foo.default~cluster.local"}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("got %v, want %v", ids, want)
	}
}

func TestGetIDsFromProxyInfo_Error(t *testing.T) {
	fake := &fakeKubeClient{
		allDiscoveryDoFunc: func(ctx context.Context, ns, path string) (map[string][]byte, error) {
			return nil, errors.New("fail")
		},
	}
	_, err := GetIDsFromProxyInfo(fake, "istio-system")
	if err == nil || err.Error() != "failed to get proxy infos: fail" {
		t.Errorf("expected error, got %v", err)
	}
}
