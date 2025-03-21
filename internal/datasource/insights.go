package datasource

import (
	"context"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	pb "deps.dev/api/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// CachedInsightsClient is a wrapper for InsightsClient that caches requests.
type CachedInsightsClient struct {
	pb.InsightsClient

	// cache fields
	mu                sync.Mutex
	cacheTimestamp    *time.Time
	packageCache      *RequestCache[packageKey, *pb.Package]
	versionCache      *RequestCache[versionKey, *pb.Version]
	requirementsCache *RequestCache[versionKey, *pb.Requirements]
}

// Comparable types to use as map keys for cache.
type packageKey struct {
	System pb.System
	Name   string
}

func makePackageKey(k *pb.PackageKey) packageKey {
	return packageKey{
		System: k.GetSystem(),
		Name:   k.GetName(),
	}
}

type versionKey struct {
	System  pb.System
	Name    string
	Version string
}

func makeVersionKey(k *pb.VersionKey) versionKey {
	return versionKey{
		System:  k.GetSystem(),
		Name:    k.GetName(),
		Version: k.GetVersion(),
	}
}

func NewCachedInsightsClient(addr string, userAgent string) (*CachedInsightsClient, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("getting system cert pool: %w", err)
	}
	creds := credentials.NewClientTLSFromCert(certPool, "")
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(creds)}

	if userAgent != "" {
		dialOpts = append(dialOpts, grpc.WithUserAgent(userAgent))
	}

	conn, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("dialling %q: %w", addr, err)
	}

	return &CachedInsightsClient{
		InsightsClient:    pb.NewInsightsClient(conn),
		packageCache:      NewRequestCache[packageKey, *pb.Package](),
		versionCache:      NewRequestCache[versionKey, *pb.Version](),
		requirementsCache: NewRequestCache[versionKey, *pb.Requirements](),
	}, nil
}

func (c *CachedInsightsClient) GetPackage(ctx context.Context, in *pb.GetPackageRequest, opts ...grpc.CallOption) (*pb.Package, error) {
	return c.packageCache.Get(makePackageKey(in.GetPackageKey()), func() (*pb.Package, error) {
		return c.InsightsClient.GetPackage(ctx, in, opts...)
	})
}

func (c *CachedInsightsClient) GetVersion(ctx context.Context, in *pb.GetVersionRequest, opts ...grpc.CallOption) (*pb.Version, error) {
	return c.versionCache.Get(makeVersionKey(in.GetVersionKey()), func() (*pb.Version, error) {
		return c.InsightsClient.GetVersion(ctx, in, opts...)
	})
}

func (c *CachedInsightsClient) GetRequirements(ctx context.Context, in *pb.GetRequirementsRequest, opts ...grpc.CallOption) (*pb.Requirements, error) {
	return c.requirementsCache.Get(makeVersionKey(in.GetVersionKey()), func() (*pb.Requirements, error) {
		return c.InsightsClient.GetRequirements(ctx, in, opts...)
	})
}
