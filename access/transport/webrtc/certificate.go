package webrtc

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/anytty/anytty/shared/remoteauth"
	pion "github.com/pion/webrtc/v4"
)

// LocalCertificateFingerprint 返回当前 PeerConnection 实际使用的本端 SHA-256 DTLS certificate fingerprint。
// daemon 用它绑定 DeviceHello；该值来自 Pion DTLSTransport，而不是 signaling SDP 或 Companion 返回字段。
func LocalCertificateFingerprint(peer *pion.PeerConnection) (string, error) {
	dtls, err := dtlsTransport(peer)
	if err != nil {
		return "", err
	}
	parameters, err := dtls.GetLocalParameters()
	if err != nil {
		return "", fmt.Errorf("read local DTLS parameters: %w", err)
	}
	for _, fingerprint := range parameters.Fingerprints {
		if strings.EqualFold(strings.TrimSpace(fingerprint.Algorithm), "sha-256") {
			return remoteauth.NormalizeDTLSCertificateFingerprint("sha-256:" + fingerprint.Value)
		}
	}
	return "", fmt.Errorf("local DTLS transport has no SHA-256 certificate fingerprint")
}

// RemoteCertificateFingerprint 计算当前 PeerConnection 实际对端证书的 SHA-256 fingerprint。
// client 用它校验 DeviceHello 的 DTLS binding；无法读取对端原始证书的平台必须 fail closed。
func RemoteCertificateFingerprint(peer *pion.PeerConnection) (string, error) {
	dtls, err := dtlsTransport(peer)
	if err != nil {
		return "", err
	}
	certificate := dtls.GetRemoteCertificate()
	if len(certificate) == 0 {
		return "", fmt.Errorf("remote DTLS certificate is unavailable")
	}
	digest := sha256.Sum256(certificate)
	parts := make([]string, len(digest))
	for index, octet := range digest {
		parts[index] = fmt.Sprintf("%02x", octet)
	}
	return remoteauth.NormalizeDTLSCertificateFingerprint("sha-256:" + strings.Join(parts, ":"))
}

func dtlsTransport(peer *pion.PeerConnection) (*pion.DTLSTransport, error) {
	if peer == nil || peer.SCTP() == nil || peer.SCTP().Transport() == nil {
		return nil, fmt.Errorf("peer connection DTLS transport is unavailable")
	}
	return peer.SCTP().Transport(), nil
}
