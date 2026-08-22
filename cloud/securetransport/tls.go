// Package securetransport 提供 Cloud 跨进程连接共用的 TLS 1.3 配置与 Edge 证书身份解析。
package securetransport

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/anytty/anytty/shared/timepolicy"
)

const (
	edgeIdentityScheme = "spiffe"
	edgeIdentityHost   = "anytty.com"
	edgeIdentityPrefix = "/edge/"
)

// ServerOptions 定义 Cloud gRPC/HTTPS server 的证书材料。
// ClientCAFile 非空时 server 强制校验 mTLS 客户端证书。
type ServerOptions struct {
	CertificateFile string
	PrivateKeyFile  string
	ClientCAFile    string
}

// NewServerTLSConfig 加载 server 证书并返回最低 TLS 1.3 的不可变装配。
// 文件不存在、PEM 无效或 Client CA 为空时显式失败。
func NewServerTLSConfig(options ServerOptions) (*tls.Config, error) {
	config, _, err := NewReloadableServerTLSConfig(options)
	return config, err
}

// ReloadableCertificate 是 server TLS 当前证书的原子内存 owner。
// Replace 只接受调用方已经完整校验过的不可变 key pair。
type ReloadableCertificate struct {
	current atomic.Pointer[tls.Certificate]
}

// NewReloadableServerTLSConfig 加载初始文件，并把后续新握手绑定到可原子替换的 loader。
func NewReloadableServerTLSConfig(options ServerOptions) (*tls.Config, *ReloadableCertificate, error) {
	certificate, err := tls.LoadX509KeyPair(strings.TrimSpace(options.CertificateFile), strings.TrimSpace(options.PrivateKeyFile))
	if err != nil {
		return nil, nil, fmt.Errorf("load server certificate: %w", err)
	}
	loader := &ReloadableCertificate{}
	loader.Replace(&certificate)
	config := &tls.Config{
		MinVersion: tls.VersionTLS13,
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			current := loader.current.Load()
			if current == nil {
				return nil, errors.New("server certificate is unavailable")
			}
			return current, nil
		},
		NextProtos: []string{"h2", "http/1.1"},
	}
	if strings.TrimSpace(options.ClientCAFile) == "" {
		return config, loader, nil
	}
	pool, err := loadCertPool(options.ClientCAFile)
	if err != nil {
		return nil, nil, fmt.Errorf("load client CA: %w", err)
	}
	config.ClientAuth = tls.RequireAndVerifyClientCert
	config.ClientCAs = pool
	return config, loader, nil
}

// Replace 原子替换后续 TLS 握手使用的证书；调用后不再修改输入对象。
func (loader *ReloadableCertificate) Replace(certificate *tls.Certificate) {
	if loader == nil || certificate == nil {
		return
	}
	loader.current.Store(certificate)
}

// ValidatedServerPair 是一次完整校验后的 server key pair 和叶证书。
type ValidatedServerPair struct {
	Certificate *tls.Certificate
	Leaf        *x509.Certificate
}

// ValidateServerPair 校验证书链、私钥、DNS SAN、可选公网入口和有效期。
// now 为零值时只用于恢复本机旧证书，不检查有效期，以便 Edge 仍能连回 Controller 收敛。
func ValidateServerPair(certificatePEM, privateKeyPEM []byte, publicEndpoint string, now time.Time) (*ValidatedServerPair, error) {
	pair, err := tls.X509KeyPair(certificatePEM, privateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("certificate and private key do not match: %w", err)
	}
	if len(pair.Certificate) == 0 {
		return nil, errors.New("certificate chain contains no certificate")
	}
	var leaf *x509.Certificate
	for index, der := range pair.Certificate {
		certificate, parseErr := x509.ParseCertificate(der)
		if parseErr != nil {
			return nil, fmt.Errorf("parse certificate chain item %d: %w", index+1, parseErr)
		}
		if index == 0 {
			leaf = certificate
		}
	}
	if len(leaf.DNSNames) == 0 && len(leaf.IPAddresses) == 0 {
		return nil, errors.New("certificate must contain at least one DNS or IP SAN")
	}
	if !now.IsZero() {
		now = now.UTC()
		if !leaf.NotAfter.After(leaf.NotBefore) || leaf.NotBefore.After(now.Add(timepolicy.ClockSkewTolerance)) || !leaf.NotAfter.After(now.Add(-timepolicy.ClockSkewTolerance)) {
			return nil, errors.New("certificate is not currently valid")
		}
	}
	if endpoint := strings.TrimSpace(publicEndpoint); endpoint != "" {
		host := endpoint
		if parsed, _, splitErr := net.SplitHostPort(host); splitErr == nil {
			host = strings.Trim(parsed, "[]")
		}
		if err := leaf.VerifyHostname(host); err != nil {
			return nil, fmt.Errorf("certificate does not cover public endpoint %q: %w", host, err)
		}
	}
	pair.Leaf = leaf
	return &ValidatedServerPair{Certificate: &pair, Leaf: leaf}, nil
}

// ClientOptions 定义 Edge 连接 Controller 时的 mTLS 材料和 TLS server name。
type ClientOptions struct {
	CertificateFile string
	PrivateKeyFile  string
	RootCAFile      string
	ServerName      string
	// GetClientCertificate allows an EdgeIdentity manager to atomically rotate
	// the credential used by future handshakes without disturbing an existing stream.
	GetClientCertificate func(*tls.CertificateRequestInfo) (*tls.Certificate, error)
}

// NewClientTLSConfig 加载 EdgeIdentity 客户端证书与 Controller CA。
// ServerName 是必填信任边界，禁止 InsecureSkipVerify 或系统 CA fallback。
func NewClientTLSConfig(options ClientOptions) (*tls.Config, error) {
	serverName := strings.TrimSpace(options.ServerName)
	if serverName == "" {
		return nil, errors.New("controller TLS server name is required")
	}
	var certificates []tls.Certificate
	if options.GetClientCertificate == nil {
		certificate, err := tls.LoadX509KeyPair(strings.TrimSpace(options.CertificateFile), strings.TrimSpace(options.PrivateKeyFile))
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}
		certificates = []tls.Certificate{certificate}
	}
	pool, err := loadCertPool(options.RootCAFile)
	if err != nil {
		return nil, fmt.Errorf("load controller CA: %w", err)
	}
	return &tls.Config{
		MinVersion:           tls.VersionTLS13,
		Certificates:         certificates,
		GetClientCertificate: options.GetClientCertificate,
		RootCAs:              pool,
		ServerName:           serverName,
		NextProtos:           []string{"h2"},
	}, nil
}

// EdgeCACertificateDERFingerprint 返回 locator 中唯一 CA 证书 DER 的 SHA-256。
// pairing 二维码只携带该 pin；完整 CA 仍保留在 enrollment 和配对成功后的 EdgeLocator 中。
func EdgeCACertificateDERFingerprint(certificatePEM []byte) ([]byte, error) {
	rest := bytes.TrimSpace(certificatePEM)
	block, trailing := pem.Decode(rest)
	if block == nil || block.Type != "CERTIFICATE" || len(block.Headers) != 0 || len(bytes.TrimSpace(trailing)) != 0 {
		return nil, errors.New("Edge CA PEM must contain exactly one certificate")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil || !certificate.IsCA {
		return nil, errors.New("Edge CA certificate is invalid")
	}
	digest := sha256.Sum256(certificate.Raw)
	return append([]byte(nil), digest[:]...), nil
}

// NewPinnedEdgeClientTLSConfig 验证 Edge 实际发送的完整证书链，并把唯一信任锚固定到二维码中的 DER pin。
// InsecureSkipVerify 只关闭系统默认 roots；VerifyConnection 无条件执行等价的 hostname、时间、EKU 和链验证。
func NewPinnedEdgeClientTLSConfig(serverName string, caCertificateDERFingerprint []byte, now func() time.Time) (*tls.Config, error) {
	serverName = strings.TrimSpace(serverName)
	if serverName == "" || len(caCertificateDERFingerprint) != sha256.Size {
		return nil, errors.New("pinned Edge TLS server name and CA fingerprint are required")
	}
	pinned := append([]byte(nil), caCertificateDERFingerprint...)
	if now == nil {
		now = time.Now
	}
	return &tls.Config{
		MinVersion:         tls.VersionTLS13,
		ServerName:         serverName,
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true, // Verification is replaced, not disabled; see VerifyConnection below.
		VerifyConnection: func(state tls.ConnectionState) error {
			return verifyPinnedEdgeConnection(state, serverName, pinned, now().UTC())
		},
	}, nil
}

func verifyPinnedEdgeConnection(state tls.ConnectionState, serverName string, pinned []byte, now time.Time) error {
	if len(state.PeerCertificates) < 2 {
		return errors.New("Edge TLS server did not send a complete certificate chain")
	}
	var trustAnchor *x509.Certificate
	for _, certificate := range state.PeerCertificates[1:] {
		digest := sha256.Sum256(certificate.Raw)
		if bytes.Equal(digest[:], pinned) {
			if trustAnchor != nil {
				return errors.New("Edge TLS chain contains duplicate pinned trust anchors")
			}
			trustAnchor = certificate
		}
	}
	if trustAnchor == nil || !trustAnchor.IsCA {
		return errors.New("Edge TLS chain does not contain the pinned CA certificate")
	}
	roots := x509.NewCertPool()
	roots.AddCert(trustAnchor)
	intermediates := x509.NewCertPool()
	for _, certificate := range state.PeerCertificates[1:] {
		if !bytes.Equal(certificate.Raw, trustAnchor.Raw) {
			intermediates.AddCert(certificate)
		}
	}
	leaf := state.PeerCertificates[0]
	verificationTime := certificateVerificationTime(leaf, now)
	_, err := leaf.Verify(x509.VerifyOptions{
		DNSName: serverName, Roots: roots, Intermediates: intermediates, CurrentTime: verificationTime,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		return fmt.Errorf("verify pinned Edge TLS certificate: %w", err)
	}
	return nil
}

func certificateVerificationTime(certificate *x509.Certificate, now time.Time) time.Time {
	if certificate.NotBefore.After(now) && !certificate.NotBefore.After(now.Add(timepolicy.ClockSkewTolerance)) {
		return certificate.NotBefore
	}
	if !certificate.NotAfter.After(now) && certificate.NotAfter.After(now.Add(-timepolicy.ClockSkewTolerance)) {
		return certificate.NotAfter.Add(-time.Nanosecond)
	}
	return now
}

// EdgeIdentityURI 返回 EdgeIdentity 证书唯一允许的 URI SAN。
// Edge ID 是 Controller 生成的不透明 ID，不允许路径分隔符或空白。
func EdgeIdentityURI(edgeID string) (*url.URL, error) {
	edgeID = strings.TrimSpace(edgeID)
	if edgeID == "" || strings.ContainsAny(edgeID, "/\\") || strings.ContainsFunc(edgeID, func(value rune) bool { return value <= ' ' }) {
		return nil, fmt.Errorf("invalid edge ID %q", edgeID)
	}
	return &url.URL{Scheme: edgeIdentityScheme, Host: edgeIdentityHost, Path: edgeIdentityPrefix + edgeID}, nil
}

// EdgeIDFromCertificate 从已完成 CA 校验的客户端证书解析 Edge ID。
// 缺失、重复或格式错误的 Edge URI SAN 都会拒绝连接。
func EdgeIDFromCertificate(certificate *x509.Certificate) (string, error) {
	if certificate == nil {
		return "", errors.New("client certificate is missing")
	}
	var edgeID string
	for _, identity := range certificate.URIs {
		if identity == nil || identity.Scheme != edgeIdentityScheme || identity.Host != edgeIdentityHost || !strings.HasPrefix(identity.Path, edgeIdentityPrefix) {
			continue
		}
		if identity.User != nil || identity.Opaque != "" || identity.RawPath != "" || identity.ForceQuery || identity.RawQuery != "" || identity.Fragment != "" {
			return "", errors.New("Edge URI SAN contains unsupported URI components")
		}
		candidate := strings.TrimPrefix(identity.Path, edgeIdentityPrefix)
		if _, err := EdgeIdentityURI(candidate); err != nil {
			return "", fmt.Errorf("invalid Edge URI SAN: %w", err)
		}
		if edgeID != "" {
			return "", errors.New("client certificate contains multiple Edge URI SANs")
		}
		edgeID = candidate
	}
	if edgeID == "" {
		return "", errors.New("client certificate does not contain an Edge URI SAN")
	}
	return edgeID, nil
}

func loadCertPool(path string) (*x509.CertPool, error) {
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(payload) {
		return nil, errors.New("PEM contains no certificates")
	}
	return pool, nil
}
